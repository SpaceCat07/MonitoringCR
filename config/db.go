package config

import (
	"MonCR/models"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// singleton DB instance with connection pool
var (
	dbInstance *gorm.DB
	dbOnce     sync.Once
	dbErr      error
)

func DBConnect() (*gorm.DB, error) {
	dbOnce.Do(func() {
		_ = godotenv.Load()

		host := os.Getenv("DB_HOST")
		port := os.Getenv("DB_PORT")
		user := os.Getenv("DB_USER")
		password := os.Getenv("DB_PASSWORD")
		dbname := os.Getenv("DB_NAME")

		dsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, dbname,
		)

		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			dbErr = fmt.Errorf("failed to connect to database: %w", err)
			return
		}

		// Configure connection pool — prevents "too many clients"
		sqlDB, err := db.DB()
		if err != nil {
			dbErr = fmt.Errorf("failed to get sql.DB: %w", err)
			return
		}
		sqlDB.SetMaxOpenConns(10)               // max 10 open connections
		sqlDB.SetMaxIdleConns(5)                // keep 5 idle connections ready
		sqlDB.SetConnMaxLifetime(5 * time.Minute) // recycle connections every 5 min
		sqlDB.SetConnMaxIdleTime(2 * time.Minute) // close idle after 2 min

		db.AutoMigrate(
			&models.Users{},
			&models.ChangeRequest{},
			&models.Activity{},
			&models.SubTask{},
		)

		var count int64
		db.Model(&models.Users{}).Count(&count)
		if count == 0 {
			seedUsers := []struct {
				Fullname string
				Email    string
				Password string
				Role     string
			}{
				{"Super Admin", "admin@mail.com", "admin123", "Admin"},
				{"Manager PLN", "manager@mail.com", "manager123", "Manager"},
				{"PIC Utama", "pic@mail.com", "pic123", "PIC"},
				{"Collaborator A", "collaborator@mail.com", "collaborator123", "Collaborator"},
			}

			for _, u := range seedUsers {
				hashedPass, _ := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
				db.Create(&models.Users{
					Fullname:     u.Fullname,
					Email:        u.Email,
					PasswordHash: string(hashedPass),
					Role:         u.Role,
				})
				fmt.Printf("Seeded user: %s / %s (Role: %s)\n", u.Email, u.Password, u.Role)
			}
		}

		dbInstance = db
	})

	return dbInstance, dbErr
}
