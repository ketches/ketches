package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/services"
)

type authRateLimitRule struct {
	scope    string
	identity string
	limit    int
	window   time.Duration
}

func enforceAuthRateLimits(c *gin.Context, rules ...authRateLimitRule) bool {
	for _, rule := range rules {
		if err := services.EnforceAuthRateLimit(rule.scope, rule.identity, rule.limit, rule.window); err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, services.ErrAuthRateLimited) {
				status = http.StatusTooManyRequests
			}
			api.Error(c, status, err)
			return false
		}
	}
	return true
}
