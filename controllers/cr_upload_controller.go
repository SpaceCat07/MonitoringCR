package controllers

import (
	"MonCR/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

type uploadAttachmentItem struct {
	OriginalFilename string `json:"original_filename"`
	URL              string `json:"url"`
	StoredPath       string `json:"stored_path"`
	Bytes            int64  `json:"bytes"`
}

// UploadCRAttachments godoc
// @Summary Upload CR attachments to local storage
// @Description Upload one or multiple files using multipart/form-data field name files. Returns local URLs for file_attachment.
// @Tags CR
// @Accept multipart/form-data
// @Produce json
// @Param files formData []file true "Attachment files"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/cr/attachments/upload [post]
func UploadCRAttachments(c *gin.Context) {
	claims, ok := getClaims(c)
	if !ok {
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid multipart form data",
			"details": err.Error(),
		})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "No files uploaded. Use form-data key: files",
		})
		return
	}

	uploaded := make([]uploadAttachmentItem, 0, len(files))
	fileAttachment := make([]string, 0, len(files))

	for _, fileHeader := range files {
		result, uploadErr := utils.UploadFileToLocal(fileHeader, "uploads/cr", "/uploads/cr")
		if uploadErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to upload attachment",
				"details": uploadErr.Error(),
			})
			return
		}

		uploaded = append(uploaded, uploadAttachmentItem{
			OriginalFilename: result.OriginalFilename,
			URL:              result.URL,
			StoredPath:       result.StoredPath,
			Bytes:            result.Bytes,
		})
		fileAttachment = append(fileAttachment, result.URL)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Attachments uploaded successfully",
		"data": gin.H{
			"uploaded_by":     claims.UserID,
			"uploaded":        uploaded,
			"file_attachment": fileAttachment,
		},
	})
}
