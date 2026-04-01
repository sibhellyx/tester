package api

import (
	"log/slog"

	cors "github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/sibhellyx/tester/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type ScenarioHandler interface {
	CreateScenario(c *gin.Context)
	GetScenario(c *gin.Context)
	ListScenarios(c *gin.Context)
	DeleteScenario(c *gin.Context)
	UpdateScenario(c *gin.Context)
	ListContainers(c *gin.Context)
}

type RunHandler interface {
	StartTest(c *gin.Context)  // POST /scenarios/:id/runs
	StopTest(c *gin.Context)   // POST /runs/:run_id/stop
	GetStatus(c *gin.Context)  // GET  /runs/:run_id
	ListRuns(c *gin.Context)   // GET  /scenarios/:id/runs
	GetSummary(c *gin.Context) // GET  /runs/:run_id/summary
}

type ResultsHandler interface {
	GetReport(c *gin.Context)         // GET /runs/:run_id/report
	GetChartData(c *gin.Context)      // GET /runs/:run_id/charts
	GetReportFile(c *gin.Context)     // GET /runs/:run_id/report/download
	GetListWithStatus(c *gin.Context) // GET /runs
}

// Router структура для хранения обработчиков.
type Router struct {
	router          *gin.Engine
	scenarioHandler ScenarioHandler
	runHandler      RunHandler
	resultsHandler  ResultsHandler
}

// NewRouter - создает роутер.
func NewRouter(
	scenarioHandler ScenarioHandler,
	runHandler RunHandler,
	resultsHandler ResultsHandler,
) *Router {
	return &Router{
		router:          gin.Default(),
		scenarioHandler: scenarioHandler,
		runHandler:      runHandler,
		resultsHandler:  resultsHandler,
	}
}

func (r *Router) SetupRoutes(logger *slog.Logger) *gin.Engine {
	r.router.Use(gin.Recovery())
	// for allow requests from frontend
	r.router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:5173"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders: []string{"Content-Type"},
	}))

	r.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.router.Group("/api/v1")
	{
		// --- Сценарии (CRUD) ---
		scenarios := v1.Group("/scenarios")
		{
			scenarios.POST("", r.scenarioHandler.CreateScenario)
			scenarios.GET("", r.scenarioHandler.ListScenarios)
			scenarios.GET("/:id", r.scenarioHandler.GetScenario)
			scenarios.PUT("/:id", r.scenarioHandler.UpdateScenario)
			scenarios.DELETE("/:id", r.scenarioHandler.DeleteScenario)

			// Запуски, вложенные в сценарий.
			scenarios.POST("/:id/runs", r.runHandler.StartTest)
			scenarios.GET("/:id/runs", r.runHandler.ListRuns)
		}

		// --- Управление запусками и результаты ---
		runs := v1.Group("/runs")
		{
			// Управление (TestRunHandler).
			runs.GET("", r.resultsHandler.GetListWithStatus)      // дашборд всех запусков
			runs.GET("/:run_id", r.runHandler.GetStatus)          // статус конкретного
			runs.POST("/:run_id/stop", r.runHandler.StopTest)     // остановить
			runs.GET("/:run_id/summary", r.runHandler.GetSummary) // быстрая сводка из БД

			// Результаты (TestResultsHandler).
			runs.GET("/:run_id/report", r.resultsHandler.GetReport)              // полный отчёт (JSON)
			runs.GET("/:run_id/charts", r.resultsHandler.GetChartData)           // только графики
			runs.GET("/:run_id/report/download", r.resultsHandler.GetReportFile) // скачать CSV
		}
		// Для получения доступных контейнеров.
		chaos := v1.Group("/chaos")
		{
			chaos.GET("/containers", r.scenarioHandler.ListContainers)
		}
	}

	return r.router
}
