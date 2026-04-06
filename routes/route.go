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
		api.POST("register", controllers.Register)
		api.POST("login", controllers.Login)
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
		protected.POST("/cr/attachments/upload", controllers.UploadCRAttachments)
		protected.POST("/cr", controllers.CreateCR)
		protected.GET("/cr", controllers.GetCRs)
		protected.GET("/cr/:id", controllers.GetCRByID)
		protected.PUT("/cr/:id", controllers.UpdateCR)
		protected.DELETE("/cr/:id", controllers.DeleteCR)
	}

	r.GET("/oke", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"message": "Oke"})
	})

	r.Static("/uploads", "./uploads")

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
