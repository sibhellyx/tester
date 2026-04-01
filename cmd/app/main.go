package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/sibhellyx/tester/internal/api"
	"github.com/sibhellyx/tester/internal/api/handlers"
	"github.com/sibhellyx/tester/internal/core/chaos"
	"github.com/sibhellyx/tester/internal/core/coordinator"
	"github.com/sibhellyx/tester/internal/core/load"
	"github.com/sibhellyx/tester/internal/database"
	"github.com/sibhellyx/tester/internal/processor"
	"github.com/sibhellyx/tester/internal/service"
	"github.com/sibhellyx/tester/pkg/config"
	"github.com/sibhellyx/tester/pkg/docker"
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

	// Инициализация database.
	db, err := sql.Open("postgres", currentCfg.DbConfig.Dsn)
	if err != nil {
		log.Error("sql.Open:", slog.String("error", err.Error()))
	}
	defer db.Close()

	ctx := context.Background()

	// Применение миграций — до старта HTTP-сервера.
	err = database.RunMigrations(ctx, db, log)
	if err != nil {
		log.Error("Failed to run migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log.Info("Starting service", slog.String("port", currentCfg.App.Port))

	// Инициализация движка для нагрузочного тестирования.
	attacker := load.NewAttacker(10 * time.Second)
	loadEngine := load.NewEngine(log, attacker)

	// Инициализаци движка для стрессового тестирования.
	var chaosEngine *chaos.Engine
	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		log.Warn("Failed to connect to Docker. Chaos Engine disabled.", slog.String("error", err.Error()))
	} else {
		chaosEngine = chaos.NewEngine(log, dockerClient)
		defer dockerClient.Close()
		log.Info("Chaos Engine initialized successfully")
	}

	// Инициализация координатора для тестирования.
	coordinator := coordinator.NewCoordinator(log, loadEngine, chaosEngine)

	// Инициализация repository для управления сценариями.
	scenarioRepository := database.NewScenarioRepository(log, db)
	// Инициализация сервиса для управления сценариями.
	scenarioService := service.NewTestManagementService(log, scenarioRepository, dockerClient)
	// Инициализация handler's для обработки запросов связанных со сценариями.
	scenarioHandler := handlers.NewScenarioHandler(log, scenarioService)
	// Инициализация репозитория для запуска и хранения выполнения тестов.
	runRepository := database.NewTestRunRepository(log, db)
	// Инициализация сервиса для запуска и выполнения тестов.
	runService := service.NewTestRunService(log, coordinator, runRepository, scenarioRepository, dockerClient)
	// Инициализация handler для запуска тестов и управления.
	runHandler := handlers.NewTestRunHandler(log, runService)
	// Инициализация процессора для обработки результатов.
	resultProcessor := processor.NewResultProcessor()
	// Инициализация репозитория для получения результатов тестирования.
	resultRepository := database.NewTestResultsRepository(log, db)
	// Инициализация сервиса получения результатов тестирования.
	resultService := service.NewTestResultsService(log, resultRepository, resultProcessor, currentCfg.App.Dir)
	// Инициализация handler's для обработки результатов.
	resultHandler := handlers.NewTestResultsHandler(log, resultService)
	// Инициализация роутера.
	router := api.NewRouter(scenarioHandler, runHandler, resultHandler)
	// Создания и запуск веб-сервера.
	server := &http.Server{
		Addr:    currentCfg.App.Port,
		Handler: router.SetupRoutes(log),
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
