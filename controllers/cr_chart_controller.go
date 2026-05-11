package controllers

import (
	"MonCR/models"
	"MonCR/utils"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type chartBucket struct {
	Label string `json:"label"`
	Value int64  `json:"value"`
}

type chartSeries struct {
	Name string  `json:"name"`
	Data []int64 `json:"data"`
}

type ContributorStat struct {
	Name       string `json:"name"`
	InProgress int    `json:"inProgress"`
	Under7Days int    `json:"under7Days"`
	Count      int    `json:"count"`
}

func parseChartDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}

	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t, nil
	}

	return time.Parse(time.RFC3339, value)
}

func applyChartFilters(c *gin.Context, db *gorm.DB) (*gorm.DB, bool) {
	status := strings.TrimSpace(c.Query("status"))
	category := strings.TrimSpace(c.Query("category"))
	module := strings.TrimSpace(c.Query("modul"))
	from := strings.TrimSpace(c.Query("from"))
	to := strings.TrimSpace(c.Query("to"))

	if status != "" {
		db = db.Where("status = ?", status)
	}

	if category != "" {
		db = db.Where("category = ?", category)
	}

	if module != "" {
		db = db.Where("modul = ?", module)
	}

	if from != "" {
		fromDate, err := parseChartDate(from)
		if err != nil {
			c.JSON(http.StatusBadRequest, utils.FormatResponse("Invalid from date. Use YYYY-MM-DD or RFC3339", http.StatusBadRequest, "error", nil))
			return nil, false
		}
		db = db.Where("release_date >= ?", fromDate)
	}

	if to != "" {
		toDate, err := parseChartDate(to)
		if err != nil {
			c.JSON(http.StatusBadRequest, utils.FormatResponse("Invalid to date. Use YYYY-MM-DD or RFC3339", http.StatusBadRequest, "error", nil))
			return nil, false
		}
		db = db.Where("release_date <= ?", toDate)
	}

	return db, true
}

func bucketsByOptions(options []string, counts map[string]int64) []chartBucket {
	buckets := make([]chartBucket, 0, len(options))
	for _, option := range options {
		buckets = append(buckets, chartBucket{Label: option, Value: counts[option]})
	}
	return buckets
}

func sortedMonthBuckets(counts map[string]int64) []chartBucket {
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	buckets := make([]chartBucket, 0, len(keys))
	for _, key := range keys {
		buckets = append(buckets, chartBucket{Label: key, Value: counts[key]})
	}
	return buckets
}

