package controllers

import (
	"MonCR/models"
	"MonCR/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type LazyCollaborator struct {
    ID       uint   `json:"id"`
    Fullname string `json:"fullname"`
    Email    string `json:"email"`
}

type LazyPIC struct {
    ID            uint              `json:"id"`
    Fullname      string            `json:"fullname"`
    Collaborators []LazyCollaborator `json:"collaborators"`
}

type LazyCRItem struct {
    ID       uint      `json:"id"`
    Title    string    `json:"title"`
    EndDate  time.Time `json:"end_date"`
    Module   string    `json:"module"`
    Category string    `json:"category"`
    PIC      *LazyPIC  `json:"pic,omitempty"`
}

func getCRLazyByStatus(c *gin.Context, status string) {
    claims, ok := getClaims(c)
    if !ok {
        return
    }

    db, ok := connectDB(c)
    if !ok {
        return
    }

    pagination := utils.ParsePagination(c, 10, 10)

    query := db.Model(&models.ChangeRequest{}).
        Where("change_requests.deleted_at IS NULL")

    // Role-based visibility
    if claims.Role == "PIC" {
        query = query.Where("change_requests.pic_id = ?", claims.UserID)
    } else if claims.Role == "Collaborator" {
        var user models.Users
        if err := db.Select("parent_pic").First(&user, claims.UserID).Error; err != nil {
            c.JSON(http.StatusInternalServerError, utils.FormatResponse(
                "Failed to resolve collaborator parent PIC",
                http.StatusInternalServerError,
                "error",
                err.Error(),
            ))
            return
        }

        if user.ParentPIC == nil {
            c.JSON(http.StatusOK, utils.FormatResponse(
                "CR retrieved successfully",
                http.StatusOK,
                "success",
                gin.H{
                    "items":      []LazyCRItem{},
                    "pagination": utils.BuildPaginationMeta(pagination.Offset, pagination.Limit, 0, 0),
                },
            ))
            return
        }

        query = query.Where("change_requests.pic_id = ?", *user.ParentPIC)
    }

    if status != "" {
        query = query.Where("change_requests.status = ?", status)
    }

    var total int64
    if err := query.Count(&total).Error; err != nil {
        c.JSON(http.StatusInternalServerError, utils.FormatResponse(
            "Failed to count CR data",
            http.StatusInternalServerError,
            "error",
            err.Error(),
        ))
        return
    }

    var crs []models.ChangeRequest
    if err := query.
        Select("id, title, end_date, modul, category, pic_id").
        Preload("PIC", func(db *gorm.DB) *gorm.DB {
            return db.Select("id, fullname")
        }).
        Order("change_requests.end_date ASC, change_requests.id DESC").
        Offset(pagination.Offset).
        Limit(pagination.Limit).
        Find(&crs).Error; err != nil {
        c.JSON(http.StatusInternalServerError, utils.FormatResponse(
            "Failed to load CR data",
            http.StatusInternalServerError,
            "error",
            err.Error(),
        ))
        return
    }

    picIDs := make([]uint, 0, len(crs))
    for _, cr := range crs {
        if cr.PICID != nil {
            picIDs = append(picIDs, *cr.PICID)
        }
    }

    collabsByPIC := make(map[uint][]LazyCollaborator)
    if len(picIDs) > 0 {
        var collabs []models.Users
        if err := db.
            Select("id, fullname, email, parent_pic").
            Where("role = ?", "Collaborator").
            Where("parent_pic IN ?", picIDs).
            Order("fullname ASC").
            Find(&collabs).Error; err != nil {
            c.JSON(http.StatusInternalServerError, utils.FormatResponse(
                "Failed to load collaborators",
                http.StatusInternalServerError,
                "error",
                err.Error(),
            ))
            return
        }

        for _, collab := range collabs {
            if collab.ParentPIC == nil {
                continue
            }
            collabsByPIC[*collab.ParentPIC] = append(collabsByPIC[*collab.ParentPIC], LazyCollaborator{
                ID:       collab.ID,
                Fullname: collab.Fullname,
                Email:    collab.Email,
            })
        }
    }

    items := make([]LazyCRItem, 0, len(crs))
    for _, cr := range crs {
        item := LazyCRItem{
            ID:       cr.ID,
            Title:    cr.Title,
            EndDate:  cr.EndDate,
            Module:   cr.Modul,
            Category: cr.Category,
        }

        if cr.PIC != nil {
            item.PIC = &LazyPIC{
                ID:            cr.PIC.ID,
                Fullname:      cr.PIC.Fullname,
                Collaborators: collabsByPIC[cr.PIC.ID],
            }
        }

        items = append(items, item)
    }

    c.JSON(http.StatusOK, utils.FormatResponse(
        "CR retrieved successfully",
        http.StatusOK,
        "success",
        gin.H{
            "items":      items,
            "pagination": utils.BuildPaginationMeta(pagination.Offset, pagination.Limit, len(items), total),
        },
    ))
}

