package controllers

import (
	"MonCR/config"
	"MonCR/models"
	"MonCR/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)



type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login godoc
// @Summary Login user
// @Description Authenticate user and return JWT token.
// @Tags Auth
// @Accept json
// @Produce json
// @Param payload body loginRequest true "Login payload"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 401 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /api/login [post]
func Login(c *gin.Context) {
	var request loginRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, utils.FormatResponse("Invalid input data", http.StatusBadRequest, "error", err.Error()))
		return
	}

	db, err := config.DBConnect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to connect to database", http.StatusInternalServerError, "error", nil))
		return
	}

	var user models.Users
	if err := db.Where("email = ?", request.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, utils.FormatResponse("Invalid email or password", http.StatusUnauthorized, "error", nil))
		} else {
			c.JSON(http.StatusInternalServerError, utils.FormatResponse("Database error", http.StatusInternalServerError, "error", nil))
		}
		return
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, utils.FormatResponse("Invalid Username Or Password", http.StatusUnauthorized, "error", nil))
		return
	}

	token, err := utils.GenerateJWT(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to Generate token", http.StatusInternalServerError, "error", nil))
		return
	}

	c.JSON(http.StatusOK, utils.FormatResponse("Login successful", http.StatusOK, "success", gin.H{
		"id":        user.ID,
		"full_name": user.Fullname,
		"token":     token,
	}))
}
