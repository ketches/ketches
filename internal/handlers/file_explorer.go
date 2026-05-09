package handlers

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	appcore "github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

// GetHomeDir returns the home directory of the container
func GetHomeDir(c *gin.Context) {
	appID := c.Param("appID")
	instanceName := c.Param("instanceName")
	containerName := c.Query("container")

	if containerName == "" {
		api.Error(c, http.StatusBadRequest, appcore.NewErrorf("container parameter is required"))
		return
	}

	app, err := services.GetAppContext(c.Request.Context(), appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, appcore.WrapError("app not found", err))
		return
	}

	home, err := services.GetHomeDir(app, instanceName, containerName)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, gin.H{"path": home})
}

// CompressFiles compresses selected files into a tar.gz inside the container
func CompressFiles(c *gin.Context) {
	appID := c.Param("appID")
	instanceName := c.Param("instanceName")
	containerName := c.Query("container")

	if containerName == "" {
		api.Error(c, http.StatusBadRequest, appcore.NewErrorf("container parameter is required"))
		return
	}

	var req models.CompressFilesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.GetAppContext(c.Request.Context(), appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, appcore.WrapError("app not found", err))
		return
	}

	if err := services.CompressFiles(app, instanceName, containerName, req.BaseDir, req.FileNames, req.DestPath); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

// CompressAndDownloadFiles compresses selected files and downloads as tar.gz
func CompressAndDownloadFiles(c *gin.Context) {
	appID := c.Param("appID")
	instanceName := c.Param("instanceName")
	containerName := c.Query("container")

	if containerName == "" {
		api.Error(c, http.StatusBadRequest, appcore.NewErrorf("container parameter is required"))
		return
	}

	var req models.CompressAndDownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.GetAppContext(c.Request.Context(), appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, appcore.WrapError("app not found", err))
		return
	}

	c.Header("Content-Type", "application/gzip")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, req.ArchiveName))

	if err := services.CompressAndDownloadFiles(app, instanceName, containerName, req.BaseDir, req.FileNames, c.Writer); err != nil {
		_ = c.Error(appcore.NewErrorf("failed to compress and download: %v", err))
	}
}

// ListFiles lists files in a directory inside a container
func ListFiles(c *gin.Context) {
	appID := c.Param("appID")
	instanceName := c.Param("instanceName")
	containerName := c.Query("container")
	path := c.Query("path")

	if containerName == "" {
		api.Error(c, http.StatusBadRequest, appcore.NewErrorf("container parameter is required"))
		return
	}

	app, err := services.GetAppContext(c.Request.Context(), appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, appcore.WrapError("app not found", err))
		return
	}

	result, err := services.ListFiles(app, instanceName, containerName, path)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, result)
}

// ReadFile reads the content of a file inside a container
func ReadFile(c *gin.Context) {
	appID := c.Param("appID")
	instanceName := c.Param("instanceName")
	containerName := c.Query("container")
	path := c.Query("path")

	if containerName == "" {
		api.Error(c, http.StatusBadRequest, appcore.NewErrorf("container parameter is required"))
		return
	}

	if path == "" {
		api.Error(c, http.StatusBadRequest, appcore.NewErrorf("path parameter is required"))
		return
	}

	app, err := services.GetAppContext(c.Request.Context(), appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, appcore.WrapError("app not found", err))
		return
	}

	result, err := services.ReadFile(app, instanceName, containerName, path)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, result)
}

// WriteFile writes content to a file inside a container
func WriteFile(c *gin.Context) {
	appID := c.Param("appID")
	instanceName := c.Param("instanceName")
	containerName := c.Query("container")

	if containerName == "" {
		api.Error(c, http.StatusBadRequest, appcore.NewErrorf("container parameter is required"))
		return
	}

	var req models.WriteFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.GetAppContext(c.Request.Context(), appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, appcore.WrapError("app not found", err))
		return
	}

	if err := services.WriteFile(app, instanceName, containerName, req.Path, req.Content); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

