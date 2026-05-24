package controllers

import (
	"MonCR/config"
	"MonCR/models"
	"MonCR/utils"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
)

var roleOptions = []string{
	"Admin",
	"Manager",
	"PIC",
	"Collaborator",
}

type userRequest struct {
	Fullname  string `json:"fullname" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password"`
	Role      string `json:"role" binding:"required,oneof=Admin Manager PIC Collaborator"`
	ParentPIC *uint  `json:"parent_pic"`
}

// GetRoleOptions godoc
// @Summary Get role options
// @Description Get available options for Role field.
// @Tags Users
// @Produce json
// @Success 200 {object} utils.APIResponse
// @Router /api/roles [get]
func GetRoleOptions(c *gin.Context) {
	c.JSON(http.StatusOK, utils.FormatResponse("Role options retrieved", http.StatusOK, "success", roleOptions))
}

// CreateUser godoc
// @Summary Create a new user
// @Tags Users
// @Accept json
// @Produce json
// @Param payload body userRequest true "User data"
// @Success 201 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 409 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /api/users [post]
// @Security BearerAuth
func CreateUser(c *gin.Context) {
	var request userRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, utils.FormatResponse(friendlyValidationError(err), http.StatusBadRequest, "error", nil))
		return
	}

	db, err := config.DBConnect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to connect to database", http.StatusInternalServerError, "error", nil))
		return
	}

	// Check if email already exists
	var existingUser models.Users
	if err := db.Where("email = ?", strings.ToLower(request.Email)).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, utils.FormatResponse("Email already registered", http.StatusConflict, "error", nil))
		return
	}

	// Password is required for creation
	if request.Password == "" {
		c.JSON(http.StatusBadRequest, utils.FormatResponse("Password is required", http.StatusBadRequest, "error", nil))
		return
	}

	hashedpass, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to hash password", http.StatusInternalServerError, "error", nil))
		return
	}

	user := models.Users{
		Fullname:     request.Fullname,
		Email:        strings.ToLower(request.Email),
		PasswordHash: string(hashedpass),
		Role:         request.Role,
		ParentPIC:    request.ParentPIC,
	}

	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to create user", http.StatusInternalServerError, "error", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, utils.FormatResponse("User created successfully", http.StatusCreated, "success", gin.H{
		"id":         user.ID,
		"fullname":   user.Fullname,
		"email":      user.Email,
		"role":       user.Role,
		"parent_pic": user.ParentPIC,
		"created_at": user.CreatedAt,
	}))
}

// GetUsers godoc
// @Summary Get all users
// @Tags Users
// @Produce json
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /api/users [get]
// @Security BearerAuth
func GetUsers(c *gin.Context) {
	db, err := config.DBConnect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to connect to database", http.StatusInternalServerError, "error", nil))
		return
	}

	var users []models.Users
	// Preload parent pic detail
	if err := db.Preload("Parent").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to fetch users", http.StatusInternalServerError, "error", err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.FormatResponse("Users retrieved successfully", http.StatusOK, "success", users))
}

// GetUserByID godoc
// @Summary Get user by ID
// @Tags Users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /api/users/{id} [get]
// @Security BearerAuth
func GetUserByID(c *gin.Context) {
	db, err := config.DBConnect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to connect to database", http.StatusInternalServerError, "error", nil))
		return
	}

	id := c.Param("id")
	var user models.Users

	if err := db.Preload("Parent").First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, utils.FormatResponse("User not found", http.StatusNotFound, "error", nil))
		return
	}

	c.JSON(http.StatusOK, utils.FormatResponse("User retrieved successfully", http.StatusOK, "success", user))
}

