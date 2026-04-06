package controllers

import (
	"MonCR/config"
	"MonCR/models"
	"MonCR/utils"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const defaultCRStatus = "ISSUED"

var categoryOptions = []string{
	"FLOW",
	"REPORT",
	"INTERFACE",
	"CONVERTION",
	"ENHANCEMENT",
	"FORM",
	"CONFIGURATION",
	"AUTORIZATION",
}

var statusOptions = []string{
	"ISSUED",
	"RELEASE",
	"IN_PROGRESS",
	"COMPLETE",
	"CANCEL",
}

type createCRRequest struct {
	Title          string    `json:"title" binding:"required,min=3,max=255"`
	Description    string    `json:"description" binding:"required,min=3"`
	Category       string    `json:"category" binding:"required,oneof=FLOW REPORT INTERFACE CONVERTION ENHANCEMENT FORM CONFIGURATION AUTORIZATION"`
	Status         string    `json:"status" binding:"omitempty,oneof=ISSUED RELEASE IN_PROGRESS COMPLETE CANCEL"`
	ReleaseDate    time.Time `json:"release_date" binding:"required"`
	EndDate        time.Time `json:"end_date" binding:"required"`
	FileAttachment []string  `json:"file_attachment" binding:"required,min=1,dive,required"`
}

type updateCRRequest struct {
	Title          string    `json:"title" binding:"required,min=3,max=255"`
	Description    string    `json:"description" binding:"required,min=3"`
	Category       string    `json:"category" binding:"required,oneof=FLOW REPORT INTERFACE CONVERTION ENHANCEMENT FORM CONFIGURATION AUTORIZATION"`
	Status         string    `json:"status" binding:"required,oneof=ISSUED RELEASE IN_PROGRESS COMPLETE CANCEL"`
	ReleaseDate    time.Time `json:"release_date" binding:"required"`
	EndDate        time.Time `json:"end_date" binding:"required"`
	FileAttachment []string  `json:"file_attachment" binding:"required,min=1,dive,required"`
}

// GetCROptions godoc
// @Summary Get category and status options
// @Description Get available options for CATEGORY and STATUS fields when creating/updating change request.
// @Tags CR
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/spr/options [get]
func GetCROptions(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"category_options": categoryOptions,
			"status_options":   statusOptions,
		},
	})
}

func connectDB(c *gin.Context) (*gorm.DB, bool) {
	db, err := config.DBConnect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to database"})
		return nil, false
	}
	return db, true
}

func getClaims(c *gin.Context) (*utils.Claims, bool) {
	claimsValue, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Unauthorized",
		})
		return nil, false
	}

	claims, ok := claimsValue.(*utils.Claims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Invalid token claims",
		})
		return nil, false
	}

	return claims, true
}

func parseCRID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid Change Request id",
		})
		return 0, false
	}
	return uint(id), true
}

func findCRByID(db *gorm.DB, id uint) (*models.ChangeRequest, error) {
	var cr models.ChangeRequest
	if err := db.Preload("Creator").First(&cr, id).Error; err != nil {
		return nil, err
	}
	return &cr, nil
}

// CreateCR godoc
// @Summary Create Change Request
// @Description Create a new change request.
// @Tags CR
// @Accept json
// @Produce json
// @Param payload body createCRRequest true "Create CR payload"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/spr [post]
func CreateCR(c *gin.Context) {
	var request createCRRequest

	db, ok := connectDB(c)
	if !ok {
		return
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid input data",
			"details": err.Error(),
		})
		return
	}

	claims, ok := getClaims(c)
	if !ok {
		return
	}

	cr := models.ChangeRequest{
		Title:          request.Title,
		Description:    request.Description,
		Category:       request.Category,
		ReleaseDate:    request.ReleaseDate,
		EndDate:        request.EndDate,
		FileAttachment: request.FileAttachment,
		CreatedBy:      claims.UserID,
	}

	if request.Status != "" {
		cr.Status = request.Status
	} else {
		cr.Status = defaultCRStatus
	}

	if err := db.Create(&cr).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create Change Request",
			"details": err.Error(),
		})
		return
	}

	db.Preload("Creator").First(&cr, cr.ID)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Change Request created successfully",
		"data":    cr,
	})
}

// GetCRs godoc
// @Summary List Change Requests
// @Description Get paginated list of change requests.
// @Tags CR
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/spr [get]
func GetCRs(c *gin.Context) {
	db, ok := connectDB(c)
	if !ok {
		return
	}

	pagination := utils.ParsePagination(c, 10, 100)

	var total int64
	db.Model(&models.ChangeRequest{}).Count(&total)

	var crs []models.ChangeRequest
	db.Preload("Creator").
		Order("id DESC").
		Offset(pagination.Offset).
		Limit(pagination.Limit).
		Find(&crs)

	paginationMeta := utils.BuildPaginationMeta(pagination.Offset, pagination.Limit, len(crs), total)

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"data":       crs,
		"pagination": paginationMeta,
	})
}

// GetCRByID godoc
// @Summary Get Change Request by ID
// @Description Retrieve one change request by id.
// @Tags CR
// @Produce json
// @Param id path int true "CR ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/spr/{id} [get]
func GetCRByID(c *gin.Context) {
	db, ok := connectDB(c)
	if !ok {
		return
	}

	id, ok := parseCRID(c)
	if !ok {
		return
	}

	cr, err := findCRByID(db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Change Request not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve Change Request",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cr,
	})
}

// UpdateCR godoc
// @Summary Update Change Request
// @Description Fully update change request by id.
// @Tags CR
// @Accept json
// @Produce json
// @Param id path int true "CR ID"
// @Param payload body updateCRRequest true "Update CR payload"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/spr/{id} [put]
func UpdateCR(c *gin.Context) {
	db, ok := connectDB(c)
	if !ok {
		return
	}

	id, ok := parseCRID(c)
	if !ok {
		return
	}

	_, err := findCRByID(db, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Change Request not found",
		})
		return
	}

	var request updateCRRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid input data",
			"details": err.Error(),
		})
		return
	}

	updates := models.ChangeRequest{
		Title:          request.Title,
		Description:    request.Description,
		Category:       request.Category,
		Status:         request.Status,
		ReleaseDate:    request.ReleaseDate,
		EndDate:        request.EndDate,
		FileAttachment: request.FileAttachment,
	}

	if err := db.Model(&models.ChangeRequest{}).Where("id = ?", id).Updates(&updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update Change Request",
		})
		return
	}

	updatedCR, _ := findCRByID(db, id)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Change Request updated successfully",
		"data":    updatedCR,
	})
}

// DeleteCR godoc
// @Summary Delete Change Request
// @Description Delete change request by id.
// @Tags CR
// @Produce json
// @Param id path int true "CR ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/spr/{id} [delete]
func DeleteCR(c *gin.Context) {
	db, ok := connectDB(c)
	if !ok {
		return
	}

	id, ok := parseCRID(c)
	if !ok {
		return
	}

	cr, err := findCRByID(db, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Change Request not found",
		})
		return
	}

	if err := db.Delete(cr).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete Change Request",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Change Request deleted successfully",
	})
}