// MkdirInContainer creates a directory inside a container
func MkdirInContainer(c *gin.Context) {
	appID := c.Param("appID")
	instanceName := c.Param("instanceName")
	containerName := c.Query("container")

	if containerName == "" {
		api.Error(c, http.StatusBadRequest, appcore.NewErrorf("container parameter is required"))
		return
	}

	var req models.MkdirRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.GetAppContext(c.Request.Context(), appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, appcore.WrapError("app not found", err))
		return
	}

	if err := services.MkdirInContainer(app, instanceName, containerName, req.Path); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

// DeleteFileInContainer deletes a file or directory inside a container
func DeleteFileInContainer(c *gin.Context) {
	appID := c.Param("appID")
	instanceName := c.Param("instanceName")
	containerName := c.Query("container")

	if containerName == "" {
		api.Error(c, http.StatusBadRequest, appcore.NewErrorf("container parameter is required"))
		return
	}

	var req models.DeleteFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.GetAppContext(c.Request.Context(), appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, appcore.WrapError("app not found", err))
		return
	}

	if err := services.DeleteFileInContainer(app, instanceName, containerName, req.Path); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

// MoveFileInContainer moves/renames a file or directory inside a container
func MoveFileInContainer(c *gin.Context) {
	appID := c.Param("appID")
	instanceName := c.Param("instanceName")
	containerName := c.Query("container")

	if containerName == "" {
		api.Error(c, http.StatusBadRequest, appcore.NewErrorf("container parameter is required"))
		return
	}

	var req models.MoveFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.GetAppContext(c.Request.Context(), appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, appcore.WrapError("app not found", err))
		return
	}

	if err := services.MoveFileInContainer(app, instanceName, containerName, req.Source, req.Destination); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

// CopyFileInContainer copies a file or directory inside a container
func CopyFileInContainer(c *gin.Context) {
	appID := c.Param("appID")
	instanceName := c.Param("instanceName")
	containerName := c.Query("container")

	if containerName == "" {
		api.Error(c, http.StatusBadRequest, appcore.NewErrorf("container parameter is required"))
		return
	}

	var req models.CopyFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.GetAppContext(c.Request.Context(), appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, appcore.WrapError("app not found", err))
		return
	}

	if err := services.CopyFileInContainer(app, instanceName, containerName, req.Source, req.Destination); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

// DownloadFile downloads a file from a container
func DownloadFile(c *gin.Context) {
	appID := c.Param("appID")
	instanceName := c.Param("instanceName")
	containerName := c.Query("container")
	path := c.Query("path")

	if containerName == "" {
		api.Error(c, http.StatusBadRequest, appcore.NewErrorf("container parameter is required"))
		return
	}

	if path == "" {
		api.Error(c, http.StatusBadRequest, appcore.NewErrorf("path parameter is required"))
		return
	}

	app, err := services.GetAppContext(c.Request.Context(), appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, appcore.WrapError("app not found", err))
		return
	}

	// Stream the tar output to a buffer, then extract the actual file from it
	var tarBuf bytes.Buffer
	if err := services.DownloadFile(app, instanceName, containerName, path, &tarBuf); err != nil {
		api.Error(c, http.StatusInternalServerError, appcore.NewErrorf("failed to download file: %v", err))
		return
	}

	// Extract the file from the tar archive so the browser gets the raw file
	tr := tar.NewReader(&tarBuf)
	header, err := tr.Next()
	if err != nil {
		// Fallback: send the raw tar if extraction fails
		c.Header("Content-Type", "application/octet-stream")
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.tar"`, sanitizeFilename(path)))
		tarBuf.Reset()
		if err := services.DownloadFile(app, instanceName, containerName, path, &tarBuf); err != nil {
			api.Error(c, http.StatusInternalServerError, appcore.NewErrorf("failed to download file: %v", err))
			return
		}
		c.Header("Content-Length", fmt.Sprintf("%d", tarBuf.Len()))
		if _, err := io.Copy(c.Writer, &tarBuf); err != nil {
			_ = c.Error(appcore.NewErrorf("failed to stream archive: %v", err))
		}
		return
	}

	filename := sanitizeFilename(path)
	contentType := detectContentType(filename)

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if header.Size > 0 {
		c.Header("Content-Length", fmt.Sprintf("%d", header.Size))
	}

	if _, err := io.Copy(c.Writer, tr); err != nil {
		_ = c.Error(appcore.NewErrorf("failed to stream file: %v", err))
	}
}

