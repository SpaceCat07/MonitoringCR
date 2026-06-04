package controllers

import (
	"MonCR/config"
	"MonCR/models"
	"MonCR/utils"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
	"gorm.io/gorm"
)

const defaultCRStatus = "DRAFT"

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
	"DRAFT",
	"ISSUED",
	"IN_PROGRESS",
	"APPROVAL_TO_RELEASE",
	"RELEASE",
	"APPROVAL_TO_COMPLETE",
	"COMPLETE",
	"CANCEL",
}

type createCRRequest struct {
	Title          string    `json:"title" binding:"required,min=3,max=255"`
	Description    string    `json:"description" binding:"required,min=3"`
	Goal           string    `json:"goal" binding:"required"`
	Impact         string    `json:"impact" binding:"required"`
	Keterangan     string    `json:"keterangan" binding:"omitempty"`
	Modul          string    `json:"modul" binding:"required,oneof=FINANCE 'MATERIAL MANAGEMENT' 'HUMAN RESOURCE' BASIS ABAP"`
	Category       string    `json:"category" binding:"required,oneof=FLOW REPORT INTERFACE CONVERTION ENHANCEMENT FORM CONFIGURATION AUTHORIZATION"`
	Status         string    `json:"status" binding:"omitempty,oneof=DRAFT ISSUED IN_PROGRESS APPROVAL_TO_RELEASE RELEASE APPROVAL_TO_COMPLETE COMPLETE CANCEL"`
	ReleaseDate    time.Time `json:"release_date" binding:"required"`
	StartDate      time.Time `json:"start_date" binding:"required"`
	EndDate        time.Time `json:"end_date" binding:"required"`
	FileAttachment []string  `json:"file_attachment" binding:"required,min=1,dive,required"`
	PICID          *uint     `json:"pic_id" binding:"omitempty"`
}

type updateCRRequest struct {
	Title          string    `json:"title" binding:"required,min=3,max=255"`
	Description    string    `json:"description" binding:"required,min=3"`
	Goal           string    `json:"goal" binding:"required"`
	Impact         string    `json:"impact" binding:"required"`
	Keterangan     string    `json:"keterangan" binding:"omitempty"`
	Modul          string    `json:"modul" binding:"required,oneof=FINANCE 'MATERIAL MANAGEMENT' 'HUMAN RESOURCE' BASIS ABAP"`
	Category       string    `json:"category" binding:"required,oneof=FLOW REPORT INTERFACE CONVERTION ENHANCEMENT FORM CONFIGURATION AUTHORIZATION"`
	Status         string    `json:"status" binding:"required,oneof=DRAFT ISSUED IN_PROGRESS APPROVAL_TO_RELEASE RELEASE APPROVAL_TO_COMPLETE COMPLETE CANCEL"`
	ReleaseDate    time.Time `json:"release_date" binding:"required"`
	StartDate      time.Time `json:"start_date" binding:"required"`
	EndDate        time.Time `json:"end_date" binding:"required"`
	FileAttachment []string  `json:"file_attachment" binding:"required,min=1,dive,required"`
	PICID          *uint     `json:"pic_id" binding:"omitempty"`
}

func requiresKeterangan(status string) bool {
	// Keterangan hanya wajib saat CANCEL (bukan ISSUED)
	return status == "CANCEL"
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

func truncateActivityText(s string) string {
	if len(s) <= 255 {
		return s
	}
	return s[:255]
}

func createChangeActivity(db *gorm.DB, crID uint, userID uint, text string) {
	activityLog := models.Activity{
		CRID:       &crID,
		UserID:     &userID,
		Action:     "Change",
		Activities: truncateActivityText(text),
	}
	db.Create(&activityLog)
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
// @Success 200 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/spr/options [get]
func GetCROptions(c *gin.Context) {
	c.JSON(http.StatusOK, utils.FormatResponse("Options retrieved successfully", http.StatusOK, "success", gin.H{
		"module_options":   moduleOptions,
		"category_options": categoryOptions,
		"status_options":   statusOptions,
	}))
}

func connectDB(c *gin.Context) (*gorm.DB, bool) {
	db, err := config.DBConnect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to connect to database", http.StatusInternalServerError, "error", nil))
		return nil, false
	}
	return db, true
}

func getClaims(c *gin.Context) (*utils.Claims, bool) {
	claimsValue, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, utils.FormatResponse("Unauthorized", http.StatusUnauthorized, "error", nil))
		return nil, false
	}

	claims, ok := claimsValue.(*utils.Claims)
	if !ok {
		c.JSON(http.StatusUnauthorized, utils.FormatResponse("Invalid token claims", http.StatusUnauthorized, "error", nil))
		return nil, false
	}

	return claims, true
}

func parseCRID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, utils.FormatResponse("Invalid Change Request id", http.StatusBadRequest, "error", nil))
		return 0, false
	}
	return uint(id), true
}

