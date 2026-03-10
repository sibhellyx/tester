package chaos

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// ContainerInfo - информация о контейнере доступном для chaos-тестирования.
type ContainerInfo struct {
	ID     string
	Name   string
	Image  string
	Status string
}

// DockerClient - структура обертка, для взаимодействия с контейнерами.
type DockerClient struct {
	client *client.Client
}

// NewDockerClient - функция инициализации клиента для работы с Docker Sdk.
func NewDockerClient() (*DockerClient, error) {
	client, err := client.New(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return &DockerClient{client: client}, nil
}

// Close - функция закрывающая соединение с Docker.
func (c *DockerClient) Close() error {
	return c.client.Close()
}

// StartContainer - функция запускающая контейнер.
func (c *DockerClient) StartContainer(ctx context.Context, containerID string) error {
	_, err := c.client.ContainerStart(ctx, containerID, client.ContainerStartOptions{})
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
	_, err := c.client.ContainerStop(ctx, containerID, options)
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
	execCreateResp, err := c.client.ExecCreate(ctx, containerID, config)
	if err != nil {
		return fmt.Errorf("failed to create exec configuration: %w", err)
	}
	// Запуск выполнения и подключения к потокам (Exec Attach).
	response, err := c.client.ExecAttach(ctx, execCreateResp.ID, client.ExecAttachOptions{})
	if err != nil {
		return fmt.Errorf("failed to attach to exec process: %w", err)
	}
	defer response.Close()
	// Читаем выполнение команды в пустоту.
	_, err = io.Copy(io.Discard, response.Reader)
	if err != nil {
		return fmt.Errorf("error reading exec output: %w", err)
	}

	inspectResponse, err := c.client.ExecInspect(ctx, execCreateResp.ID, client.ExecInspectOptions{})
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
	_, err := c.client.ContainerUpdate(ctx, containerID, updateConfig)
	if err != nil {
		return fmt.Errorf("failed to update resources for %s: %w", containerID, err)
	}

	return nil
}

// ListContainers возвращает список запущенных контейнеров.
// Только running-контейнеры подходят для воспроизведения сбоев —
// остановленные нельзя атаковать сетевыми командами или лимитами ресурсов.
func (c *DockerClient) ListContainers(ctx context.Context) ([]ContainerInfo, error) {
	containers, err := c.client.ContainerList(ctx, client.ContainerListOptions{
		All: false, // только running, не stopped/paused
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	result := make([]ContainerInfo, 0, len(containers.Items))
	for _, ct := range containers.Items {
		name := ct.ID[:12] // fallback если имён нет
		if len(ct.Names) > 0 {
			// Docker возвращает имена с ведущим "/", обрезаем.
			name = strings.TrimPrefix(ct.Names[0], "/")
		}
		result = append(result, ContainerInfo{
			ID:     ct.ID[:12], // короткий ID — удобнее для передачи в chaos events
			Name:   name,
			Image:  ct.Image,
			Status: ct.Status,
		})
	}
	return result, nil
}
