package controllers

import (
	"MonCR/config"
	"MonCR/models"
	"MonCR/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Register(c *gin.Context) {
	var request struct {
		Fullname string `json:"fullname" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	db, err := config.DBConnect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to database"})
        return
	}

	// binding json
	if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error":   "Invalid input data",
            "details": err.Error(),
        })
        return
    }

	// cek email existing
	var existingUser models.Users
	if err := db.Where("email = ?", strings.ToLower(request.Email)).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
            "success": false,
            "error":   "Email already registered",
        })
        return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{
            "success": false,
            "error":   "Failed to check existing user",
        })
        return
	}

	hashedpass, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// object users
	user := models.Users{
		Fullname: request.Fullname,
		Email: request.Email,
		PasswordHash: string(hashedpass),
	}

	// create user in db
	if err := db.Create(&user).Error ;err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
            "success": false,
            "error":   "Failed to create user",
            "details": err.Error(),
        })
        return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success" : true,
		"message" : "User registered successfully",
		"data" : gin.H{
			"id" : user.ID,
			"fullname" : user.Fullname,
			"created_at" : user.CreatedAt,
		},
	})
}

func Login(c *gin.Context) {
	var request struct {
		Email string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	// binding json
	if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error":   "Invalid input data",
            "details": err.Error(),
        })
        return
	}

	// connect db
	db, err := config.DBConnect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to database"})
		return
	}

	var user models.Users

	// cari user berdasarkan email
	if err := db.Where("email = ?", request.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid email or password",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Database error",
			})
		}
		return
	}

	// compare password
	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)) ; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error" : "Invalid Username Or Password"})
		return 
	}

	// generate jwt
	token, err := utils.GenerateJWT(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error" : "Failed to Generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "Login successful",
        "data": gin.H{
            "id":            user.ID,
            "full_name":     user.Fullname,
        },
        "token": token,
    })
}