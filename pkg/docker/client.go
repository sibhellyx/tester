package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/sibhellyx/tester/internal/models"
)

// DockerClient - структура обертка, для взаимодействия с контейнерами.
type DockerClient struct {
	cli *client.Client
}

// NewDockerClient - функция инициализации клиента для работы с Docker Sdk.
func NewDockerClient() (*DockerClient, error) {
	client, err := client.New(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return &DockerClient{cli: client}, nil
}

// Close - функция закрывающая соединение с Docker.
func (c *DockerClient) Close() error {
	return c.cli.Close()
}

// StartContainer - функция запускающая контейнер.
func (c *DockerClient) StartContainer(ctx context.Context, containerID string) error {
	_, err := c.cli.ContainerStart(ctx, containerID, client.ContainerStartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container %s: %w", containerID, err)
	}
	return nil
}

// StopContainer функция останавливающая контейнер.
func (c *DockerClient) StopContainer(ctx context.Context, containerID string, timeout int) error {
	options := client.ContainerStopOptions{
		Timeout: &timeout,
	}
	_, err := c.cli.ContainerStop(ctx, containerID, options)
	if err != nil {
		return fmt.Errorf("failed to stop container %s: %w", containerID, err)
	}
	return nil
}

// ExecCommand выполняет shell-команду внутри контейнера и ждет её завершения.
func (c *DockerClient) ExecCommand(ctx context.Context, containerID string, cmd []string) error {
	// Создание конфигурации запуска (Exec Create).
	config := client.ExecCreateOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
		Privileged:   true,   // Для сетевых команд (tc, iptables).
		User:         "root", // Выполнние команды от root.
	}
	execCreateResp, err := c.cli.ExecCreate(ctx, containerID, config)
	if err != nil {
		return fmt.Errorf("failed to create exec configuration: %w", err)
	}
	// Запуск выполнения и подключения к потокам (Exec Attach).
	response, err := c.cli.ExecAttach(ctx, execCreateResp.ID, client.ExecAttachOptions{})
	if err != nil {
		return fmt.Errorf("failed to attach to exec process: %w", err)
	}
	defer response.Close()
	// Читаем выполнение команды в пустоту.
	_, err = io.Copy(io.Discard, response.Reader)
	if err != nil {
		return fmt.Errorf("error reading exec output: %w", err)
	}

	inspectResponse, err := c.cli.ExecInspect(ctx, execCreateResp.ID, client.ExecInspectOptions{})
	if err != nil {
		return fmt.Errorf("failed to inspect exec result: %w", err)
	}

	if inspectResponse.ExitCode != 0 {
		return fmt.Errorf("command failed with exit code %d", inspectResponse.ExitCode)
	}

	return nil
}

// UpdateResources меняет лимиты CPU/Memory на лету.
func (c *DockerClient) UpdateResources(ctx context.Context, containerID string, cpuQuota int64, memoryBytes int64) error {
	// Формируем структуру ресурсов.
	resources := container.Resources{
		CPUQuota: cpuQuota,
		Memory:   memoryBytes,
	}

	// Опции обновления.
	updateConfig := client.ContainerUpdateOptions{
		Resources: &resources,
	}

	// Вызываем API.
	_, err := c.cli.ContainerUpdate(ctx, containerID, updateConfig)
	if err != nil {
		return fmt.Errorf("failed to update resources for %s: %w", containerID, err)
	}

	return nil
}

// GetStats возвращает статистику контейнера по загруженности.
func (c *DockerClient) GetStats(ctx context.Context, containerID string) (*models.ContainerStats, error) {
	// false = один снимок, не бесконечный stream.
	resp, err := c.cli.ContainerStats(ctx, containerID, client.ContainerStatsOptions{
		Stream: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get stats for %s: %w", containerID, err)
	}
	defer resp.Body.Close()

	var stats container.StatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	// Формула CPU из официальной документации Docker.
	// PreCPUStats — предыдущий снимок, CPUStats — текущий.
	// Дельта показывает сколько CPU-времени потратил контейнер между снимками.
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	onlineCPUs := float64(stats.CPUStats.OnlineCPUs)

	var cpuPercent float64
	if systemDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / systemDelta) * onlineCPUs * 100.0
	}

	// Формула RAM.
	// Нужно вычесть page cache — Docker включает его в Usage,
	// но это не реальное потребление приложения.
	cache := stats.MemoryStats.Stats["cache"]
	usedMemory := float64(stats.MemoryStats.Usage - cache)
	memPercent := (usedMemory / float64(stats.MemoryStats.Limit)) * 100.0

	return &models.ContainerStats{
		CPUPercent: cpuPercent,
		MemPercent: memPercent,
	}, nil
}

// ListContainers возвращает список запущенных контейнеров.
// Только running-контейнеры подходят для воспроизведения сбоев —
// остановленные нельзя атаковать сетевыми командами или лимитами ресурсов.
// Контейнеры собственной инфраструктуры (помечены лейблом tester.internal=true) исключаются.
func (c *DockerClient) ListContainers(ctx context.Context) ([]models.ContainerInfo, error) {
	containers, err := c.cli.ContainerList(ctx, client.ContainerListOptions{
		All: false, // только running, не stopped/paused
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	result := make([]models.ContainerInfo, 0, len(containers.Items))
	for _, ct := range containers.Items {
		// Исключаем контейнеры собственной инфраструктуры сервиса.
		if ct.Labels["tester.internal"] == "true" {
			continue
		}

		name := ct.ID[:12] // fallback если имён нет
		if len(ct.Names) > 0 {
			// Docker возвращает имена с ведущим "/", обрезаем.
			name = strings.TrimPrefix(ct.Names[0], "/")
		}
		result = append(result, models.ContainerInfo{
			ID:     ct.ID[:12], // короткий ID — удобнее для передачи в chaos events
			Name:   name,
			Image:  ct.Image,
			Status: ct.Status,
		})
	}
	return result, nil
}
