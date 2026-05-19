package controllers

import (
	"MonCR/models"
	"MonCR/utils"
	"net/http"
	"sort"
	"strconv"
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

	// 1. FILTER SOFT DELETE & PRELOAD ACTIVITIES
	// Memastikan CR tidak terhapus dan mengurutkan log Activity dari yang paling lama ke baru
	err := filteredDB.
		Where("change_requests.deleted_at IS NULL"). 
		Preload("PIC").
		Preload("Activities", func(db *gorm.DB) *gorm.DB {
			return db.Where("activities.deleted_at IS NULL").Order("activities.created_at ASC")
		}).
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
	// PARAMETER TAHUN UNTUK LINE CHART
	// ======================================
	targetYear := time.Now().Year()
	if yearStr := c.Query("year"); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			targetYear = y
		}
	}

	// ======================================
	// COUNTERS
	// ======================================

	statusCounts := make(map[string]int64)
	categoryCounts := make(map[string]int64)
	moduleCounts := make(map[string]int64)

	statusByModule := make(map[string]map[string]int64)
	categoryByModule := make(map[string]map[string]int64)

	contributorMap := make(map[string]*ContributorStat)

	now := time.Now()
	sevenDaysFromNow := now.Add(7 * 24 * time.Hour)

	var activeCount int64

	// BUCKET 12 BULAN
	monthLabels := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	issuedData := make([]int64, 12)
	inProgressData := make([]int64, 12)
	completeData := make([]int64, 12)

	// ======================================
	// LOOP DATA
	// ======================================

	for _, r := range records {

		// PIE CHART COUNTER
		statusCounts[r.Status]++
		categoryCounts[r.Category]++
		moduleCounts[r.Modul]++

		// ======================================
		// 2. LIFECYCLE DARI TABEL ACTIVITY
		// ======================================
		var issuedDate, inProgressDate, completeDate time.Time

		for _, act := range r.Activities {
			if act.Action == "Change" {
				// Menggunakan HasSuffix agar string seperti "dari ISSUED ke DRAFT" tidak keliru terbaca
				// dan menggunakan IsZero agar kita mendapatkan tanggal PERTAMA KALI dipindahkan ke status tersebut.
				if strings.HasSuffix(act.Activities, "ke ISSUED") && issuedDate.IsZero() {
					issuedDate = act.CreatedAt
				} else if strings.HasSuffix(act.Activities, "ke IN_PROGRESS") && inProgressDate.IsZero() {
					inProgressDate = act.CreatedAt
				} else if strings.HasSuffix(act.Activities, "ke COMPLETE") && completeDate.IsZero() {
					completeDate = act.CreatedAt
				}
			}
		}

		// Jika tanggal ditemukan dan tahunnya sesuai dengan filter pencarian
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

		// ======================================
		// STACKED STATUS & CATEGORY BY MODULE
		// ======================================

		if _, exists := statusByModule[r.Modul]; !exists {
			statusByModule[r.Modul] = map[string]int64{}
		}
		statusByModule[r.Modul][r.Status]++

		if _, exists := categoryByModule[r.Modul]; !exists {
			categoryByModule[r.Modul] = map[string]int64{}
		}
		categoryByModule[r.Modul][r.Category]++

		// ======================================
		// ACTIVE COUNT & CONTRIBUTORS
		// ======================================

		if r.Status != "COMPLETE" && r.Status != "CANCEL" {
			activeCount++
		}

		if r.PIC != nil && r.PIC.Fullname != "" {
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

			if r.Status != "COMPLETE" && r.Status != "CANCEL" {
				if r.EndDate.Before(sevenDaysFromNow) {
					stat.Under7Days++
				}
			}
		}
	}

	// ======================================
	// STACKED SERIES FORMULATION
	// ======================================

	moduleLabels := make([]string, 0, len(moduleOptions))
	for _, module := range moduleOptions {
		moduleLabels = append(moduleLabels, module)
	}

	stackedSeries := make([]chartSeries, 0, len(statusOptions))
	for _, status := range statusOptions {
		data := make([]int64, 0, len(moduleLabels))
		for _, module := range moduleLabels {
			data = append(data, statusByModule[module][status])
		}
		stackedSeries = append(stackedSeries, chartSeries{Name: status, Data: data})
	}

	stackedSeries2 := make([]chartSeries, 0, len(categoryOptions))
	for _, category := range categoryOptions {
		data := make([]int64, 0, len(moduleLabels))
		for _, module := range moduleLabels {
			data = append(data, categoryByModule[module][category])
		}
		stackedSeries2 = append(stackedSeries2, chartSeries{Name: category, Data: data})
	}

	// ======================================
	// TOP CONTRIBUTORS SORTING
	// ======================================

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
				"line": gin.H{
					"lifecycle": gin.H{
						"labels": monthLabels,
						"series": []chartSeries{
							{Name: "Issued", Data: issuedData},
							{Name: "Complete", Data: completeData},
							{Name: "In Progress", Data: inProgressData},
						},
					},
				},
				"bar": gin.H{
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
			},
		),
	)
}