// UpdateUser godoc
// @Summary Update user by ID
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param payload body userRequest true "User data"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /api/users/{id} [put]
// @Security BearerAuth
func UpdateUser(c *gin.Context) {
	db, err := config.DBConnect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to connect to database", http.StatusInternalServerError, "error", nil))
		return
	}

	id := c.Param("id")
	var user models.Users

	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, utils.FormatResponse("User not found", http.StatusNotFound, "error", nil))
		return
	}

	var request userRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, utils.FormatResponse(friendlyValidationError(err), http.StatusBadRequest, "error", nil))
		return
	}

	// Update fields
	user.Fullname = request.Fullname
	user.Email = strings.ToLower(request.Email)
	user.Role = request.Role
	user.ParentPIC = request.ParentPIC

	// Update password if provided
	if request.Password != "" {
		hashedpass, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to hash password", http.StatusInternalServerError, "error", nil))
			return
		}
		user.PasswordHash = string(hashedpass)
	}

	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to update user", http.StatusInternalServerError, "error", err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.FormatResponse("User updated successfully", http.StatusOK, "success", gin.H{
		"id":         user.ID,
		"fullname":   user.Fullname,
		"email":      user.Email,
		"role":       user.Role,
		"parent_pic": user.ParentPIC,
	}))
}

// DeleteUser godoc
// @Summary Delete user by ID
// @Tags Users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /api/users/{id} [delete]
// @Security BearerAuth
func DeleteUser(c *gin.Context) {
	db, err := config.DBConnect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to connect to database", http.StatusInternalServerError, "error", nil))
		return
	}

	id := c.Param("id")
	var user models.Users

	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, utils.FormatResponse("User not found", http.StatusNotFound, "error", nil))
		return
	}

	// Prevent deleting Admin accounts
	if strings.EqualFold(user.Role, "Admin") {
		c.JSON(http.StatusForbidden, utils.FormatResponse("Admin accounts cannot be deleted", http.StatusForbidden, "error", nil))
		return
	}

	if err := db.Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to delete user", http.StatusInternalServerError, "error", err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.FormatResponse("User deleted successfully", http.StatusOK, "success", nil))
}

// friendlyValidationError converts validator errors into human-readable English messages.
func friendlyValidationError(err error) string {
	var ve validator.ValidationErrors
	if !strings.Contains(fmt.Sprintf("%T", err), "ValidationErrors") {
		// Not a validation error, return generic message
		return "Invalid request body. Please check your input."
	}
	if ok := true; ok {
		_ = ok
	}
	// Try type assertion
	if errs, ok := err.(validator.ValidationErrors); ok {
		ve = errs
	} else {
		return "Invalid request body. Please check your input."
	}
	var msgs []string
	for _, fe := range ve {
		field := strings.ToLower(fe.Field())
		switch fe.Tag() {
		case "required":
			msgs = append(msgs, fmt.Sprintf("'%s' is required.", field))
		case "email":
			msgs = append(msgs, fmt.Sprintf("'%s' must be a valid email address (e.g. user@example.com).", field))
		case "oneof":
			msgs = append(msgs, fmt.Sprintf("'%s' must be one of: %s.", field, fe.Param()))
		case "min":
			msgs = append(msgs, fmt.Sprintf("'%s' must be at least %s characters.", field, fe.Param()))
		case "max":
			msgs = append(msgs, fmt.Sprintf("'%s' must be at most %s characters.", field, fe.Param()))
		default:
			msgs = append(msgs, fmt.Sprintf("'%s' is invalid.", field))
		}
	}
	if len(msgs) == 0 {
		return "Invalid input. Please check your data."
	}
	return strings.Join(msgs, " ")
}


func ReturnCollabByPIC(c *gin.Context)  {
	db, err := config.DBConnect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to connect to database", http.StatusInternalServerError, "error", nil))
		return
	}

	PICID := c.Param("PIC_ID")
	
	var collabs []models.Users
	if err := db.Where("parent_pic = ?", PICID).Find(&collabs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to fetch collaborators", http.StatusInternalServerError, "error", err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.FormatResponse("Collaborators retrieved successfully", http.StatusOK, "success", collabs))
}