func findCRByID(db *gorm.DB, id uint) (*models.ChangeRequest, error) {
	var cr models.ChangeRequest
	if err := db.
		Preload("Creator").
		Preload("PIC").
		Preload("Activities", func(db *gorm.DB) *gorm.DB {
			return db.Order("activities.created_at DESC")
		}).
		Preload("Activities.User").
		Preload("SubTasks", func(db *gorm.DB) *gorm.DB {
			return db.Order("sub_tasks.created_at ASC")
		}).
		First(&cr, id).Error; err != nil {
		return nil, err
	}
	return &cr, nil
}

// CreateCR godoc
// @Summary Create Change Request
// @Description Create a new change request. Only Admin can create CR.
// @Tags CR
// @Accept json
// @Produce json
// @Param payload body createCRRequest true "Create CR payload"
// @Success 201 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 401 {object} utils.APIResponse
// @Failure 403 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/spr [post]
func CreateCR(c *gin.Context) {
	claims, ok := getClaims(c)
	if !ok {
		return
	}

	if claims.Role != "Admin" {
		c.JSON(http.StatusForbidden, utils.FormatResponse("Hanya Admin yang dapat membuat Change Request", http.StatusForbidden, "error", nil))
		return
	}

	var request createCRRequest

	db, ok := connectDB(c)
	if !ok {
		return
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, utils.FormatResponse("Invalid input data", http.StatusBadRequest, "error", err.Error()))
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
		Goal:           request.Goal,
		Impact:         request.Impact,
		Keterangan:     request.Keterangan,
		Modul:          request.Modul,
		Category:       request.Category,
		ReleaseDate:    request.ReleaseDate,
		StartDate:      request.StartDate,
		EndDate:        request.EndDate,
		FileAttachment: request.FileAttachment,
		CreatedBy:      claims.UserID,
		PICID:          request.PICID,
	}

	cr.Status = effectiveStatus

	if err := db.Create(&cr).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to create Change Request", http.StatusInternalServerError, "error", err.Error()))
		return
	}

	createChangeActivity(db, cr.ID, claims.UserID, fmt.Sprintf("Created a new Change Request: '%s'", cr.Title))

	db.Preload("Creator").First(&cr, cr.ID)

	c.JSON(http.StatusCreated, utils.FormatResponse("Change Request created successfully", http.StatusCreated, "success", cr))
}

// GetCRs godoc
// @Summary List Change Requests
// @Description Get paginated list of change requests.
// @Tags CR
// @Produce json
// @Param status query string false "Filter by status"
// @Param modul query string false "Filter by modul"
// @Param category query string false "Filter by category"
// @Param search query string false "Search by ID or title"
// @Param id query string false "Search by ID"
// @Param title query string false "Search by title"
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/spr [get]
func GetCRs(c *gin.Context) {
	claims, ok := getClaims(c)
	if !ok {
		return
	}

	db, ok := connectDB(c)
	if !ok {
		return
	}

	// pagination := utils.ParsePagination(c, 10, 100)
	query := db.Model(&models.ChangeRequest{})

	// Role-based visibility
	if claims.Role == "PIC" {
		query = query.Where("pic_id = ?", claims.UserID)
	} else if claims.Role == "Collaborator" {
		var user models.Users
		if err := db.Select("parent_pic").First(&user, claims.UserID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to resolve collaborator parent PIC", http.StatusInternalServerError, "error", err.Error()))
			return
		}
		if user.ParentPIC == nil {
			query = query.Where("1 = 0")
		} else {
			query = query.Where("pic_id = ?", *user.ParentPIC)
		}
	}

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if modul := c.Query("modul"); modul != "" {
		query = query.Where("modul = ?", modul)
	}
	if category := c.Query("category"); category != "" {
		query = query.Where("category = ?", category)
	}

	// Search parameters
	if search := c.Query("search"); search != "" {
		// Search by ID or title
		query = query.Where("id LIKE ? OR title LIKE ?", "%"+search+"%", "%"+search+"%")
	} else {
		// Search by specific ID
		if id := c.Query("id"); id != "" {
			query = query.Where("id LIKE ?", "%"+id+"%")
		}
		// Search by title
		if title := c.Query("title"); title != "" {
			query = query.Where("title LIKE ?", "%"+title+"%")
		}
	}

	// var total int64
	// query.Count(&total)

	var crs []models.ChangeRequest
	// query.Preload("Creator").Preload("PIC").Preload("SubTasks").Preload("SubTasks.Collaborator").
	// 	Order("id DESC").
	// 	Offset(pagination.Offset).
	// 	Limit(pagination.Limit).
	// 	Find(&crs)

	query.Preload("Creator").Preload("PIC").Preload("SubTasks").Preload("SubTasks.Collaborator").
		Order("id DESC").
		Find(&crs)

	// paginationMeta := utils.BuildPaginationMeta(pagination.Offset, pagination.Limit, len(crs), total)

	c.JSON(http.StatusOK, utils.FormatResponse("Change Requests retrieved successfully", http.StatusOK, "success", gin.H{
		"items": crs,
		// "pagination": paginationMeta,
	}))
}

