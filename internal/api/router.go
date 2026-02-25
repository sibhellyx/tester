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

// Router структура для хранения обработчиков.
type Router struct {
	router          *gin.Engine
	scenarioHandler ScenarioHandlerInterface
	runHandler      TestRunHandlerInterface
}

// NewRouter - создает роутер.
func NewRouter(scenarioHandler ScenarioHandlerInterface, runHandler TestRunHandlerInterface) *Router {
	return &Router{
		router:          gin.Default(),
		scenarioHandler: scenarioHandler,
		runHandler:      runHandler,
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

			// --- Запуски, вложенные в сценарий ---
			scenarios.POST("/:id/runs", r.runHandler.StartTest) // запустить тест
			scenarios.GET("/:id/runs", r.runHandler.ListRuns)   // история запусков
		}

		// --- Управление конкретным запуском ---
		runs := v1.Group("/runs")
		{
			runs.GET("/:run_id", r.runHandler.GetStatus)          // статус
			runs.POST("/:run_id/stop", r.runHandler.StopTest)     // остановить
			runs.GET("/:run_id/summary", r.runHandler.GetSummary) // итоговая статистика
		}
	}

	return r.router
}
