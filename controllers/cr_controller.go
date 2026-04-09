package controllers

import (
	"MonCR/config"
	"MonCR/models"
	"MonCR/utils"
	"errors"
	"net/http"
	"strconv"
	"strings"
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
	"AUTHORIZATION",
}

var moduleOptions = []string{
	"FINANCE",
	"MATERIAL MANAGEMENT",
	"HUMAN RESOURCE",
	"BASIS",
	"ABAP",
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
	PIC            string    `json:"pic" binding:"required,min=2,max=150"`
	Modul          string    `json:"modul" binding:"required,oneof=FINANCE 'MATERIAL MANAGEMENT' 'HUMAN RESOURCE' BASIS ABAP"`
	Category       string    `json:"category" binding:"required,oneof=FLOW REPORT INTERFACE CONVERTION ENHANCEMENT FORM CONFIGURATION AUTHORIZATION"`
	Status         string    `json:"status" binding:"omitempty,oneof=ISSUED RELEASE IN_PROGRESS COMPLETE CANCEL"`
	Keterangan     string    `json:"keterangan"`
	ReleaseDate    time.Time `json:"release_date" binding:"required"`
	EndDate        time.Time `json:"end_date" binding:"required"`
	FileAttachment []string  `json:"file_attachment" binding:"required,min=1,dive,required"`
}

type updateCRRequest struct {
	Title          string    `json:"title" binding:"required,min=3,max=255"`
	Description    string    `json:"description" binding:"required,min=3"`
	PIC            string    `json:"pic" binding:"required,min=2,max=150"`
	Modul          string    `json:"modul" binding:"required,oneof=FINANCE 'MATERIAL MANAGEMENT' 'HUMAN RESOURCE' BASIS ABAP"`
	Category       string    `json:"category" binding:"required,oneof=FLOW REPORT INTERFACE CONVERTION ENHANCEMENT FORM CONFIGURATION AUTHORIZATION"`
	Status         string    `json:"status" binding:"required,oneof=ISSUED RELEASE IN_PROGRESS COMPLETE CANCEL"`
	Keterangan     string    `json:"keterangan"`
	ReleaseDate    time.Time `json:"release_date" binding:"required"`
	EndDate        time.Time `json:"end_date" binding:"required"`
	FileAttachment []string  `json:"file_attachment" binding:"required,min=1,dive,required"`
}

func requiresKeterangan(status string) bool {
	return status == "ISSUED" || status == "CANCEL"
}

func validateKeteranganByStatus(status, keterangan string) error {
	if requiresKeterangan(status) && strings.TrimSpace(keterangan) == "" {
		return errors.New("keterangan is required when status is ISSUED or CANCEL")
	}

	return nil
}

func isAllowedValue(value string, allowed []string) bool {
	for _, option := range allowed {
		if value == option {
			return true
		}
	}

	return false
}

func listCRsWithFilter(c *gin.Context, field string, value string) {
	db, ok := connectDB(c)
	if !ok {
		return
	}

	pagination := utils.ParsePagination(c, 10, 100)

	query := db.Model(&models.ChangeRequest{}).Where(field+" = ?", value)

	var total int64
	query.Count(&total)

	var crs []models.ChangeRequest
	query.Preload("Creator").
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
			"module_options":   moduleOptions,
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

	effectiveStatus := request.Status
	if effectiveStatus == "" {
		effectiveStatus = defaultCRStatus
	}

	if err := validateKeteranganByStatus(effectiveStatus, request.Keterangan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	cr := models.ChangeRequest{
		Title:          request.Title,
		Description:    request.Description,
		PIC:            request.PIC,
		Modul:          request.Modul,
		Category:       request.Category,
		Keterangan:     request.Keterangan,
		ReleaseDate:    request.ReleaseDate,
		EndDate:        request.EndDate,
		FileAttachment: request.FileAttachment,
		CreatedBy:      claims.UserID,
	}

	cr.Status = effectiveStatus

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
		PIC:            request.PIC,
		Modul:          request.Modul,
		Category:       request.Category,
		Status:         request.Status,
		Keterangan:     request.Keterangan,
		ReleaseDate:    request.ReleaseDate,
		EndDate:        request.EndDate,
		FileAttachment: request.FileAttachment,
	}

	if err := validateKeteranganByStatus(request.Status, request.Keterangan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
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

// GetCRsByStatus godoc
// @Summary List Change Requests by status
// @Description Get paginated list of change requests filtered by status.
// @Tags CR
// @Produce json
// @Param status path string true "CR status"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/cr/status/{status} [get]
func GetCRsByStatus(c *gin.Context) {
	status := strings.ToUpper(strings.TrimSpace(c.Param("status")))

	if !isAllowedValue(status, statusOptions) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid status value",
		})
		return
	}

	listCRsWithFilter(c, "status", status)
}

// GetCRsByModule godoc
// @Summary List Change Requests by modul
// @Description Get paginated list of change requests filtered by modul.
// @Tags CR
// @Produce json
// @Param modul path string true "CR modul"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/cr/modul/{modul} [get]
func GetCRsByModule(c *gin.Context) {
	modul := strings.ToUpper(strings.TrimSpace(c.Param("modul")))

	if !isAllowedValue(modul, moduleOptions) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid modul value",
		})
		return
	}

	listCRsWithFilter(c, "modul", modul)
}
