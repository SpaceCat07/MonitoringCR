package main

import (
	"MonCR/config"
	_ "MonCR/docs"
	"MonCR/routes"
	"MonCR/utils"

	"github.com/joho/godotenv"
)

// @title MonitoringCR API
// @version 1.0
// @description API documentation for MonitoringCR.
// @BasePath /
// @securityDefinitions.apiKey BearerAuth
// @in header
// @name Authorization
// @description Bearer token authentication

func main() {
	godotenv.Load()

	go utils.CleanupClients()

	// connect to postgres
	postsql, err := config.DBConnect()
	if err != nil {
		panic("Failed to get database connection: " + err.Error())
	}

	// get underlying sql DB
	sqlDB, err := postsql.DB()
	if err != nil {
		panic("Failed to get underlying database connection: " + err.Error())
	}

	defer sqlDB.Close()

	router := routes.InitRoutes()

	router.Run()
}
