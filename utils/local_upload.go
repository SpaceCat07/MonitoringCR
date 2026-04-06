package utils

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type LocalUploadResult struct {
	OriginalFilename string `json:"original_filename"`
	URL              string `json:"url"`
	StoredPath       string `json:"stored_path"`
	Bytes            int64  `json:"bytes"`
}

func sanitizeFilename(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	base := strings.TrimSpace(strings.TrimSuffix(name, filepath.Ext(name)))
	base = strings.ToLower(base)
	re := regexp.MustCompile(`[^a-z0-9_-]+`)
	base = re.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "file"
	}
	return base + ext
}

func UploadFileToLocal(fileHeader *multipart.FileHeader, saveDir string, publicBasePath string) (*LocalUploadResult, error) {
	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	safeName := sanitizeFilename(fileHeader.Filename)
	storedName := fmt.Sprintf("%d-%s", time.Now().UnixNano(), safeName)
	storedPath := filepath.Join(saveDir, storedName)

	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file %s: %w", fileHeader.Filename, err)
	}
	defer src.Close()

	dst, err := os.Create(storedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file %s: %w", storedPath, err)
	}
	defer dst.Close()

	bytesWritten, err := io.Copy(dst, src)
	if err != nil {
		return nil, fmt.Errorf("failed to write file %s: %w", storedPath, err)
	}

	publicBasePath = strings.TrimRight(publicBasePath, "/")
	urlPath := publicBasePath + "/" + storedName

	return &LocalUploadResult{
		OriginalFilename: fileHeader.Filename,
		URL:              urlPath,
		StoredPath:       storedPath,
		Bytes:            bytesWritten,
	}, nil
}
