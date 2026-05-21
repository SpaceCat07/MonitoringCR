package controllers

import (
	"MonCR/models"
	"MonCR/utils"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type createActivityRequest struct {
	CRID       uint   `json:"cr_id" binding:"required"`
	Action     string `json:"action" binding:"required,oneof=Comment Change"`
	Activities string `json:"activity" binding:"omitempty"`
	Comment    string `json:"comment" binding:"omitempty"`
	ReplyToID  *uint  `json:"reply_to_id" binding:"omitempty"` // Optional: for replying to activities
}

type updateActivityRequest struct {
	Action     string `json:"action" binding:"required,oneof=Comment Change"`
	Activities string `json:"activity" binding:"omitempty"`
	Comment    string `json:"comment" binding:"omitempty"`
}

func parseActivityID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, utils.FormatResponse("Invalid Activity id", http.StatusBadRequest, "error", nil))
		return 0, false
	}
	return uint(id), true
}

func findActivityByID(db *gorm.DB, id uint) (*models.Activity, error) {
	var activity models.Activity
	if err := db.Preload("CR").Preload("User").
		Preload("RepliedActivity").Preload("RepliedActivity.User").
		Preload("Replies", func(db *gorm.DB) *gorm.DB {
			return db.Preload("User").Order("created_at ASC")
		}).
		First(&activity, id).Error; err != nil {
		return nil, err
	}
	return &activity, nil
}

// CreateActivity godoc
// @Summary Create Activity
// @Description Create a new activity. UserID is extracted from token.
// @Tags Activity
// @Accept json
// @Produce json
// @Param payload body createActivityRequest true "Create Activity payload"
// @Success 201 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 401 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/activities [post]
func CreateActivity(c *gin.Context) {
	var request createActivityRequest

	db, ok := connectDB(c)
	if !ok {
		return
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, utils.FormatResponse("Invalid input data", http.StatusBadRequest, "error", err.Error()))
		return
	}

	claims, ok := getClaims(c)
	if !ok {
		return
	}

	// Verify CRID exists
	var cr models.ChangeRequest
	if err := db.First(&cr, request.CRID).Error; err != nil {
		c.JSON(http.StatusBadRequest, utils.FormatResponse("Invalid CR ID", http.StatusBadRequest, "error", nil))
		return
	}

	activity := models.Activity{
		CRID:       &request.CRID,
		UserID:     &claims.UserID,
		Action:     request.Action,
		Activities: request.Activities,
		Comment:    request.Comment,
		ReplyToID:  request.ReplyToID, // Set reply_to_id if provided
	}

	if err := db.Create(&activity).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to create Activity", http.StatusInternalServerError, "error", err.Error()))
		return
	}

	db.Preload("CR").Preload("User").Preload("RepliedActivity").Preload("RepliedActivity.User").First(&activity, activity.ID)

	c.JSON(http.StatusCreated, utils.FormatResponse("Activity created successfully", http.StatusCreated, "success", activity))
}

// GetActivities godoc
// @Summary List Activities
// @Description Get paginated list of activities, optionally filter by cr_id
// @Tags Activity
// @Produce json
// @Param cr_id query int false "Filter by CR ID"
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/activities [get]
func GetActivities(c *gin.Context) {
	db, ok := connectDB(c)
	if !ok {
		return
	}

	pagination := utils.ParsePagination(c, 10, 100)
	query := db.Model(&models.Activity{})

	// Optional filter by cr_id
	crIDStr := c.Query("cr_id")
	if crIDStr != "" {
		if crID, err := strconv.ParseUint(crIDStr, 10, 32); err == nil {
			query = query.Where("cr_id = ?", crID)
		}
	}

	var total int64
	query.Count(&total)

	var activities []models.Activity
	query.Preload("CR").Preload("User").
		Preload("RepliedActivity").Preload("RepliedActivity.User").
		Preload("Replies", func(db *gorm.DB) *gorm.DB {
			return db.Preload("User").Order("created_at ASC")
		}).
		Order("id DESC").
		Offset(pagination.Offset).
		Limit(pagination.Limit).
		Find(&activities)

	paginationMeta := utils.BuildPaginationMeta(pagination.Offset, pagination.Limit, len(activities), total)

	c.JSON(http.StatusOK, utils.FormatResponse("Activities retrieved successfully", http.StatusOK, "success", gin.H{
		"items":      activities,
		"pagination": paginationMeta,
	}))
}

