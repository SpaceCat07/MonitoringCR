package controllers

import (
	"MonCR/models"
	"MonCR/utils"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type createSubtaskRequest struct {
	CRID           uint      `json:"cr_id" binding:"required"`
	CollaboratorID *uint     `json:"collaborator_id" binding:"omitempty"`
	TaskName       string    `json:"task_name" binding:"required,min=3,max=255"`
	DueDate        time.Time `json:"due_date" binding:"required"`
	Progress       uint      `json:"progress" binding:"omitempty,max=100"`
	Done           bool      `json:"done" binding:"omitempty"`
}

type updateSubtaskRequest struct {
	CRID           uint      `json:"cr_id" binding:"required"`
	CollaboratorID *uint     `json:"collaborator_id" binding:"omitempty"`
	TaskName       string    `json:"task_name" binding:"required,min=3,max=255"`
	DueDate        time.Time `json:"due_date" binding:"required"`
	Progress       uint      `json:"progress" binding:"omitempty,max=100"`
	Done           bool      `json:"done" binding:"omitempty"`
}

func parseSubtaskID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, utils.FormatResponse("Invalid Subtask id", http.StatusBadRequest, "error", nil))
		return 0, false
	}
	return uint(id), true
}

func findSubtaskByID(db *gorm.DB, id uint) (*models.SubTask, error) {
	var subtask models.SubTask
	if err := db.Preload("CR").Preload("Collaborator").First(&subtask, id).Error; err != nil {
		return nil, err
	}
	return &subtask, nil
}

// CreateSubtask godoc
// @Summary Create Subtask
// @Description Create a new subtask.
// @Tags Subtask
// @Accept json
// @Produce json
// @Param payload body createSubtaskRequest true "Create Subtask payload"
// @Success 201 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 401 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/subtasks [post]
func CreateSubtask(c *gin.Context) {
	claims, ok := getClaims(c)
	if !ok {
		return
	}

	var request createSubtaskRequest

	db, ok := connectDB(c)
	if !ok {
		return
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, utils.FormatResponse("Invalid input data", http.StatusBadRequest, "error", err.Error()))
		return
	}

	subtask := models.SubTask{
		CRID:           &request.CRID,
		CollaboratorID: request.CollaboratorID,
		TaskName:       request.TaskName,
		DueDate:        request.DueDate,
		Progress:       request.Progress,
		Done:           request.Done,
	}

	// Verify CRID exists
	var cr models.ChangeRequest
	if err := db.First(&cr, request.CRID).Error; err != nil {
		c.JSON(http.StatusBadRequest, utils.FormatResponse("Invalid CR ID", http.StatusBadRequest, "error", nil))
		return
	}

	// Authorization Check
	if claims.Role != "Admin" {
		if claims.Role != "PIC" || cr.PICID == nil || *cr.PICID != claims.UserID {
			c.JSON(http.StatusForbidden, utils.FormatResponse("Hanya Admin atau PIC CR ini yang dapat membuat subtask", http.StatusForbidden, "error", nil))
			return
		}
	}

	// Verify CollaboratorID exists if provided
	if request.CollaboratorID != nil {
		var user models.Users
		if err := db.First(&user, *request.CollaboratorID).Error; err != nil {
			c.JSON(http.StatusBadRequest, utils.FormatResponse("Invalid Collaborator ID", http.StatusBadRequest, "error", nil))
			return
		}
	}

	if err := db.Create(&subtask).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to create Subtask", http.StatusInternalServerError, "error", err.Error()))
		return
	}

	db.Preload("CR").Preload("Collaborator").First(&subtask, subtask.ID)

	c.JSON(http.StatusCreated, utils.FormatResponse("Subtask created successfully", http.StatusCreated, "success", subtask))
}

