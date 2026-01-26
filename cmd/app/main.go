package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sibhellyx/tester/pkg/config"
	"github.com/sibhellyx/tester/pkg/logger"
)

func main() {
	// Инициализация менеджера конфигурации.
	cfgManager, err := config.New("confs", "config")
	if err != nil {
		panic("failed to load config: " + err.Error())
	}
	// Получение текущей конфигурации.
	currentCfg := cfgManager.Get()

	// Инициализация логгера.
	logWrapper := logger.Setup(currentCfg.LogConfig.Level)
	log := logWrapper.Logger

	log.Info("Starting service", slog.String("port", currentCfg.App.Port))

	// Создания и запуск веб-сервера.
	server := &http.Server{
		Addr: currentCfg.App.Port,
		// Handler: nil,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Настройка каналов для Graceful Shutdown.
	// Инициализация канала, который слушает системные сигналы (Ctrl+C, kill).
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Создание канала, для получения обновлений конфигураций.
	configUpdates := cfgManager.Updates()

	log.Info("Application is running. Waiting for signals or config changes...")

	// Обработка событий.
	for {
		select {
		// Произошло обновление конфигурации.
		case newCfg := <-configUpdates:
			log.Info("Configuration updated detected")

			// Обновление уровня логирования.
			if newCfg.LogConfig.Level != currentCfg.LogConfig.Level {
				log.Info("Changing log level",
					slog.String("old", currentCfg.LogConfig.Level),
					slog.String("new", newCfg.LogConfig.Level),
				)
				logWrapper.SetLevel(newCfg.LogConfig.Level)
			}

			// Проверка изменения Включаем/выключаем Debug режим.
			if newCfg.App.Debug != currentCfg.App.Debug {
				log.Info("Debug mode changed", slog.Bool("enabled", newCfg.App.Debug))
			}

			// Обновление текущего конфига.
			currentCfg = newCfg
			log.Info("Current configs", slog.String("Config Port", currentCfg.App.Port), slog.String("Config Db Dsn", currentCfg.DbConfig.Dsn))

		// Обработка сигнала завершения (Ctrl+C).
		case sig := <-quit:
			log.Info("Shutdown signal received", slog.String("signal", sig.String()))

			// Graceful Shutdown сервера, устанавливаем задержку в 5 секунд.
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := server.Shutdown(ctx); err != nil {
				log.Error("Server forced to shutdown", slog.String("error", err.Error()))
			} else {
				log.Info("Server exited properly")
			}
			return
		}
	}
}