// GetCRCharts godoc
// @Summary Get CR chart analytics
// @Description Return chart-ready data for pie and bar charts. Optional filters: status, category, modul, from, to.
// @Tags CR
// @Produce json
// @Param status query string false "Filter by status"
// @Param category query string false "Filter by category"
// @Param modul query string false "Filter by modul"
// @Param from query string false "Start date (YYYY-MM-DD or RFC3339)"
// @Param to query string false "End date (YYYY-MM-DD or RFC3339)"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/cr/charts [get]
func GetCRCharts(c *gin.Context) {
	db, ok := connectDB(c)
	if !ok {
		return
	}

	filteredDB, ok := applyChartFilters(c, db.Model(&models.ChangeRequest{}))
	if !ok {
		return
	}

	var records []models.ChangeRequest
	if err := filteredDB.Select("id", "modul", "category", "status", "release_date", "end_date", "pic_id").Preload("PIC").Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utils.FormatResponse("Failed to load chart data", http.StatusInternalServerError, "error", err.Error()))
		return
	}

	statusCounts := make(map[string]int64)
	categoryCounts := make(map[string]int64)
	moduleCounts := make(map[string]int64)
	monthCounts := make(map[string]int64)
	
	// Pisahkan map untuk status dan category
	statusByModule := make(map[string]map[string]int64)
	categoryByModule := make(map[string]map[string]int64) // TAMBAHAN: Map khusus category

	contributorMap := make(map[string]*ContributorStat)
	now := time.Now()
	sevenDaysFromNow := now.Add(7 * 24 * time.Hour)

	var activeCount int64
	for _, r := range records {
		statusCounts[r.Status]++
		categoryCounts[r.Category]++
		moduleCounts[r.Modul]++

		monthKey := r.ReleaseDate.Format("2006-01")
		monthCounts[monthKey]++

		// 1. Populate statusByModule
		if _, exists := statusByModule[r.Modul]; !exists {
			statusByModule[r.Modul] = map[string]int64{}
		}
		statusByModule[r.Modul][r.Status]++

		// 2. Populate categoryByModule (TAMBAHAN LOGIKA)
		if _, exists := categoryByModule[r.Modul]; !exists {
			categoryByModule[r.Modul] = map[string]int64{}
		}
		categoryByModule[r.Modul][r.Category]++

		if r.Status != "COMPLETE" && r.Status != "CANCEL" {
			activeCount++
		}

		// 3. Populate contributors
		if r.PIC != nil && r.PIC.Fullname != "" {
			name := r.PIC.Fullname
			if _, exists := contributorMap[name]; !exists {
				contributorMap[name] = &ContributorStat{Name: name}
			}
			stat := contributorMap[name]
			stat.Count++

			if r.Status == "IN_PROGRESS" {
				stat.InProgress++
			}

			if r.Status != "COMPLETE" && r.Status != "CANCEL" {
				if r.EndDate.Before(sevenDaysFromNow) {
					stat.Under7Days++
				}
			}
		}
	}

	moduleLabels := make([]string, 0, len(moduleOptions))
	for _, module := range moduleOptions {
		moduleLabels = append(moduleLabels, module)
	}

	// Series untuk Stacked Status
	stackedSeries := make([]chartSeries, 0, len(statusOptions))
	for _, status := range statusOptions {
		data := make([]int64, 0, len(moduleLabels))
		for _, module := range moduleLabels {
			data = append(data, statusByModule[module][status])
		}
		stackedSeries = append(stackedSeries, chartSeries{Name: status, Data: data})
	}

	// Series untuk Stacked Category (PERBAIKAN LOGIKA)
	stackedSeries2 := make([]chartSeries, 0, len(categoryOptions))
	for _, category := range categoryOptions {
		data := make([]int64, 0, len(moduleLabels))
		for _, module := range moduleLabels {
			// Gunakan categoryByModule, bukan statusByModule
			data = append(data, categoryByModule[module][category]) 
		}
		stackedSeries2 = append(stackedSeries2, chartSeries{Name: category, Data: data})
	}

	var contributors []ContributorStat
	for _, stat := range contributorMap {
		contributors = append(contributors, *stat)
	}
	sort.Slice(contributors, func(i, j int) bool {
		return contributors[i].Count > contributors[j].Count
	})
	if len(contributors) > 5 {
		contributors = contributors[:5]
	}

	total := int64(len(records))

	c.JSON(http.StatusOK, utils.FormatResponse("Chart data retrieved successfully", http.StatusOK, "success", gin.H{
		"summary": gin.H{
			"total":    total,
			"active":   activeCount,
			"complete": statusCounts["COMPLETE"],
			"cancel":   statusCounts["CANCEL"],
			"completion": func() float64 {
				if total == 0 {
					return 0
				}
				return float64(statusCounts["COMPLETE"]) / float64(total) * 100
			}(),
		},
		"pie": gin.H{
			"by_status":   bucketsByOptions(statusOptions, statusCounts),
			"by_category": bucketsByOptions(categoryOptions, categoryCounts),
			"by_modul":    bucketsByOptions(moduleOptions, moduleCounts),
		},
		"bar": gin.H{
			"by_month": sortedMonthBuckets(monthCounts),
			"stacked_status_by_modul": gin.H{
				"labels": moduleLabels,
				"series": stackedSeries,
			},
			"stacked_category_by_modul": gin.H{
				"labels": moduleLabels,
				"series": stackedSeries2,
			},
		},
		"top_contributors": contributors,
	}))
}