// GetCRByID godoc
// @Summary Get Change Request by ID
// @Description Retrieve one change request by id.
// @Tags CR
// @Produce json
// @Param id path int true "CR ID"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
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
			c.JSON(http.StatusNotFound, utils.FormatResponse("Change Request not found", http.StatusNotFound, "error", nil))
			return
		}

		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to retrieve Change Request", http.StatusInternalServerError, "error", nil))
		return
	}

	c.JSON(http.StatusOK, utils.FormatResponse("Change Request retrieved successfully", http.StatusOK, "success", cr))
}

// UpdateCR godoc
// @Summary Update Change Request
// @Description Fully update change request by id. State machine validation is applied.
// @Tags CR
// @Accept json
// @Produce json
// @Param id path int true "CR ID"
// @Param payload body updateCRRequest true "Update CR payload"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 403 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/spr/{id} [put]
func UpdateCR(c *gin.Context) {
	claims, ok := getClaims(c)
	if !ok {
		return
	}

	db, ok := connectDB(c)
	if !ok {
		return
	}

	id, ok := parseCRID(c)
	if !ok {
		return
	}

	currentCR, err := findCRByID(db, id)
	if err != nil {
		c.JSON(http.StatusNotFound, utils.FormatResponse("Change Request not found", http.StatusNotFound, "error", nil))
		return
	}

	var request updateCRRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, utils.FormatResponse("Invalid input data", http.StatusBadRequest, "error", err.Error()))
		return
	}

	// State Machine Validation
	if request.Status != currentCR.Status {
		role := claims.Role

		// Require all subtasks to be 100% before PIC can move to approval steps
		if role == "PIC" && (request.Status == "APPROVAL_TO_RELEASE" || request.Status == "APPROVAL_TO_COMPLETE") {
			var total int64
			var incomplete int64
			db.Model(&models.SubTask{}).Where("cr_id = ?", id).Count(&total)
			if total > 0 {
				db.Model(&models.SubTask{}).Where("cr_id = ? AND (progress < 100 OR done = false)", id).Count(&incomplete)
				if incomplete > 0 {
					c.JSON(http.StatusBadRequest, utils.FormatResponse("All subtasks must be 100% before moving to the next status", http.StatusBadRequest, "error", nil))
					return
				}
			}
		}

		// Aturan Keterangan Wajib saat CANCEL
		if request.Status == "CANCEL" && request.Keterangan == "" {
			c.JSON(http.StatusBadRequest, utils.FormatResponse("Keterangan wajib diisi ketika membatalkan CR", http.StatusBadRequest, "error", nil))
			return
		}

		if role != "Admin" {
			switch request.Status {
			case "ISSUED":
				if role != "PIC" || currentCR.Status != "DRAFT" {
					c.JSON(http.StatusForbidden, utils.FormatResponse("Transisi ke ISSUED hanya dapat dilakukan oleh PIC/Admin dari status DRAFT", http.StatusForbidden, "error", nil))
					return
				}
			case "IN_PROGRESS":
				if role != "Manager" {
					c.JSON(http.StatusForbidden, utils.FormatResponse("Perubahan ke IN_PROGRESS memerlukan persetujuan Manager", http.StatusForbidden, "error", nil))
					return
				}
				if currentCR.Status != "ISSUED" && currentCR.Status != "APPROVAL_TO_RELEASE" && currentCR.Status != "APPROVAL_TO_COMPLETE" {
					c.JSON(http.StatusBadRequest, utils.FormatResponse("Transisi status tidak valid", http.StatusBadRequest, "error", nil))
					return
				}
			case "APPROVAL_TO_RELEASE":
				if role != "PIC" || currentCR.Status != "IN_PROGRESS" || currentCR.PICID == nil || *currentCR.PICID != claims.UserID {
					c.JSON(http.StatusForbidden, utils.FormatResponse("Transisi ke APPROVAL_TO_RELEASE hanya dapat dilakukan oleh PIC dari CR ini", http.StatusForbidden, "error", nil))
					return
				}
			case "RELEASE":
				if role != "Manager" || currentCR.Status != "APPROVAL_TO_RELEASE" {
					c.JSON(http.StatusForbidden, utils.FormatResponse("Transisi ke RELEASE memerlukan persetujuan Manager dari status APPROVAL_TO_RELEASE", http.StatusForbidden, "error", nil))
					return
				}
			case "APPROVAL_TO_COMPLETE":
				if role != "PIC" || currentCR.Status != "RELEASE" || currentCR.PICID == nil || *currentCR.PICID != claims.UserID {
					c.JSON(http.StatusForbidden, utils.FormatResponse("Transisi ke APPROVAL_TO_COMPLETE hanya dapat dilakukan oleh PIC dari CR ini", http.StatusForbidden, "error", nil))
					return
				}
			case "COMPLETE":
				if role != "Manager" || currentCR.Status != "APPROVAL_TO_COMPLETE" {
					c.JSON(http.StatusForbidden, utils.FormatResponse("Penyelesaian CR memerlukan persetujuan Manager dari status APPROVAL_TO_COMPLETE", http.StatusForbidden, "error", nil))
					return
				}
			case "CANCEL":
				if role != "Manager" {
					c.JSON(http.StatusForbidden, utils.FormatResponse("Hanya Manager/Admin yang dapat melakukan CANCEL", http.StatusForbidden, "error", nil))
					return
				}
			default:
				c.JSON(http.StatusBadRequest, utils.FormatResponse("Transisi status tidak dikenal", http.StatusBadRequest, "error", nil))
				return
			}
		}
	}

	updates := models.ChangeRequest{
		Title:          request.Title,
		Description:    request.Description,
		Goal:           request.Goal,
		Impact:         request.Impact,
		Keterangan:     request.Keterangan,
		Modul:          request.Modul,
		Category:       request.Category,
		Status:         request.Status,
		ReleaseDate:    request.ReleaseDate,
		StartDate:      request.StartDate,
		EndDate:        request.EndDate,
		FileAttachment: request.FileAttachment,
		PICID:          request.PICID,
	}

	ptrUintEqual := func(a, b *uint) bool {
		if a == nil && b == nil {
			return true
		}
		if a == nil || b == nil {
			return false
		}
		return *a == *b
	}
	resolvePICName := func(id *uint) string {
		if id == nil {
			return "Unassigned"
		}
		var u models.Users
		if err := db.Select("fullname").First(&u, *id).Error; err != nil || strings.TrimSpace(u.Fullname) == "" {
			return fmt.Sprintf("User#%d", *id)
		}
		return u.Fullname
	}

	if err := validateKeteranganByStatus(request.Status, request.Keterangan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if err := db.Model(&models.ChangeRequest{}).Where("id = ?", id).Updates(&updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to update Change Request", http.StatusInternalServerError, "error", nil))
		return
	}

	toDateStr := func(t time.Time) string {
		return t.Format("2006-01-02")
	}
	toFilesStr := func(files []string) string {
		if len(files) == 0 {
			return "-"
		}
		return strings.Join(files, ", ")
	}
	logFieldChange := func(fieldName, oldValue, newValue string) {
		createChangeActivity(db, id, claims.UserID, fmt.Sprintf("Changed %s from '%s' to '%s'", fieldName, oldValue, newValue))
	}

	if request.Title != currentCR.Title {
		logFieldChange("Title", currentCR.Title, request.Title)
	}
	if request.Description != currentCR.Description {
		logFieldChange("Description", currentCR.Description, request.Description)
	}
	if request.Goal != currentCR.Goal {
		logFieldChange("Goal", currentCR.Goal, request.Goal)
	}
	if request.Impact != currentCR.Impact {
		logFieldChange("Impact", currentCR.Impact, request.Impact)
	}
	if request.Keterangan != currentCR.Keterangan {
		logFieldChange("Keterangan", currentCR.Keterangan, request.Keterangan)
	}
	if request.Modul != currentCR.Modul {
		logFieldChange("Module", currentCR.Modul, request.Modul)
	}
	if request.Category != currentCR.Category {
		logFieldChange("Category", currentCR.Category, request.Category)
	}
	if request.Status != currentCR.Status {
		logFieldChange("Status", currentCR.Status, request.Status)
	}
	if !request.ReleaseDate.Equal(currentCR.ReleaseDate) {
		logFieldChange("Release Date", toDateStr(currentCR.ReleaseDate), toDateStr(request.ReleaseDate))
	}
	if !request.StartDate.Equal(currentCR.StartDate) {
		logFieldChange("Start Date", toDateStr(currentCR.StartDate), toDateStr(request.StartDate))
	}
	if !request.EndDate.Equal(currentCR.EndDate) {
		logFieldChange("End Date", toDateStr(currentCR.EndDate), toDateStr(request.EndDate))
	}
	if strings.Join(request.FileAttachment, "|") != strings.Join(currentCR.FileAttachment, "|") {
		logFieldChange("File Attachment", toFilesStr(currentCR.FileAttachment), toFilesStr(request.FileAttachment))
	}
	if !ptrUintEqual(request.PICID, currentCR.PICID) {
		logFieldChange("PIC", resolvePICName(currentCR.PICID), resolvePICName(request.PICID))
	}

	updatedCR, _ := findCRByID(db, id)

	c.JSON(http.StatusOK, utils.FormatResponse("Change Request updated successfully", http.StatusOK, "success", updatedCR))
}

