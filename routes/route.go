package routes

import (
	"MonCR/controllers"
	"MonCR/middleware"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func InitRoutes() *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOriginFunc:  func(origin string) bool { return true },
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	api := r.Group("/api")
	{
		// auth
		api.POST("login", controllers.Login)
		api.GET("roles", controllers.GetRoleOptions)
	}

	// this rate limiter only works in single server only
	// the limiter data clients are stored in server memory
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware(), middleware.RateLimitByUser())
	{
		// endpoint with middleware
		protected.GET("/cek", func(ctx *gin.Context) {
			ctx.JSON(200, gin.H{"message": "keren banget jwt nya udah bisa"})
		})

		// SPR endpoints
		protected.GET("/cr/options", controllers.GetCROptions)
		protected.GET("/cr/charts", controllers.GetCRCharts)
		protected.GET("/cr/export", controllers.ExportCRsPDF)
		protected.POST("/cr/attachments/upload", controllers.UploadCRAttachments)
		protected.POST("/cr", controllers.CreateCR)
		protected.GET("/cr", controllers.GetCRs)
		protected.GET("/cr/:id", controllers.GetCRByID)
		protected.PUT("/cr/:id", controllers.UpdateCR)
		protected.DELETE("/cr/:id", controllers.DeleteCR)

		// Subtask endpoints
		protected.POST("/subtasks", controllers.CreateSubtask)
		protected.GET("/subtasks", controllers.GetSubtasks)
		protected.GET("/subtasks/:id", controllers.GetSubtaskByID)
		protected.PUT("/subtasks/:id", controllers.UpdateSubtask)
		protected.DELETE("/subtasks/:id", controllers.DeleteSubtask)

		// Activity endpoints
		protected.POST("/activities", controllers.CreateActivity)
		protected.GET("/activities", controllers.GetActivities)
		protected.GET("/activities/:id", controllers.GetActivityByID)
		protected.PUT("/activities/:id", controllers.UpdateActivity)
		protected.DELETE("/activities/:id", controllers.DeleteActivity)

		// User endpoints (Admin only)
		adminGroup := protected.Group("/users")
		adminGroup.Use(middleware.RoleMiddleware("Admin"))
		{
			adminGroup.POST("", controllers.CreateUser)
			adminGroup.GET("", controllers.GetUsers)
			adminGroup.GET("/:id", controllers.GetUserByID)
			adminGroup.PUT("/:id", controllers.UpdateUser)
			adminGroup.DELETE("/:id", controllers.DeleteUser)
		}
	}

	r.GET("/oke", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"message": "Oke"})
	})

	r.Static("/uploads", "./uploads")

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