// GetActivityByID godoc
// @Summary Get Activity by ID
// @Description Retrieve one activity by id.
// @Tags Activity
// @Produce json
// @Param id path int true "Activity ID"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/activities/{id} [get]
func GetActivityByID(c *gin.Context) {
	db, ok := connectDB(c)
	if !ok {
		return
	}

	id, ok := parseActivityID(c)
	if !ok {
		return
	}

	activity, err := findActivityByID(db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, utils.FormatResponse("Activity not found", http.StatusNotFound, "error", nil))
			return
		}

		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to retrieve Activity", http.StatusInternalServerError, "error", nil))
		return
	}

	c.JSON(http.StatusOK, utils.FormatResponse("Activity retrieved successfully", http.StatusOK, "success", activity))
}

// UpdateActivity godoc
// @Summary Update Activity
// @Description Fully update activity by id. Note: CRID and UserID cannot be updated here.
// @Tags Activity
// @Accept json
// @Produce json
// @Param id path int true "Activity ID"
// @Param payload body updateActivityRequest true "Update Activity payload"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/activities/{id} [put]
func UpdateActivity(c *gin.Context) {
	claims, ok := getClaims(c)
	if !ok {
		return
	}

	db, ok := connectDB(c)
	if !ok {
		return
	}

	id, ok := parseActivityID(c)
	if !ok {
		return
	}

	activity, err := findActivityByID(db, id)
	if err != nil {
		c.JSON(http.StatusNotFound, utils.FormatResponse("Activity not found", http.StatusNotFound, "error", nil))
		return
	}
	if activity.Action != "Comment" {
		c.JSON(http.StatusForbidden, utils.FormatResponse("Hanya komentar yang dapat diedit", http.StatusForbidden, "error", nil))
		return
	}
	if activity.UserID == nil || (*activity.UserID != claims.UserID && claims.Role != "Admin") {
		c.JSON(http.StatusForbidden, utils.FormatResponse("Hanya pemilik komentar yang dapat mengedit komentar ini", http.StatusForbidden, "error", nil))
		return
	}

	var request updateActivityRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, utils.FormatResponse("Invalid input data", http.StatusBadRequest, "error", err.Error()))
		return
	}

	updates := models.Activity{
		Action:  "Comment",
		Comment: request.Comment,
	}

	if err := db.Model(&models.Activity{}).Where("id = ?", id).Updates(&updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to update Activity", http.StatusInternalServerError, "error", nil))
		return
	}

	updatedActivity, _ := findActivityByID(db, id)

	c.JSON(http.StatusOK, utils.FormatResponse("Activity updated successfully", http.StatusOK, "success", updatedActivity))
}

// DeleteActivity godoc
// @Summary Delete Activity
// @Description Delete activity by id.
// @Tags Activity
// @Produce json
// @Param id path int true "Activity ID"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/activities/{id} [delete]
func DeleteActivity(c *gin.Context) {
	claims, ok := getClaims(c)
	if !ok {
		return
	}

	db, ok := connectDB(c)
	if !ok {
		return
	}

	id, ok := parseActivityID(c)
	if !ok {
		return
	}

	activity, err := findActivityByID(db, id)
	if err != nil {
		c.JSON(http.StatusNotFound, utils.FormatResponse("Activity not found", http.StatusNotFound, "error", nil))
		return
	}
	if activity.Action != "Comment" {
		c.JSON(http.StatusForbidden, utils.FormatResponse("Hanya komentar yang dapat dihapus", http.StatusForbidden, "error", nil))
		return
	}
	if activity.UserID == nil || (*activity.UserID != claims.UserID && claims.Role != "Admin") {
		c.JSON(http.StatusForbidden, utils.FormatResponse("Hanya pemilik komentar yang dapat menghapus komentar ini", http.StatusForbidden, "error", nil))
		return
	}

	if err := db.Delete(activity).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to delete Activity", http.StatusInternalServerError, "error", nil))
		return
	}

	c.JSON(http.StatusOK, utils.FormatResponse("Activity deleted successfully", http.StatusOK, "success", nil))
}
