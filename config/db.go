package config

import (
	"MonCR/models"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func DBConnect() (*gorm.DB, error) {
	err := godotenv.Load()

	if err != nil {
		fmt.Println("Warning: .env file not found, using system environment variables")
	}

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db.AutoMigrate(
		&models.Users{},
		&models.ChangeRequest{},
		&models.Activity{},
		&models.SubTask{},
	)

	var count int64
	db.Model(&models.Users{}).Count(&count)
	if count == 0 {
		hashedPass, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		db.Create(&models.Users{
			Fullname:     "Super Admin",
			Email:        "admin@mail.com",
			PasswordHash: string(hashedPass),
			Role:         "Admin",
		})
		fmt.Println("Seeded initial Admin user: admin@mail.com / admin123")
	}

	return db, nil
}
