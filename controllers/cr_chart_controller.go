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

// ======================================
// FILTER CHART
// ======================================

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

	// ======================================
	// FILTER UPDATED_AT FROM
	// ======================================

	if from != "" {

		fromDate, err := parseChartDate(from)

		if err != nil {

			c.JSON(
				http.StatusBadRequest,
				utils.FormatResponse(
					"Invalid from date",
					http.StatusBadRequest,
					"error",
					nil,
				),
			)

			return nil, false
		}

		db = db.Where("updated_at >= ?", fromDate)
	}

	// ======================================
	// FILTER UPDATED_AT TO
	// ======================================

	if to != "" {

		toDate, err := parseChartDate(to)

		if err != nil {

			c.JSON(
				http.StatusBadRequest,
				utils.FormatResponse(
					"Invalid to date",
					http.StatusBadRequest,
					"error",
					nil,
				),
			)

			return nil, false
		}

		db = db.Where("updated_at <= ?", toDate)
	}

	return db, true
}

func bucketsByOptions(options []string, counts map[string]int64) []chartBucket {

	buckets := make([]chartBucket, 0, len(options))

	for _, option := range options {

		buckets = append(buckets, chartBucket{
			Label: option,
			Value: counts[option],
		})
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

		buckets = append(buckets, chartBucket{
			Label: key,
			Value: counts[key],
		})
	}

	return buckets
}

// ======================================
// GET CR CHARTS
// ======================================

