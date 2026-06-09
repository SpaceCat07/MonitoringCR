package controllers

import (
	"MonCR/models"
	"MonCR/utils"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type KPISummaryData struct {
	TotalCR      int64 `json:"total_cr"`
	InProgress   int64 `json:"in_progress"`
	NearDeadline int64 `json:"near_deadline"`
	Overdue      int64 `json:"overdue"`
	Cancel       int64 `json:"cancel"`
	Complete     int64 `json:"complete"`
}

// KPISummary godoc
// @Summary Get KPI summary
// @Description Retrieve dashboard KPI summary for change requests.
// @Tags Dashboard
// @Produce json
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/dashboard/kpi-summary [get]
func KPISummary(c *gin.Context) {
	db, ok := connectDB(c)
	if !ok {
		return
	}

	var summary KPISummaryData

	err := db.Model(&models.ChangeRequest{}).
		Where("deleted_at IS NULL").
		Select(`
            COUNT(*) AS total_cr,
            COUNT(*) FILTER (WHERE change_requests.status = 'IN_PROGRESS') AS in_progress,
            COUNT(*) FILTER (
                WHERE change_requests.status NOT IN ('COMPLETE', 'CANCEL')
                AND change_requests.end_date::date BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL '7 days'
            ) AS near_deadline,
            COUNT(*) FILTER (
                WHERE change_requests.status NOT IN ('COMPLETE', 'CANCEL')
                AND change_requests.end_date::date < CURRENT_DATE
            ) AS overdue,
            COUNT(*) FILTER (WHERE change_requests.status = 'CANCEL') AS cancel,
            COUNT(*) FILTER (WHERE change_requests.status = 'COMPLETE') AS complete
        `).
		Scan(&summary).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse(
			"Failed to load KPI Summary",
			http.StatusInternalServerError,
			"error",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, utils.FormatResponse(
		"KPI summary retrieved successfully",
		http.StatusOK,
		"success",
		summary,
	))
}

type PICStats struct {
	PICID      uint
	Name       string
	InProgress int64
	Under7Days int64
	Count      int64
}

// TopPIC godoc
// @Summary Get top PIC statistics
// @Description Retrieve PIC statistics with pagination.
// @Tags Dashboard
// @Produce json
// @Param offset query int false "Offset (default: 0)"
// @Param limit query int false "Limit (default: 5)"
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/dashboard/top-pic [get]
func TopPIC(c *gin.Context) {
	db, ok := connectDB(c)
	if !ok {
		return
	}

	pagination := utils.ParsePagination(c, 5, 5)

	baseQuery := db.Model(&models.ChangeRequest{}).
		Joins("JOIN users ON users.id = change_requests.pic_id").
		Where("change_requests.deleted_at IS NULL").
		Where("change_requests.pic_id IS NOT NULL")

	var total int64
	if err := baseQuery.
		Distinct("change_requests.pic_id").
		Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse(
			"Failed to count PIC statistics",
			http.StatusInternalServerError,
			"error",
			err.Error(),
		))
		return
	}

	var stats []PICStats
	err := baseQuery.
		Select(`
            change_requests.pic_id AS pic_id,
            users.fullname AS name,
            COUNT(*) FILTER (WHERE change_requests.status = 'IN_PROGRESS') AS in_progress,
            COUNT(*) FILTER (
                WHERE change_requests.status NOT IN ('COMPLETE', 'CANCEL')
                AND change_requests.end_date::date BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL '7 days'
            ) AS under7_days,
            COUNT(*) AS count
        `).
		Group("change_requests.pic_id, users.fullname, users.id").
		Order("count DESC").
		Offset(pagination.Offset).
		Limit(pagination.Limit).
		Scan(&stats).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse(
			"Failed to load PIC statistics",
			http.StatusInternalServerError,
			"error",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, utils.FormatResponse(
		"PIC statistics retrieved successfully",
		http.StatusOK,
		"success",
		gin.H{
			"data":       stats,
			"pagination": utils.BuildPaginationMeta(pagination.Offset, pagination.Limit, len(stats), total),
		},
	))
}

type DueToday struct {
	Title   string
	PICName uint
	Module  string
	Status  string
	Total   uint
}