// DeleteCR godoc
// @Summary Delete Change Request
// @Description Delete change request by id.
// @Tags CR
// @Produce json
// @Param id path int true "CR ID"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/spr/{id} [delete]
func DeleteCR(c *gin.Context) {
	claims, ok := getClaims(c)
	if !ok {
		return
	}

	if claims.Role != "Admin" {
		c.JSON(http.StatusForbidden, utils.FormatResponse("Hanya Admin yang dapat menghapus Change Request", http.StatusForbidden, "error", nil))
		return
	}

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
		c.JSON(http.StatusNotFound, utils.FormatResponse("Change Request not found", http.StatusNotFound, "error", nil))
		return
	}
	if cr.Status != "DRAFT" {
		c.JSON(http.StatusBadRequest, utils.FormatResponse("CR hanya dapat dihapus ketika status DRAFT", http.StatusBadRequest, "error", nil))
		return
	}

	if err := db.Delete(cr).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to delete Change Request", http.StatusInternalServerError, "error", nil))
		return
	}

	c.JSON(http.StatusOK, utils.FormatResponse("Change Request deleted successfully", http.StatusOK, "success", nil))
}

type exportCRFilters struct {
	Status       string
	Module       string
	Category     string
	Deadline     string
	PICID        *uint
	PICName      string
	Year         int
	Semester     int
	DateFrom     time.Time
	DateTo       time.Time
	HasDateRange bool
}

