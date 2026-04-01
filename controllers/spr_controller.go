package controllers

import (
	"MonCR/config"
	"MonCR/models"
	"MonCR/utils"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const defaultSPRStatus = "draft"

type createSPRRequest struct {
	Title                 string  `json:"title" binding:"required"`
	BudgetType            string  `json:"budget_type"`
	BudgetYear            int     `json:"budget_year"`
	ExpenseClassification string  `json:"expense_classification"`
	SPRType               string  `json:"spr_type"`
	ProcurementType       string  `json:"procurement_type"`
	BudgetCode            string  `json:"budget_code"`
	WorkProgram           string  `json:"work_program"`
	RemainingBudget       float64 `json:"remaining_budget"`
	Status                string  `json:"status"`
}

type updateSPRRequest struct {
	Title                 *string  `json:"title"`
	BudgetType            *string  `json:"budget_type"`
	BudgetYear            *int     `json:"budget_year"`
	ExpenseClassification *string  `json:"expense_classification"`
	SPRType               *string  `json:"spr_type"`
	ProcurementType       *string  `json:"procurement_type"`
	BudgetCode            *string  `json:"budget_code"`
	WorkProgram           *string  `json:"work_program"`
	RemainingBudget       *float64 `json:"remaining_budget"`
	Status                *string  `json:"status"`
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

func parseSPRID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid SPR id",
		})
		return 0, false
	}

	return uint(id), true
}

func findSPRByID(db *gorm.DB, id uint) (*models.SPR, error) {
	var spr models.SPR
	if err := db.Preload("Creator").First(&spr, id).Error; err != nil {
		return nil, err
	}

	return &spr, nil
}

// CreateSPR - Endpoint untuk membuat data SPR baru
func CreateSPR(c *gin.Context) {
	var request createSPRRequest

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

	spr := models.SPR{
		Title:                 request.Title,
		BudgetType:            request.BudgetType,
		BudgetYear:            request.BudgetYear,
		ExpenseClassification: request.ExpenseClassification,
		SPRType:               request.SPRType,
		ProcurementType:       request.ProcurementType,
		BudgetCode:            request.BudgetCode,
		WorkProgram:           request.WorkProgram,
		RemainingBudget:       request.RemainingBudget,
		CreatedBy:             claims.UserID,
	}

	if request.Status != "" {
		spr.Status = request.Status
	} else {
		spr.Status = defaultSPRStatus
	}

	if err := db.Create(&spr).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create SPR",
			"details": err.Error(),
		})
		return
	}

	if err := db.Preload("Creator").First(&spr, spr.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to load created SPR",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "SPR created successfully",
		"data":    spr,
	})
}

// GetSPRs - Endpoint untuk mendapatkan semua data SPR
func GetSPRs(c *gin.Context) {
	db, ok := connectDB(c)
	if !ok {
		return
	}

	pagination := utils.ParsePagination(c, 10, 100)

	var total int64
	if err := db.Model(&models.SPR{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to count SPR data",
			"details": err.Error(),
		})
		return
	}

	var sprs []models.SPR
	if err := db.Preload("Creator").Order("id DESC").Offset(pagination.Offset).Limit(pagination.Limit).Find(&sprs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve SPR data",
			"details": err.Error(),
		})
		return
	}

	paginationMeta := utils.BuildPaginationMeta(pagination.Offset, pagination.Limit, len(sprs), total)

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"data":       sprs,
		"pagination": paginationMeta,
	})
}

// GetSPRByID - Endpoint untuk mendapatkan detail SPR berdasarkan ID
func GetSPRByID(c *gin.Context) {
	db, ok := connectDB(c)
	if !ok {
		return
	}

	id, ok := parseSPRID(c)
	if !ok {
		return
	}

	spr, err := findSPRByID(db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "SPR not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve SPR",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    spr,
	})
}

// UpdateSPR - Endpoint untuk mengubah data SPR berdasarkan ID
func UpdateSPR(c *gin.Context) {
	db, ok := connectDB(c)
	if !ok {
		return
	}

	id, ok := parseSPRID(c)
	if !ok {
		return
	}

	spr, err := findSPRByID(db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "SPR not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve SPR",
			"details": err.Error(),
		})
		return
	}

	var request updateSPRRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid input data",
			"details": err.Error(),
		})
		return
	}

	updates := map[string]interface{}{}
	if request.Title != nil {
		updates["title"] = *request.Title
	}
	if request.BudgetType != nil {
		updates["budget_type"] = *request.BudgetType
	}
	if request.BudgetYear != nil {
		updates["budget_year"] = *request.BudgetYear
	}
	if request.ExpenseClassification != nil {
		updates["expense_classification"] = *request.ExpenseClassification
	}
	if request.SPRType != nil {
		updates["spr_type"] = *request.SPRType
	}
	if request.ProcurementType != nil {
		updates["procurement_type"] = *request.ProcurementType
	}
	if request.BudgetCode != nil {
		updates["budget_code"] = *request.BudgetCode
	}
	if request.WorkProgram != nil {
		updates["work_program"] = *request.WorkProgram
	}
	if request.RemainingBudget != nil {
		updates["remaining_budget"] = *request.RemainingBudget
	}
	if request.Status != nil {
		updates["status"] = *request.Status
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "No fields provided for update",
		})
		return
	}

	if err := db.Model(spr).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update SPR",
			"details": err.Error(),
		})
		return
	}

	updatedSPR, err := findSPRByID(db, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to load updated SPR",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "SPR updated successfully",
		"data":    updatedSPR,
	})
}

// DeleteSPR - Endpoint untuk menghapus SPR (Soft Delete dengan GORM)
func DeleteSPR(c *gin.Context) {
	db, ok := connectDB(c)
	if !ok {
		return
	}

	id, ok := parseSPRID(c)
	if !ok {
		return
	}

	spr, err := findSPRByID(db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "SPR not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve SPR",
			"details": err.Error(),
		})
		return
	}

	if err := db.Delete(spr).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete SPR",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "SPR deleted successfully",
	})
}
