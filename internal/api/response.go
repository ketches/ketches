package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Response struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{Data: data})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Response{Data: data})
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func Error(c *gin.Context, status int, err error) {
	if err == nil {
		err = errors.New(http.StatusText(status))
	}
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		status = http.StatusRequestEntityTooLarge
	}

	if status >= http.StatusInternalServerError {
		slog.Error("request failed",
			"status", status,
			"method", c.Request.Method,
			"path", c.FullPath(),
			"error", err,
		)
	} else {
		slog.Warn("request rejected",
			"status", status,
			"method", c.Request.Method,
			"path", c.FullPath(),
			"error", err,
		)
	}

	message := clientErrorMessage(status, err)

	c.JSON(status, Response{Error: message})
}

func clientErrorMessage(status int, err error) string {
	if status >= http.StatusInternalServerError {
		return http.StatusText(status)
	}
	if shouldSanitizeClientError(err) {
		return http.StatusText(status)
	}
	return err.Error()
}

func shouldSanitizeClientError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true
	}

	message := strings.ToLower(err.Error())
	databaseErrorMarkers := []string{
		"record not found",
		"unique constraint failed",
		"duplicate key value",
		"violates unique constraint",
		"sqlstate ",
		"constraint failed",
		"foreign key constraint",
		"error 1062",
	}
	for _, marker := range databaseErrorMarkers {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}