func firstNonEmptyQuery(c *gin.Context, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(c.Query(key))
		if value != "" {
			return value
		}
	}
	return ""
}

func normalizeExportStatus(raw string) string {
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value == "" || value == "ALL" {
		return ""
	}
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func normalizeExportModule(raw string) string {
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value == "" || value == "ALL" {
		return ""
	}
	return value
}

func normalizeExportCategory(raw string) string {
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value == "" || value == "ALL" {
		return ""
	}
	switch value {
	case "CONVERSION":
		return "CONVERTION"
	case "AUTORIZATION":
		return "AUTHORIZATION"
	default:
		return value
	}
}

func normalizeExportDeadline(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "", "all":
		return ""
	case "3", "3day":
		return "3"
	case "7", "7day":
		return "7"
	case "overdue":
		return "overdue"
	default:
		return ""
	}
}

func parseExportCRFilters(c *gin.Context) (exportCRFilters, error) {
	filters := exportCRFilters{}

	filters.Status = normalizeExportStatus(firstNonEmptyQuery(c, "status"))
	if filters.Status != "" && !isAllowedValue(filters.Status, statusOptions) {
		return filters, fmt.Errorf("invalid status filter: %s", filters.Status)
	}

	filters.Module = normalizeExportModule(firstNonEmptyQuery(c, "modul", "module"))
	if filters.Module != "" && !isAllowedValue(filters.Module, moduleOptions) {
		return filters, fmt.Errorf("invalid module filter: %s", filters.Module)
	}

	filters.Category = normalizeExportCategory(firstNonEmptyQuery(c, "category"))
	if filters.Category != "" {
		switch filters.Category {
		case "AUTHORIZATION", "AUTORIZATION":
			// Allow both spellings for backward compatibility.
		default:
			if !isAllowedValue(filters.Category, categoryOptions) {
				return filters, fmt.Errorf("invalid category filter: %s", filters.Category)
			}
		}
	}

	deadlineRaw := firstNonEmptyQuery(c, "deadline")
	filters.Deadline = normalizeExportDeadline(deadlineRaw)
	if strings.TrimSpace(deadlineRaw) != "" && filters.Deadline == "" {
		return filters, fmt.Errorf("invalid deadline filter: %s", deadlineRaw)
	}

	if picIDRaw := firstNonEmptyQuery(c, "pic_id"); picIDRaw != "" {
		parsedPICID, err := strconv.ParseUint(picIDRaw, 10, 32)
		if err != nil || parsedPICID == 0 {
			return filters, fmt.Errorf("invalid pic_id filter: %s", picIDRaw)
		}
		picID := uint(parsedPICID)
		filters.PICID = &picID
	}

	picName := strings.TrimSpace(firstNonEmptyQuery(c, "pic_name"))
	if !strings.EqualFold(picName, "all") {
		filters.PICName = picName
	}

	yearRaw := firstNonEmptyQuery(c, "year")
	if yearRaw != "" {
		parsedYear, err := strconv.Atoi(yearRaw)
		if err != nil || parsedYear < 2000 || parsedYear > 2100 {
			return filters, fmt.Errorf("invalid year filter: %s", yearRaw)
		}
		filters.Year = parsedYear
	}

	semesterRaw := firstNonEmptyQuery(c, "semester")
	if semesterRaw != "" {
		parsedSemester, err := strconv.Atoi(semesterRaw)
		if err != nil || (parsedSemester != 1 && parsedSemester != 2) {
			return filters, fmt.Errorf("invalid semester filter: %s", semesterRaw)
		}
		filters.Semester = parsedSemester
	}

	if filters.Semester != 0 && filters.Year == 0 {
		filters.Year = time.Now().Year()
	}

	if filters.Year != 0 {
		loc := time.UTC
		switch filters.Semester {
		case 1:
			filters.DateFrom = time.Date(filters.Year, time.January, 1, 0, 0, 0, 0, loc)
			filters.DateTo = time.Date(filters.Year, time.July, 1, 0, 0, 0, 0, loc)
		case 2:
			filters.DateFrom = time.Date(filters.Year, time.July, 1, 0, 0, 0, 0, loc)
			filters.DateTo = time.Date(filters.Year+1, time.January, 1, 0, 0, 0, 0, loc)
		default:
			filters.DateFrom = time.Date(filters.Year, time.January, 1, 0, 0, 0, 0, loc)
			filters.DateTo = time.Date(filters.Year+1, time.January, 1, 0, 0, 0, 0, loc)
		}
		filters.HasDateRange = true
	}

	return filters, nil
}

