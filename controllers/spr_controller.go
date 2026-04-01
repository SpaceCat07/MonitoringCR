package controllers

import (
	"MonCR/config"
	"MonCR/models"
	"MonCR/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateSPR - Endpoint untuk membuat data SPR baru
func CreateSPR(c *gin.Context) {
	var request struct {
		Title                 string  `json:"title" binding:"required"`
		BudgetType            string  `json:"budget_type"`
		BudgetYear            int     `json:"budget_year"`
		ExpenseClassification string  `json:"expense_classification"`
		SPRType               string  `json:"spr_type"`
		ProcurementType       string  `json:"procurement_type"`
		BudgetCode            string  `json:"budget_code"`
		WorkProgram           string  `json:"work_program"`
		RemainingBudget       float64 `json:"remaining_budget"`
		Status                string  `json:"status"` // Optional, default 'draft' is handled by DB
	}

	db, err := config.DBConnect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to database"})
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

	claimsValue, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Unauthorized",
		})
		return
	}

	claims, ok := claimsValue.(*utils.Claims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Invalid token claims",
		})
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
		spr.Status = "draft"
	}

	if err := db.Create(&spr).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create SPR",
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
	db, err := config.DBConnect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to database"})
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
		"success": true,
		"data":    sprs,
		"pagination": paginationMeta,
	})
}

// GetSPRByID - Endpoint untuk mendapatkan detail SPR berdasarkan ID
func GetSPRByID(c *gin.Context) {
	id := c.Param("id")

	db, err := config.DBConnect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to database"})
		return
	}

	var spr models.SPR
	if err := db.First(&spr, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "SPR not found",
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
	id := c.Param("id")

	db, err := config.DBConnect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to database"})
		return
	}

	var spr models.SPR
	if err := db.First(&spr, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "SPR not found",
		})
		return
	}

	var request struct {
		Title                 string  `json:"title"`
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

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid input data",
			"details": err.Error(),
		})
		return
	}

	// Update fields selectively if they are provided
	if request.Title != "" {
		spr.Title = request.Title
	}
	if request.BudgetType != "" {
		spr.BudgetType = request.BudgetType
	}
	if request.BudgetYear != 0 {
		spr.BudgetYear = request.BudgetYear
	}
	if request.ExpenseClassification != "" {
		spr.ExpenseClassification = request.ExpenseClassification
	}
	if request.SPRType != "" {
		spr.SPRType = request.SPRType
	}
	if request.ProcurementType != "" {
		spr.ProcurementType = request.ProcurementType
	}
	if request.BudgetCode != "" {
		spr.BudgetCode = request.BudgetCode
	}
	if request.WorkProgram != "" {
		spr.WorkProgram = request.WorkProgram
	}
	if request.RemainingBudget != 0 {
		spr.RemainingBudget = request.RemainingBudget
	}
	if request.Status != "" {
		spr.Status = request.Status
	}

	if err := db.Save(&spr).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update SPR",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "SPR updated successfully",
		"data":    spr,
	})
}

// DeleteSPR - Endpoint untuk menghapus SPR (Soft Delete dengan GORM)
func DeleteSPR(c *gin.Context) {
	id := c.Param("id")

	db, err := config.DBConnect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to database"})
		return
	}

	var spr models.SPR
	if err := db.First(&spr, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "SPR not found",
		})
		return
	}

	if err := db.Delete(&spr).Error; err != nil {
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
