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
	UpdateScenario(с *gin.Context)
}

// Router структура для хранения обработчиков для управления тестированием.
type Router struct {
	router          *gin.Engine
	scenarioHandler ScenarioHandlerInterface
}

// NewRouter - создает роутер для управления сервисом.
func NewRouter(scenarioHandler ScenarioHandlerInterface) *Router {
	return &Router{
		router:          gin.Default(),
		scenarioHandler: scenarioHandler,
	}
}

func (r *Router) SetupRoutes(logger *slog.Logger) *gin.Engine {
	r.router.Use(gin.Recovery())

	// Swagger UI будет доступен по адресу /swagger/index.html.
	r.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.router.Group("/api/v1")
	{
		scenarios := v1.Group("/scenarios")
		{
			// CRUD операции
			scenarios.POST("", r.scenarioHandler.CreateScenario)       // Создать
			scenarios.GET("", r.scenarioHandler.ListScenarios)         // Получить список
			scenarios.GET("/:id", r.scenarioHandler.GetScenario)       // Получить один по ID
			scenarios.PUT("/:id", r.scenarioHandler.UpdateScenario)    // Обновить (НОВЫЙ)
			scenarios.DELETE("/:id", r.scenarioHandler.DeleteScenario) // Удалить
		}

		// tests := v1.Group("/tests") { ... }
	}

	return r.router
}