// DueTodayStats godoc
// @Summary Get CRs due today
// @Description Retrieve change requests that are due today with pagination.
// @Tags Dashboard
// @Produce json
// @Param offset query int false "Offset (default: 0)"
// @Param limit query int false "Limit (default: 5)"
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/dashboard/due-today [get]
func DueTodayStats(c *gin.Context) {
	db, ok := connectDB(c)
	if !ok {
		return
	}

	pagination := utils.ParsePagination(c, 5, 5)

	baseQuery := db.Model(&models.ChangeRequest{}).
		Joins("LEFT JOIN users ON users.id = change_requests.pic_id").
		Where("change_requests.deleted_at IS NULL").
		Where("change_requests.end_date::date = CURRENT_DATE")

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse(
			"Failed to count due today CR",
			http.StatusInternalServerError,
			"error",
			err.Error(),
		))
		return
	}

	var data []DueToday
	err := baseQuery.
		Select(`
            change_requests.title AS title,
            COALESCE(users.fullname, '') AS pic_name,
            change_requests.modul AS module,
            change_requests.status AS status,
            COUNT(*) OVER() AS total
        `).
		Order("change_requests.end_date ASC, change_requests.id DESC").
		Offset(pagination.Offset).
		Limit(pagination.Limit).
		Scan(&data).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse(
			"Failed to load due today CR",
			http.StatusInternalServerError,
			"error",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, utils.FormatResponse(
		"Due today CR retrieved successfully",
		http.StatusOK,
		"success",
		gin.H{
			"data":       data,
			"pagination": utils.BuildPaginationMeta(pagination.Offset, pagination.Limit, len(data), total),
		},
	))
}

type ModuleCategoryStat struct {
	Module   string `json:"module"`
	Category string `json:"category"`
	Total    int64  `json:"total"`
}

// ModulevsCategory godoc
// @Summary Get module vs category statistics
// @Description Retrieve distribution of change requests by module and category.
// @Tags Dashboard
// @Produce json
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/dashboard/module-category [get]
func ModulevsCategory(c *gin.Context) {
	db, ok := connectDB(c)
	if !ok {
		return
	}

	var stats []ModuleCategoryStat

	err := db.Model(&models.ChangeRequest{}).
		Where("deleted_at IS NULL").
		Select(`
            modul AS module,
            category AS category,
            COUNT(*) AS total
        `).
		Group("modul, category").
		Order("modul ASC, category ASC").
		Scan(&stats).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse(
			"Failed to load module vs category statistics",
			http.StatusInternalServerError,
			"error",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, utils.FormatResponse(
		"Module vs category statistics retrieved successfully",
		http.StatusOK,
		"success",
		stats,
	))
}

type ModuleStatusStats struct {
	Module string `json:"module"`
	Status string `json:"status"`
	Total  int64  `json:"total"`
}

// ModulevsStatus godoc
// @Summary Get module vs status statistics
// @Description Retrieve distribution of change requests by module and status.
// @Tags Dashboard
// @Produce json
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/dashboard/module-status [get]
func ModulevsStatus(c *gin.Context) {
	db, ok := connectDB(c)
	if !ok {
		return
	}

	var stats []ModuleStatusStats

	err := db.Model(&models.ChangeRequest{}).
		Where("deleted_at IS NULL").
		Select(`
            modul AS module,
            status AS status,
            COUNT(*) AS total
        `).
		Group("modul, status").
		Order("modul ASC, status ASC").
		Scan(&stats).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse(
			"Failed to load module vs status statistics",
			http.StatusInternalServerError,
			"error",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, utils.FormatResponse(
		"Module vs status statistics retrieved successfully",
		http.StatusOK,
		"success",
		stats,
	))
}

type ModuleHealthOverview struct {
	Module            string  `json:"module"`
	OverduePct        float64 `json:"overdue_pct"`
	CompletionRatePct float64 `json:"completion_rate_pct"`
	HealthStatus      string  `json:"health_status"`
	TotalCR           int64   `json:"total_cr"`
	OverdueCount      int64   `json:"overdue_count"`
	CompleteCount     int64   `json:"complete_count"`
	CancelCount       int64   `json:"cancel_count"`
}

type moduleHealthOverviewRow struct {
	Module           string `gorm:"column:module"`
	TotalCR          int64  `gorm:"column:total_cr"`
	OverdueCount     int64  `gorm:"column:overdue_count"`
	OverdueBaseCount int64  `gorm:"column:overdue_base_count"`
	CompleteCount    int64  `gorm:"column:complete_count"`
	CancelCount      int64  `gorm:"column:cancel_count"`
}

func calculateHealthStatus(overduePct, completionRatePct float64) string {
	switch {
	case completionRatePct == 100 && overduePct == 0:
		return "Baik"
	case completionRatePct >= 60 && overduePct <= 20:
		return "Cukup"
	case completionRatePct >= 30 && overduePct <= 50:
		return "Buruk"
	default:
		return "Buruk"
	}
}