func applyExportCRFilters(query *gorm.DB, filters exportCRFilters) *gorm.DB {
	if filters.Status != "" {
		query = query.Where("UPPER(change_requests.status) = ?", filters.Status)
	}

	if filters.Module != "" {
		query = query.Where("UPPER(change_requests.modul) = ?", filters.Module)
	}

	if filters.Category != "" {
		if filters.Category == "AUTHORIZATION" {
			query = query.Where("UPPER(change_requests.category) IN ?", []string{"AUTHORIZATION", "AUTORIZATION"})
		} else {
			query = query.Where("UPPER(change_requests.category) = ?", filters.Category)
		}
	}

	if filters.PICID != nil {
		query = query.Where("change_requests.pic_id = ?", *filters.PICID)
	} else if filters.PICName != "" {
		query = query.Joins("LEFT JOIN users pic_users ON pic_users.id = change_requests.pic_id").
			Where("LOWER(pic_users.fullname) = ?", strings.ToLower(filters.PICName))
	}

	switch filters.Deadline {
	case "3":
		query = query.Where("change_requests.end_date::date BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL '3 days'")
	case "7":
		query = query.Where("change_requests.end_date::date > CURRENT_DATE + INTERVAL '3 days' AND change_requests.end_date::date < CURRENT_DATE + INTERVAL '7 days'")
	case "overdue":
		query = query.Where("change_requests.end_date::date < CURRENT_DATE")
	}

	if filters.HasDateRange {
		query = query.Where(
			"change_requests.start_date::date >= ?::date AND change_requests.start_date::date < ?::date",
			filters.DateFrom.Format("2006-01-02"),
			filters.DateTo.Format("2006-01-02"),
		)
	}

	return query
}

