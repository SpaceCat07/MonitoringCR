package controllers

import (
	"MonCR/models"
	"MonCR/utils"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
	Keterangan     string    `json:"keterangan" binding:"omitempty"`
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

	assigneeText := "Unassigned"
	if subtask.Collaborator != nil && subtask.Collaborator.Fullname != "" {
		assigneeText = subtask.Collaborator.Fullname
	}
	_ = assigneeText
	if subtask.CRID != nil {
		createChangeActivity(db, *subtask.CRID, claims.UserID, fmt.Sprintf("Added a new subtask: '%s'", subtask.TaskName))
	}

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

	ptrUintEqual := func(a, b *uint) bool {
		if a == nil && b == nil {
			return true
		}
		if a == nil || b == nil {
			return false
		}
		return *a == *b
	}
	toDateStr := func(t time.Time) string {
		return t.Format("2006-01-02")
	}

	isProgressChanged := request.Progress != currentSubtask.Progress
	isDueDateChanged := toDateStr(request.DueDate) != toDateStr(currentSubtask.DueDate)
	isAssigneeChanged := !ptrUintEqual(currentSubtask.CollaboratorID, request.CollaboratorID)
	autoDoneByProgress := isProgressChanged && request.Progress == 100

	// Progress must be monotonic (cannot be decreased).
	if request.Progress < currentSubtask.Progress {
		c.JSON(http.StatusBadRequest, utils.FormatResponse("Progress cannot be decreased. Progress must be equal to or greater than the previous progress.", http.StatusBadRequest, "error", nil))
		return
	}

	// Business rule: when progress reaches 100 in this update, Done is forced to true.
	effectiveDone := request.Done
	if autoDoneByProgress {
		effectiveDone = true
	}

	// Validasi: keterangan wajib jika progress berubah
	if isProgressChanged && strings.TrimSpace(request.Keterangan) == "" {
		c.JSON(http.StatusBadRequest, utils.FormatResponse("Progress reason is required when updating progress", http.StatusBadRequest, "error", nil))
		return
	}

	// Role check for Subtask changes
	role := claims.Role
	if role != "Admin" {
		if role == "Collaborator" {
			// Cek apakah collaborator mencoba melakukan self-assign (ID lama nil, ID baru adalah dirinya sendiri)
			isSelfAssign := currentSubtask.CollaboratorID == nil && request.CollaboratorID != nil && *request.CollaboratorID == claims.UserID

			if !isSelfAssign {
				// Jika bukan self-assign, pastikan subtask ini memang miliknya
				if currentSubtask.CollaboratorID == nil || *currentSubtask.CollaboratorID != claims.UserID {
					c.JSON(http.StatusForbidden, utils.FormatResponse("Hanya Collaborator yang ditugaskan atau self-assign yang dapat mengedit subtask ini", http.StatusForbidden, "error", nil))
					return
				}

				// Jika memang miliknya (bukan sedang self-assign baru), batasi field yang boleh diubah
				if request.TaskName != currentSubtask.TaskName || isDueDateChanged || isAssigneeChanged {
					c.JSON(http.StatusForbidden, utils.FormatResponse("Collaborator hanya dapat mengubah progress dan status selesai", http.StatusForbidden, "error", nil))
					return
				}
			}

			// Collaborator can update progress with any value between 0 and 100.
			if request.Progress > 100 {
				c.JSON(http.StatusBadRequest, utils.FormatResponse("Progress must be between 0 and 100", http.StatusBadRequest, "error", nil))
				return
			}

			// Collaborator dilarang mengubah status Done secara manual.
			// Exception: when progress becomes 100 in the same action, Done may change automatically.
			if request.Done != currentSubtask.Done && !autoDoneByProgress {
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

	// GORM mengabaikan update struct jika nilai boolean bernilai false atau pointer bernilai nil pada .Updates() struct.
	// Kita gunakan map untuk memastikan data primitif terupdate secara eksplisit.
	updateMap := map[string]interface{}{
		"cr_id":           request.CRID,
		"collaborator_id": request.CollaboratorID,
		"task_name":       request.TaskName,
		"due_date":        request.DueDate,
		"progress":        request.Progress,
		"done":            effectiveDone,
	}

	if err := db.Model(&models.SubTask{}).Where("id = ?", id).Updates(updateMap).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to update Subtask", http.StatusInternalServerError, "error", nil))
		return
	}

	// Catat perubahan subtask ke Activity (single consolidated entry per action)
	resolveUserName := func(id *uint) string {
		if id == nil {
			return "Unassigned"
		}
		var u models.Users
		if err := db.Select("fullname").First(&u, *id).Error; err != nil || strings.TrimSpace(u.Fullname) == "" {
			return fmt.Sprintf("User#%d", *id)
		}
		return u.Fullname
	}
	reasonText := strings.TrimSpace(request.Keterangan)
	isDoneChanged := effectiveDone != currentSubtask.Done
	hasNonProgressFieldChanges := request.TaskName != currentSubtask.TaskName || isDueDateChanged || isAssigneeChanged || (isDoneChanged && !autoDoneByProgress)
	shouldLogProgressActivity := isProgressChanged || (reasonText != "" && !hasNonProgressFieldChanges)

	if shouldLogProgressActivity {
		progressLog := fmt.Sprintf("Updated subtask '%s' -> Progress: %d%%", request.TaskName, request.Progress)
		if request.Progress == 100 && effectiveDone {
			progressLog += " (Done)"
		}
		if reasonText != "" {
			progressLog = fmt.Sprintf("%s\n\nKeterangan:\n%s", progressLog, reasonText)
		}
		createChangeActivity(db, request.CRID, claims.UserID, progressLog)
	} else {
		changeParts := make([]string, 0, 4)

		if request.TaskName != currentSubtask.TaskName {
			changeParts = append(changeParts, fmt.Sprintf("Name: '%s' -> '%s'", currentSubtask.TaskName, request.TaskName))
		}

		if isDueDateChanged {
			changeParts = append(changeParts, fmt.Sprintf("Due Date: %s -> %s", toDateStr(currentSubtask.DueDate), toDateStr(request.DueDate)))
		}

		if isAssigneeChanged {
			oldAssignee := resolveUserName(currentSubtask.CollaboratorID)
			newAssignee := resolveUserName(request.CollaboratorID)
			assigneePart := fmt.Sprintf("Assignee: '%s' -> '%s'", oldAssignee, newAssignee)
			if currentSubtask.CollaboratorID == nil && request.CollaboratorID != nil && *request.CollaboratorID == claims.UserID && role == "Collaborator" {
				assigneePart += " (self-assigned)"
			}
			changeParts = append(changeParts, assigneePart)
		}

		if isDoneChanged && !autoDoneByProgress {
			changeParts = append(changeParts, fmt.Sprintf("Done: %t -> %t", currentSubtask.Done, effectiveDone))
		}

		if len(changeParts) > 0 {
			activityLog := fmt.Sprintf("Updated subtask '%s' -> %s", request.TaskName, strings.Join(changeParts, " | "))
			createChangeActivity(db, request.CRID, claims.UserID, activityLog)
		}
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

	if subtask.CRID != nil {
		createChangeActivity(db, *subtask.CRID, claims.UserID, fmt.Sprintf("Deleted subtask: '%s'", subtask.TaskName))
	}

	c.JSON(http.StatusOK, utils.FormatResponse("Subtask deleted successfully", http.StatusOK, "success", nil))
}
