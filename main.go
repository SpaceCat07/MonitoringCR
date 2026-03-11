package main

import (
	"MonCR/config"
	"MonCR/routes"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

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