// ExportCRsPDF godoc
// @Summary Export Change Requests to PDF
// @Description Export change requests to PDF format with support for deadline and PIC filters.
// @Tags CR
// @Produce application/pdf
// @Param status query string false "Filter by status"
// @Param modul query string false "Filter by modul"
// @Param category query string false "Filter by category"
// @Param pic_id query int false "Filter by PIC ID"
// @Param deadline query string false "Filter by deadline (3, 7, or overdue)"
// @Success 200 {file} file
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/spr/export [get]
func ExportCRsPDF(c *gin.Context) {
	claims, ok := getClaims(c)
	if !ok {
		return
	}

	db, ok := connectDB(c)
	if !ok {
		return
	}

	filters, err := parseExportCRFilters(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.FormatResponse(err.Error(), http.StatusBadRequest, "error", nil))
		return
	}

	query := db.Model(&models.ChangeRequest{}).Where("change_requests.deleted_at IS NULL")

	// Role-based visibility
	if claims.Role == "PIC" {
		query = query.Where("change_requests.pic_id = ?", claims.UserID)
	} else if claims.Role == "Collaborator" {
		var user models.Users
		if err := db.Select("parent_pic").First(&user, claims.UserID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to resolve collaborator parent PIC", http.StatusInternalServerError, "error", err.Error()))
			return
		}
		if user.ParentPIC == nil {
			query = query.Where("1 = 0")
		} else {
			query = query.Where("change_requests.pic_id = ?", *user.ParentPIC)
		}
	}

	query = applyExportCRFilters(query, filters)

	var totalMatched int64
	if err := query.Count(&totalMatched).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to count export data", http.StatusInternalServerError, "error", err.Error()))
		return
	}
	log.Printf("[ExportCRsPDF] user_id=%d role=%s filters=%+v total_matched=%d", claims.UserID, claims.Role, filters, totalMatched)

	var crs []models.ChangeRequest
	if err := query.
		Preload("PIC").
		Order("change_requests.start_date ASC, change_requests.id ASC").
		Find(&crs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to fetch data for export", http.StatusInternalServerError, "error", nil))
		return
	}
	log.Printf("[ExportCRsPDF] user_id=%d exported_rows=%d", claims.UserID, len(crs))

	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(true, 10)
	pdf.AliasNbPages("{nb}")
	leftMargin, _, _, _ := pdf.GetMargins()
	pdf.SetFooterFunc(func() {
		pdf.SetY(-8)
		pdf.SetFont("Arial", "", 7)
		pdf.SetTextColor(100, 100, 100)
		pdf.CellFormat(0, 4, fmt.Sprintf("Page %d/{nb}", pdf.PageNo()), "", 0, "R", false, 0, "")
	})

	valueOrAll := func(v string) string {
		if strings.TrimSpace(v) == "" {
			return "All"
		}
		return v
	}

	semesterLabel := "All"
	switch filters.Semester {
	case 1:
		semesterLabel = "Semester 1 (Jan-Jun)"
	case 2:
		semesterLabel = "Semester 2 (Jul-Dec)"
	default:
		if filters.Year != 0 {
			semesterLabel = "Full Year"
		}
	}

	yearLabel := "All"
	if filters.Year != 0 {
		yearLabel = strconv.Itoa(filters.Year)
	}

	statusLabel := valueOrAll(filters.Status)
	moduleLabel := valueOrAll(filters.Module)
	categoryLabel := valueOrAll(filters.Category)
	picLabel := "All"
	if filters.PICID != nil {
		picLabel = fmt.Sprintf("PIC ID %d", *filters.PICID)
	} else if strings.TrimSpace(filters.PICName) != "" {
		picLabel = filters.PICName
	}
	deadlineLabel := valueOrAll(filters.Deadline)

	exportedAt := time.Now()

	headers := []string{
		"No",
		"CR ID",
		"Title",
		"Description",
		"Goal",
		"Impact",
		"Module",
		"Category",
		"Status",
		"PIC",
		"Start",
		"End",
	}
	widths := []float64{8, 16, 34, 40, 30, 30, 22, 22, 20, 25, 15, 15}
	lineHeight := 3.6
	leftPad := 0.9
	topPad := 0.6

	renderTableHeader := func() {
		pdf.SetFont("Arial", "B", 7.5)
		pdf.SetFillColor(230, 236, 244)
		pdf.SetTextColor(30, 41, 59)
		for i, header := range headers {
			pdf.CellFormat(widths[i], 6.8, header, "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)
	}

	renderPageSection := func(showSummary bool) {
		pdf.SetTextColor(20, 20, 20)
		pdf.SetFont("Arial", "B", 12)
		pdf.CellFormat(0, 6.2, "Change Request Export Report", "", 1, "L", false, 0, "")
		pdf.SetFont("Arial", "", 8)
		pdf.CellFormat(0, 4.2, fmt.Sprintf("Export Date: %s", exportedAt.Format("2006-01-02 15:04:05")), "", 1, "L", false, 0, "")
		if showSummary {
			summary := []struct {
				label string
				value string
			}{
				{label: "Year", value: yearLabel},
				{label: "Semester", value: semesterLabel},
				{label: "Module", value: moduleLabel},
				{label: "Category", value: categoryLabel},
				{label: "PIC", value: picLabel},
				{label: "Status", value: statusLabel},
				{label: "Deadline", value: deadlineLabel},
				{label: "Total Data Exported", value: strconv.Itoa(len(crs))},
			}
			for _, row := range summary {
				pdf.SetFont("Arial", "B", 8)
				pdf.CellFormat(38, 4.1, row.label+":", "", 0, "L", false, 0, "")
				pdf.SetFont("Arial", "", 8)
				pdf.CellFormat(0, 4.1, row.value, "", 1, "L", false, 0, "")
			}
		}
		pdf.Ln(1.5)
		renderTableHeader()
	}

	pdf.AddPage()
	renderPageSection(true)

	for idx, cr := range crs {
		picName := "-"
		if cr.PIC != nil && strings.TrimSpace(cr.PIC.Fullname) != "" {
			picName = strings.TrimSpace(cr.PIC.Fullname)
		}

		row := []string{
			strconv.Itoa(idx + 1),
			fmt.Sprintf("CR-%04d", cr.ID),
			cr.Title,
			cr.Description,
			cr.Goal,
			cr.Impact,
			cr.Modul,
			cr.Category,
			cr.Status,
			picName,
			cr.StartDate.Format("2006-01-02"),
			cr.EndDate.Format("2006-01-02"),
		}

		maxLines := 1
		for i, text := range row {
			safeText := strings.TrimSpace(text)
			if safeText == "" {
				safeText = "-"
			}
			lines := pdf.SplitText(safeText, widths[i]-(leftPad*2))
			if len(lines) == 0 {
				lines = []string{safeText}
			}
			if len(lines) > maxLines {
				maxLines = len(lines)
			}
		}

		rowHeight := float64(maxLines)*lineHeight + (topPad * 2)

		currentX, currentY := pdf.GetXY()
		_, pageHeight := pdf.GetPageSize()
		_, _, _, marginBottom := pdf.GetMargins()
		if currentY+rowHeight > pageHeight-marginBottom {
			pdf.AddPage()
			renderPageSection(false)
			currentX, currentY = pdf.GetXY()
		}

		if idx%2 == 1 {
			pdf.SetFillColor(249, 250, 251)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		for i, text := range row {
			safeText := strings.TrimSpace(text)
			if safeText == "" {
				safeText = "-"
			}

			cellX := currentX
			cellY := currentY

			pdf.Rect(cellX, cellY, widths[i], rowHeight, "F")
			pdf.Rect(cellX, cellY, widths[i], rowHeight, "D")

			align := "L"
			switch i {
			case 0:
				align = "C"
			case 1:
				align = "C"
			case 10, 11:
				align = "C"
			}

			pdf.SetXY(cellX+leftPad, cellY+topPad)
			pdf.SetFont("Arial", "", 7.3)
			pdf.SetTextColor(31, 41, 55)
			pdf.MultiCell(widths[i]-(leftPad*2), lineHeight, safeText, "", align, false)

			currentX += widths[i]
			pdf.SetXY(currentX, currentY)
		}

		pdf.SetXY(leftMargin, currentY+rowHeight)
	}

	filename := "change_requests.pdf"
	if filters.Year != 0 && filters.Semester != 0 {
		filename = fmt.Sprintf("change_requests_%d_semester_%d.pdf", filters.Year, filters.Semester)
	} else if filters.Year != 0 {
		filename = fmt.Sprintf("change_requests_%d.pdf", filters.Year)
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/pdf")

	if err := pdf.Output(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to generate PDF", http.StatusInternalServerError, "error", nil))
	}
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

type createDraftRequest struct {
	Title          string     `json:"title" binding:"required,min=3,max=255"`
	Description    string     `json:"description" binding:"omitempty"`
	Goal           string     `json:"goal" binding:"omitempty"`
	Impact         string     `json:"impact" binding:"omitempty"`
	Keterangan     string     `json:"keterangan" binding:"omitempty"`
	Modul          string     `json:"modul" binding:"omitempty,oneof=FINANCE 'MATERIAL MANAGEMENT' 'HUMAN RESOURCE' BASIS ABAP"`
	Category       string     `json:"category" binding:"omitempty,oneof=FLOW REPORT INTERFACE CONVERTION ENHANCEMENT FORM CONFIGURATION AUTORIZATION"`
	ReleaseDate    *time.Time `json:"release_date" binding:"omitempty"`
	StartDate      *time.Time `json:"start_date" binding:"omitempty"`
	EndDate        *time.Time `json:"end_date" binding:"omitempty"`
	FileAttachment []string   `json:"file_attachment" binding:"omitempty"`
	PICID          *uint      `json:"pic_id" binding:"omitempty"`
}

// CreateDraft godoc
// @Summary Create Draft Change Request
// @Description Create a new draft change request with minimal required fields. Only Admin can create draft CR.
// @Tags CR
// @Accept json
// @Produce json
// @Param payload body createDraftRequest true "Create Draft CR payload"
// @Success 201 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 401 {object} utils.APIResponse
// @Failure 403 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/spr/draft [post]
func CreateDraft(c *gin.Context) {
	claims, ok := getClaims(c)
	if !ok {
		return
	}

	if claims.Role != "Admin" {
		c.JSON(http.StatusForbidden, utils.FormatResponse("Hanya Admin yang dapat membuat Change Request", http.StatusForbidden, "error", nil))
		return
	}

	db, ok := connectDB(c)
	if !ok {
		return
	}

	var request createDraftRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, utils.FormatResponse("Invalid input data", http.StatusBadRequest, "error", err.Error()))
		return
	}

	// Create draft with minimal validation - only title is required
	cr := models.ChangeRequest{
		Title:          request.Title,
		Description:    request.Description,
		Goal:           request.Goal,
		Impact:         request.Impact,
		Keterangan:     request.Keterangan,
		Modul:          request.Modul,
		Category:       request.Category,
		Status:         "DRAFT",
		CreatedBy:      claims.UserID,
		PICID:          request.PICID,
		FileAttachment: request.FileAttachment,
	}

	// Set optional date fields if provided
	if request.ReleaseDate != nil {
		cr.ReleaseDate = *request.ReleaseDate
	}
	if request.StartDate != nil {
		cr.StartDate = *request.StartDate
	}
	if request.EndDate != nil {
		cr.EndDate = *request.EndDate
	}

	if err := db.Create(&cr).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to create draft Change Request", http.StatusInternalServerError, "error", err.Error()))
		return
	}

	createChangeActivity(db, cr.ID, claims.UserID, fmt.Sprintf("Created a new Change Request: '%s'", cr.Title))

	// Reload dengan relations
	db.Preload("Creator").Preload("PIC").First(&cr, cr.ID)

	c.JSON(http.StatusCreated, utils.FormatResponse("Draft Change Request created successfully", http.StatusCreated, "success", cr))
}