// DownloadFileDir downloads a directory as a tar archive from a container
func DownloadFileDir(c *gin.Context) {
	appID := c.Param("appID")
	instanceName := c.Param("instanceName")
	containerName := c.Query("container")
	path := c.Query("path")

	if containerName == "" {
		api.Error(c, http.StatusBadRequest, appcore.NewErrorf("container parameter is required"))
		return
	}

	if path == "" {
		api.Error(c, http.StatusBadRequest, appcore.NewErrorf("path parameter is required"))
		return
	}

	app, err := services.GetAppContext(c.Request.Context(), appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, appcore.WrapError("app not found", err))
		return
	}

	var tarBuf bytes.Buffer
	if err := services.DownloadFile(app, instanceName, containerName, path, &tarBuf); err != nil {
		api.Error(c, http.StatusInternalServerError, appcore.NewErrorf("failed to download: %v", err))
		return
	}

	filename := sanitizeFilename(path)
	c.Header("Content-Type", "application/x-tar")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.tar"`, filename))
	c.Header("Content-Length", fmt.Sprintf("%d", tarBuf.Len()))

	if _, err := io.Copy(c.Writer, &tarBuf); err != nil {
		_ = c.Error(appcore.NewErrorf("failed to stream archive: %v", err))
	}
}

// detectContentType returns a MIME type based on file extension
func detectContentType(filename string) string {
	ext := ""
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			ext = filename[i:]
			break
		}
	}
	switch ext {
	case ".txt", ".log", ".md", ".csv":
		return "text/plain; charset=utf-8"
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".yaml", ".yml":
		return "text/yaml; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	case ".zip":
		return "application/zip"
	case ".gz", ".gzip":
		return "application/gzip"
	default:
		return "application/octet-stream"
	}
}

// UploadFile uploads a file to a container
func UploadFile(c *gin.Context) {
	appID := c.Param("appID")
	instanceName := c.Param("instanceName")
	containerName := c.Query("container")
	destDir := c.Query("path")

	if containerName == "" {
		api.Error(c, http.StatusBadRequest, appcore.NewErrorf("container parameter is required"))
		return
	}

	if destDir == "" {
		destDir = "/"
	}

	app, err := services.GetAppContext(c.Request.Context(), appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, appcore.WrapError("app not found", err))
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		api.Error(c, http.StatusBadRequest, appcore.NewErrorf("file parameter is required: %v", err))
		return
	}
	defer func() {
		_ = file.Close()
	}()

	if err := services.UploadFile(app, instanceName, containerName, destDir, header.Filename, file, header.Size); err != nil {
		api.Error(c, http.StatusInternalServerError, appcore.NewErrorf("failed to upload file: %v", err))
		return
	}

	api.NoContent(c)
}

// sanitizeFilename extracts a safe filename from a path
func sanitizeFilename(path string) string {
	parts := splitPath(path)
	if len(parts) == 0 {
		return "download"
	}
	return parts[len(parts)-1]
}

func splitPath(path string) []string {
	var parts []string
	for _, p := range splitString(path, '/') {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func splitString(s string, sep byte) []string {
	var result []string
	start := 0
	for i := range len(s) {
		if s[i] == sep {
			if i > start {
				result = append(result, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}