// GetCRLazy godoc
// @Summary Get all CRs with pagination
// @Description Retrieve all change requests with lazy loading and pagination.
// @Tags CR
// @Produce json
// @Param offset query int false "Offset (default: 0)"
// @Param limit query int false "Limit (default: 10, max: 100)"
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/cr/lazy [get]
func GetCRLazy(c *gin.Context) {
    getCRLazyByStatus(c, "")
}

// GetCRLazyDraft godoc
// @Summary Get Draft CRs with pagination
// @Description Retrieve draft change requests with lazy loading pagination. Supports infinite scroll.
// @Tags CR
// @Produce json
// @Param offset query int false "Offset (default: 0)"
// @Param limit query int false "Limit (default: 10, max: 100)"
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/cr/lazy/draft [get]
func GetCRLazyDraft(c *gin.Context) {
    getCRLazyByStatus(c, "DRAFT")
}

// GetCRLazyIssued godoc
// @Summary Get Issued CRs with pagination
// @Description Retrieve issued change requests with lazy loading pagination. Supports infinite scroll.
// @Tags CR
// @Produce json
// @Param offset query int false "Offset (default: 0)"
// @Param limit query int false "Limit (default: 10, max: 100)"
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/cr/lazy/issued [get]
func GetCRLazyIssued(c *gin.Context) {
    getCRLazyByStatus(c, "ISSUED")
}

// GetCRLazyInProgress godoc
// @Summary Get In Progress CRs with pagination
// @Description Retrieve in progress change requests with lazy loading pagination. Supports infinite scroll.
// @Tags CR
// @Produce json
// @Param offset query int false "Offset (default: 0)"
// @Param limit query int false "Limit (default: 10, max: 100)"
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/cr/lazy/in-progress [get]
func GetCRLazyInProgress(c *gin.Context) {
    getCRLazyByStatus(c, "IN_PROGRESS")
}

// GetCRLazyApprovalToRelease godoc
// @Summary Get Approval to Release CRs with pagination
// @Description Retrieve approval to release change requests with lazy loading pagination. Supports infinite scroll.
// @Tags CR
// @Produce json
// @Param offset query int false "Offset (default: 0)"
// @Param limit query int false "Limit (default: 10, max: 100)"
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/cr/lazy/approval-to-release [get]
func GetCRLazyApprovalToRelease(c *gin.Context) {
    getCRLazyByStatus(c, "APPROVAL_TO_RELEASE")
}

// GetCRLazyRelease godoc
// @Summary Get Released CRs with pagination
// @Description Retrieve released change requests with lazy loading pagination. Supports infinite scroll.
// @Tags CR
// @Produce json
// @Param offset query int false "Offset (default: 0)"
// @Param limit query int false "Limit (default: 10, max: 100)"
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/cr/lazy/release [get]
func GetCRLazyRelease(c *gin.Context) {
    getCRLazyByStatus(c, "RELEASE")
}

// GetCRLazyApprovalToComplete godoc
// @Summary Get Approval to Complete CRs with pagination
// @Description Retrieve approval to complete change requests with lazy loading pagination. Supports infinite scroll.
// @Tags CR
// @Produce json
// @Param offset query int false "Offset (default: 0)"
// @Param limit query int false "Limit (default: 10, max: 100)"
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/cr/lazy/approval-to-complete [get]
func GetCRLazyApprovalToComplete(c *gin.Context) {
    getCRLazyByStatus(c, "APPROVAL_TO_COMPLETE")
}

// GetCRLazyComplete godoc
// @Summary Get Complete CRs with pagination
// @Description Retrieve complete change requests with lazy loading pagination. Supports infinite scroll.
// @Tags CR
// @Produce json
// @Param offset query int false "Offset (default: 0)"
// @Param limit query int false "Limit (default: 10, max: 100)"
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/cr/lazy/complete [get]
func GetCRLazyComplete(c *gin.Context) {
    getCRLazyByStatus(c, "COMPLETE")
}

// GetCRLazyCancel godoc
// @Summary Get Cancelled CRs with pagination
// @Description Retrieve cancelled change requests with lazy loading pagination. Supports infinite scroll.
// @Tags CR
// @Produce json
// @Param offset query int false "Offset (default: 0)"
// @Param limit query int false "Limit (default: 10, max: 100)"
// @Success 200 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /api/cr/lazy/cancel [get]
func GetCRLazyCancel(c *gin.Context) {
    getCRLazyByStatus(c, "CANCEL")
}