func GetCRCharts(c *gin.Context) {

	db, ok := connectDB(c)

	if !ok {
		return
	}

	filteredDB, ok := applyChartFilters(
		c,
		db.Model(&models.ChangeRequest{}),
	)

	if !ok {
		return
	}

	var records []models.ChangeRequest

	err := filteredDB.
		Preload("PIC").
		Find(&records).Error

	if err != nil {

		c.JSON(
			http.StatusInternalServerError,
			utils.FormatResponse(
				"Failed to load chart data",
				http.StatusInternalServerError,
				"error",
				err.Error(),
			),
		)

		return
	}

	// ======================================
	// COUNTERS
	// ======================================

	statusCounts := make(map[string]int64)
	categoryCounts := make(map[string]int64)
	moduleCounts := make(map[string]int64)

	issuedCounts := make(map[string]int64)
	inProgressCounts := make(map[string]int64)
	completeCounts := make(map[string]int64)

	statusByModule := make(map[string]map[string]int64)
	categoryByModule := make(map[string]map[string]int64)

	contributorMap := make(map[string]*ContributorStat)

	now := time.Now()
	sevenDaysFromNow := now.Add(7 * 24 * time.Hour)

	var activeCount int64

	// ======================================
	// LOOP DATA
	// ======================================

	for _, r := range records {

		// ======================================
		// PIE CHART COUNTER
		// ======================================

		statusCounts[r.Status]++
		categoryCounts[r.Category]++
		moduleCounts[r.Modul]++

		// ======================================
		// LIFECYCLE CHART
		// ======================================

		// ISSUED
		if r.Status == "ISSUED" {

			month := r.UpdatedAt.Format("2006-01")

			issuedCounts[month]++
		}

		// IN PROGRESS
		if r.Status == "IN_PROGRESS" {

			month := r.UpdatedAt.Format("2006-01")

			inProgressCounts[month]++
		}

		// COMPLETE
		if r.Status == "COMPLETE" {

			month := r.UpdatedAt.Format("2006-01")

			completeCounts[month]++
		}

		// ======================================
		// STACKED STATUS BY MODULE
		// ======================================

		if _, exists := statusByModule[r.Modul]; !exists {

			statusByModule[r.Modul] = map[string]int64{}
		}

		statusByModule[r.Modul][r.Status]++

		// ======================================
		// STACKED CATEGORY BY MODULE
		// ======================================

		if _, exists := categoryByModule[r.Modul]; !exists {

			categoryByModule[r.Modul] = map[string]int64{}
		}

		categoryByModule[r.Modul][r.Category]++

		// ======================================
		// ACTIVE COUNT
		// ======================================

		if r.Status != "COMPLETE" &&
			r.Status != "CANCEL" {

			activeCount++
		}

		// ======================================
		// TOP CONTRIBUTORS
		// ======================================

		if r.PIC != nil &&
			r.PIC.Fullname != "" {

			name := r.PIC.Fullname

			if _, exists := contributorMap[name]; !exists {

				contributorMap[name] = &ContributorStat{
					Name: name,
				}
			}

			stat := contributorMap[name]

			stat.Count++

			if r.Status == "IN_PROGRESS" {
				stat.InProgress++
			}

			if r.Status != "COMPLETE" &&
				r.Status != "CANCEL" {

				if r.EndDate.Before(sevenDaysFromNow) {
					stat.Under7Days++
				}
			}
		}
	}

	// ======================================
	// STACKED STATUS SERIES
	// ======================================

	moduleLabels := make([]string, 0, len(moduleOptions))

	for _, module := range moduleOptions {

		moduleLabels = append(moduleLabels, module)
	}

	stackedSeries := make([]chartSeries, 0, len(statusOptions))

	for _, status := range statusOptions {

		data := make([]int64, 0, len(moduleLabels))

		for _, module := range moduleLabels {

			data = append(
				data,
				statusByModule[module][status],
			)
		}

		stackedSeries = append(
			stackedSeries,
			chartSeries{
				Name: status,
				Data: data,
			},
		)
	}

	// ======================================
	// STACKED CATEGORY SERIES
	// ======================================

	stackedSeries2 := make([]chartSeries, 0, len(categoryOptions))

	for _, category := range categoryOptions {

		data := make([]int64, 0, len(moduleLabels))

		for _, module := range moduleLabels {

			data = append(
				data,
				categoryByModule[module][category],
			)
		}

		stackedSeries2 = append(
			stackedSeries2,
			chartSeries{
				Name: category,
				Data: data,
			},
		)
	}

	// ======================================
	// TOP CONTRIBUTORS
	// ======================================

	var contributors []ContributorStat

	for _, stat := range contributorMap {

		contributors = append(
			contributors,
			*stat,
		)
	}

	sort.Slice(contributors, func(i, j int) bool {

		return contributors[i].Count >
			contributors[j].Count
	})

	if len(contributors) > 5 {
		contributors = contributors[:5]
	}

	total := int64(len(records))

	// ======================================
	// RESPONSE
	// ======================================

	c.JSON(
		http.StatusOK,
		utils.FormatResponse(
			"Chart data retrieved successfully",
			http.StatusOK,
			"success",
			gin.H{

				// ======================================
				// SUMMARY
				// ======================================

				"summary": gin.H{
					"total":    total,
					"active":   activeCount,
					"complete": statusCounts["COMPLETE"],
					"cancel":   statusCounts["CANCEL"],

					"completion": func() float64 {

						if total == 0 {
							return 0
						}

						return float64(statusCounts["COMPLETE"]) /
							float64(total) * 100
					}(),
				},

				// ======================================
				// PIE CHART
				// ======================================

				"pie": gin.H{

					"by_status": bucketsByOptions(
						statusOptions,
						statusCounts,
					),

					"by_category": bucketsByOptions(
						categoryOptions,
						categoryCounts,
					),

					"by_modul": bucketsByOptions(
						moduleOptions,
						moduleCounts,
					),
				},

				// ======================================
				// BAR CHART
				// ======================================

				"bar": gin.H{

					// ======================================
					// LIFECYCLE
					// ======================================

					"lifecycle": gin.H{

						"issued": sortedMonthBuckets(
							issuedCounts,
						),

						"in_progress": sortedMonthBuckets(
							inProgressCounts,
						),

						"complete": sortedMonthBuckets(
							completeCounts,
						),
					},

					// ======================================
					// STACKED STATUS
					// ======================================

					"stacked_status_by_modul": gin.H{
						"labels": moduleLabels,
						"series": stackedSeries,
					},

					// ======================================
					// STACKED CATEGORY
					// ======================================

					"stacked_category_by_modul": gin.H{
						"labels": moduleLabels,
						"series": stackedSeries2,
					},
				},

				// ======================================
				// TOP CONTRIBUTORS
				// ======================================

				"top_contributors": contributors,
			},
		),
	)
}