// GetSubtasks godoc
// @Summary List Subtasks
// @Description Get paginated list of subtasks, optionally filter by cr_id
// @Tags Subtask
// @Produce json
// @Param cr_id query int false "Filter by CR ID"
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/subtasks [get]
func GetSubtasks(c *gin.Context) {
	db, ok := connectDB(c)
	if !ok {
		return
	}

	pagination := utils.ParsePagination(c, 10, 100)
	query := db.Model(&models.SubTask{})

	// Optional filter by cr_id
	crIDStr := c.Query("cr_id")
	if crIDStr != "" {
		if crID, err := strconv.ParseUint(crIDStr, 10, 32); err == nil {
			query = query.Where("cr_id = ?", crID)
		}
	}

	var total int64
	query.Count(&total)

	var subtasks []models.SubTask
	query.Preload("CR").Preload("Collaborator").
		Order("id DESC").
		Offset(pagination.Offset).
		Limit(pagination.Limit).
		Find(&subtasks)

	paginationMeta := utils.BuildPaginationMeta(pagination.Offset, pagination.Limit, len(subtasks), total)

	c.JSON(http.StatusOK, utils.FormatResponse("Subtasks retrieved successfully", http.StatusOK, "success", gin.H{
		"items":      subtasks,
		"pagination": paginationMeta,
	}))
}

// GetSubtaskByID godoc
// @Summary Get Subtask by ID
// @Description Retrieve one subtask by id.
// @Tags Subtask
// @Produce json
// @Param id path int true "Subtask ID"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/subtasks/{id} [get]
func GetSubtaskByID(c *gin.Context) {
	db, ok := connectDB(c)
	if !ok {
		return
	}

	id, ok := parseSubtaskID(c)
	if !ok {
		return
	}

	subtask, err := findSubtaskByID(db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, utils.FormatResponse("Subtask not found", http.StatusNotFound, "error", nil))
			return
		}

		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to retrieve Subtask", http.StatusInternalServerError, "error", nil))
		return
	}

	c.JSON(http.StatusOK, utils.FormatResponse("Subtask retrieved successfully", http.StatusOK, "success", subtask))
}

