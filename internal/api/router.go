package api

import (
	"log/slog"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-gonic/gin"
	_ "github.com/sibhellyx/tester/docs"
)

type ScenarioHandlerInterface interface {
	CreateScenario(c *gin.Context)
	GetScenario(c *gin.Context)
	ListScenarios(c *gin.Context)
	DeleteScenario(c *gin.Context)
	UpdateScenario(c *gin.Context)
}

type TestRunHandlerInterface interface {
	StartTest(c *gin.Context)  // POST /scenarios/:id/runs
	StopTest(c *gin.Context)   // POST /runs/:run_id/stop
	GetStatus(c *gin.Context)  // GET  /runs/:run_id
	ListRuns(c *gin.Context)   // GET  /scenarios/:id/runs
	GetSummary(c *gin.Context) // GET  /runs/:run_id/summary
}

type TestResultsHandlerInterface interface {
	GetReport(c *gin.Context)         // GET /runs/:run_id/report
	GetChartData(c *gin.Context)      // GET /runs/:run_id/charts
	GetReportFile(c *gin.Context)     // GET /runs/:run_id/report/download
	GetListWithStatus(c *gin.Context) // GET /runs
}

// Router структура для хранения обработчиков.
type Router struct {
	router          *gin.Engine
	scenarioHandler ScenarioHandlerInterface
	runHandler      TestRunHandlerInterface
	resultsHandler  TestResultsHandlerInterface
}

// NewRouter - создает роутер.
func NewRouter(
	scenarioHandler ScenarioHandlerInterface,
	runHandler TestRunHandlerInterface,
	resultsHandler TestResultsHandlerInterface,
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
	}

	return r.router
}