// ModuleHealthOverviewStats godoc
// @Summary Get module health overview
// @Description Retrieve module health metrics including overdue and completion rates.
// @Tags Dashboard
// @Produce json
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/dashboard/module-health-overview [get]
func ModuleHealthOverviewStats(c *gin.Context) {
	db, ok := connectDB(c)
	if !ok {
		return
	}

	var rows []moduleHealthOverviewRow

	err := db.Model(&models.ChangeRequest{}).
		Where("deleted_at IS NULL").
		Select(`
            modul AS module,
            COUNT(*) AS total_cr,
            COUNT(*) FILTER (
                WHERE status NOT IN ('COMPLETE', 'CANCEL', 'DRAFT')
                AND end_date::date < CURRENT_DATE
            ) AS overdue_count,
            COUNT(*) FILTER (
                WHERE status NOT IN ('COMPLETE', 'CANCEL', 'DRAFT')
            ) AS overdue_base_count,
            COUNT(*) FILTER (WHERE status = 'COMPLETE') AS complete_count,
            COUNT(*) FILTER (WHERE status = 'CANCEL') AS cancel_count
        `).
		Group("modul").
		Order("modul ASC").
		Scan(&rows).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse(
			"Failed to load module health overview",
			http.StatusInternalServerError,
			"error",
			err.Error(),
		))
		return
	}

	data := make([]ModuleHealthOverview, 0, len(rows))
	for _, row := range rows {
		completionRatePct := 0.0
		if row.TotalCR > 0 {
			completionRatePct = math.Round((float64(row.CompleteCount)/float64(row.TotalCR))*100*100) / 100
		}

		overduePct := 0.0
		if row.OverdueBaseCount > 0 {
			overduePct = math.Round((float64(row.OverdueCount)/float64(row.OverdueBaseCount))*100*100) / 100
		}

		data = append(data, ModuleHealthOverview{
			Module:            row.Module,
			OverduePct:        overduePct,
			CompletionRatePct: completionRatePct,
			HealthStatus:      calculateHealthStatus(overduePct, completionRatePct),
			TotalCR:           row.TotalCR,
			OverdueCount:      row.OverdueCount,
			CompleteCount:     row.CompleteCount,
			CancelCount:       row.CancelCount,
		})
	}

	c.JSON(http.StatusOK, utils.FormatResponse(
		"Module health overview retrieved successfully",
		http.StatusOK,
		"success",
		data,
	))
}

type LifecycleSeries struct {
	Name string  `json:"name"`
	Data []int64 `json:"data"`
}

// LifecycleLineChart godoc
// @Summary Get lifecycle line chart
// @Description Retrieve monthly lifecycle counts for Issued, In Progress, and Complete.
// @Tags Dashboard
// @Produce json
// @Param year query int false "Target year (default: current year)"
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/dashboard/lifecycle-line-chart [get]
func LifecycleLineChart(c *gin.Context) {
	db, ok := connectDB(c)
	if !ok {
		return
	}

	targetYear := time.Now().Year()
	if yearStr := c.Query("year"); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			targetYear = y
		}
	}

	var records []models.ChangeRequest
	err := db.
		Preload("Activities", func(db *gorm.DB) *gorm.DB {
			return db.Where("activities.deleted_at IS NULL").Order("activities.created_at ASC")
		}).
		Where("change_requests.deleted_at IS NULL").
		Find(&records).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse(
			"Failed to load lifecycle chart data",
			http.StatusInternalServerError,
			"error",
			err.Error(),
		))
		return
	}

	monthLabels := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	issuedData := make([]int64, 12)
	inProgressData := make([]int64, 12)
	completeData := make([]int64, 12)

	for _, r := range records {
		var issuedDate, inProgressDate, completeDate time.Time

		for _, act := range r.Activities {
			if act.Action != "Change" {
				continue
			}

			if strings.HasSuffix(act.Activities, "ke ISSUED") && issuedDate.IsZero() {
				issuedDate = act.CreatedAt
			} else if strings.HasSuffix(act.Activities, "ke IN_PROGRESS") && inProgressDate.IsZero() {
				inProgressDate = act.CreatedAt
			} else if strings.HasSuffix(act.Activities, "ke COMPLETE") && completeDate.IsZero() {
				completeDate = act.CreatedAt
			}
		}

		if !issuedDate.IsZero() && issuedDate.Year() == targetYear {
			monthIdx := int(issuedDate.Month()) - 1
			if monthIdx >= 0 && monthIdx < 12 {
				issuedData[monthIdx]++
			}
		}

		if !inProgressDate.IsZero() && inProgressDate.Year() == targetYear {
			monthIdx := int(inProgressDate.Month()) - 1
			if monthIdx >= 0 && monthIdx < 12 {
				inProgressData[monthIdx]++
			}
		}

		if !completeDate.IsZero() && completeDate.Year() == targetYear {
			monthIdx := int(completeDate.Month()) - 1
			if monthIdx >= 0 && monthIdx < 12 {
				completeData[monthIdx]++
			}
		}
	}

	c.JSON(http.StatusOK, utils.FormatResponse(
		"Lifecycle chart retrieved successfully",
		http.StatusOK,
		"success",
		gin.H{
			"labels": monthLabels,
			"series": []LifecycleSeries{
				{Name: "ISSUED", Data: issuedData},
				{Name: "IN PROGRESS", Data: inProgressData},
				{Name: "COMPLETE", Data: completeData},
			},
		},
	))
}