// UpdateSubtask godoc
// @Summary Update Subtask
// @Description Fully update subtask by id.
// @Tags Subtask
// @Accept json
// @Produce json
// @Param id path int true "Subtask ID"
// @Param payload body updateSubtaskRequest true "Update Subtask payload"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/subtasks/{id} [put]
func UpdateSubtask(c *gin.Context) {
	claims, ok := getClaims(c)
	if !ok {
		return
	}

	db, ok := connectDB(c)
	if !ok {
		return
	}

	id, ok := parseSubtaskID(c)
	if !ok {
		return
	}

	currentSubtask, err := findSubtaskByID(db, id)
	if err != nil {
		c.JSON(http.StatusNotFound, utils.FormatResponse("Subtask not found", http.StatusNotFound, "error", nil))
		return
	}

	var request updateSubtaskRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, utils.FormatResponse("Invalid input data", http.StatusBadRequest, "error", err.Error()))
		return
	}

	// Role check for Subtask changes
	role := claims.Role
	if role != "Admin" {
		if role == "Collaborator" {
			if currentSubtask.CollaboratorID == nil || *currentSubtask.CollaboratorID != claims.UserID {
				c.JSON(http.StatusForbidden, utils.FormatResponse("Hanya Collaborator yang ditugaskan yang dapat mengedit subtask ini", http.StatusForbidden, "error", nil))
				return
			}
			// Collaborator can ONLY edit Progress and Done
			if request.TaskName != currentSubtask.TaskName || !request.DueDate.Equal(currentSubtask.DueDate) || request.CollaboratorID != currentSubtask.CollaboratorID {
				c.JSON(http.StatusForbidden, utils.FormatResponse("Collaborator hanya dapat mengubah progress dan status selesai", http.StatusForbidden, "error", nil))
				return
			}
			allowedProgress := map[uint]bool{35: true, 75: true, 100: true}
			if request.Progress != currentSubtask.Progress && !allowedProgress[request.Progress] {
				c.JSON(http.StatusBadRequest, utils.FormatResponse("Progress untuk collaborator hanya 35, 75, atau 100", http.StatusBadRequest, "error", nil))
				return
			}
			if request.Done != currentSubtask.Done {
				c.JSON(http.StatusForbidden, utils.FormatResponse("Hanya PIC yang dapat memindahkan subtask ke status berikutnya", http.StatusForbidden, "error", nil))
				return
			}
		} else if role == "PIC" {
			var cr models.ChangeRequest
			if err := db.First(&cr, currentSubtask.CRID).Error; err != nil || cr.PICID == nil || *cr.PICID != claims.UserID {
				c.JSON(http.StatusForbidden, utils.FormatResponse("Hanya PIC CR ini yang dapat mengedit subtask", http.StatusForbidden, "error", nil))
				return
			}
		} else {
			c.JSON(http.StatusForbidden, utils.FormatResponse("Anda tidak memiliki akses untuk mengedit subtask ini", http.StatusForbidden, "error", nil))
			return
		}
	}

	// Verify CRID exists
	var cr models.ChangeRequest
	if err := db.First(&cr, request.CRID).Error; err != nil {
		c.JSON(http.StatusBadRequest, utils.FormatResponse("Invalid CR ID", http.StatusBadRequest, "error", nil))
		return
	}

	// Verify CollaboratorID exists if provided
	if request.CollaboratorID != nil {
		var user models.Users
		if err := db.First(&user, *request.CollaboratorID).Error; err != nil {
			c.JSON(http.StatusBadRequest, utils.FormatResponse("Invalid Collaborator ID", http.StatusBadRequest, "error", nil))
			return
		}
	}

	updates := models.SubTask{
		CRID:           &request.CRID,
		CollaboratorID: request.CollaboratorID,
		TaskName:       request.TaskName,
		DueDate:        request.DueDate,
		Progress:       request.Progress,
		Done:           request.Done,
	}

	if err := db.Model(&models.SubTask{}).Where("id = ?", id).Updates(&updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to update Subtask", http.StatusInternalServerError, "error", nil))
		return
	}

	// Catat perubahan subtask ke Activity
	if request.Progress != currentSubtask.Progress || request.Done != currentSubtask.Done || request.TaskName != currentSubtask.TaskName {
		statusText := "Progress: " + strconv.Itoa(int(request.Progress)) + "%"
		if request.Done {
			statusText = "Selesai (Done)"
		}

		activityLog := models.Activity{
			CRID:       &request.CRID,
			UserID:     &claims.UserID,
			Action:     "Change",
			Activities: "Update subtask '" + request.TaskName + "' -> " + statusText,
		}
		db.Create(&activityLog)
	}

	updatedSubtask, _ := findSubtaskByID(db, id)

	c.JSON(http.StatusOK, utils.FormatResponse("Subtask updated successfully", http.StatusOK, "success", updatedSubtask))
}

// DeleteSubtask godoc
// @Summary Delete Subtask
// @Description Delete subtask by id.
// @Tags Subtask
// @Produce json
// @Param id path int true "Subtask ID"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/subtasks/{id} [delete]
func DeleteSubtask(c *gin.Context) {
	claims, ok := getClaims(c)
	if !ok {
		return
	}

	db, ok := connectDB(c)
	if !ok {
		return
	}

	id, ok := parseSubtaskID(c)
	if !ok {
		return
	}

	subtask, err := findSubtaskByID(db, id)
	if err != nil {
		c.JSON(http.StatusNotFound, utils.FormatResponse("Subtask not found", http.StatusNotFound, "error", nil))
		return
	}

	// Authorization Check
	if claims.Role != "Admin" {
		if claims.Role != "PIC" {
			c.JSON(http.StatusForbidden, utils.FormatResponse("Hanya Admin atau PIC yang dapat menghapus subtask", http.StatusForbidden, "error", nil))
			return
		}
		var cr models.ChangeRequest
		if err := db.First(&cr, subtask.CRID).Error; err != nil || cr.PICID == nil || *cr.PICID != claims.UserID {
			c.JSON(http.StatusForbidden, utils.FormatResponse("Hanya PIC CR ini yang dapat menghapus subtask", http.StatusForbidden, "error", nil))
			return
		}
	}

	if err := db.Delete(subtask).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to delete Subtask", http.StatusInternalServerError, "error", nil))
		return
	}

	c.JSON(http.StatusOK, utils.FormatResponse("Subtask deleted successfully", http.StatusOK, "success", nil))
}
