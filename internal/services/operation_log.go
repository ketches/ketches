package services

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

const (
	operationLogRetentionSettingKey  = "operation_log_retention_days"
	defaultOperationLogRetentionDays = 90
	maxOperationLogSummaryLength     = 2048
	minOperationLogRetentionDays     = 1
	maxOperationLogRetentionDays     = 3650
)

type CreateOperationLogInput struct {
	UserID         string
	Username       string
	Action         string
	ResourceType   string
	ResourceID     string
	ProjectID      string
	EnvID          string
	AppID          string
	RepoID         string
	Status         string
	StatusCode     int
	Sensitivity    string
	RequestSummary string
	ClientIP       string
}

func CreateOperationLog(input CreateOperationLogInput) error {
	log := entities.OperationLog{
		Base:           entities.Base{ID: uuid.New()},
		Username:       input.Username,
		Action:         input.Action,
		ResourceType:   input.ResourceType,
		ResourceID:     input.ResourceID,
		Status:         normalizeOperationLogStatus(input.Status),
		StatusCode:     input.StatusCode,
		Sensitivity:    normalizeOperationLogSensitivity(input.Sensitivity),
		RequestSummary: truncateOperationLogSummary(input.RequestSummary),
		ClientIP:       input.ClientIP,
	}

	log.UserID = nullableString(input.UserID)
	log.ProjectID = nullableString(input.ProjectID)
	log.EnvID = nullableString(input.EnvID)
	log.AppID = nullableString(input.AppID)
	log.RepoID = nullableString(input.RepoID)

	return db.DB.Create(&log).Error
}

func ListOperationLogs(req models.OperationLogListRequest) (int64, []models.OperationLogItem, error) {
	return listOperationLogsWithScope(req, operationLogQueryScope{AdminGlobal: true})
}

func ListActivities(req models.OperationLogListRequest, currentUserID string, isAdmin bool) (int64, []models.OperationLogItem, error) {
	scope := operationLogQueryScope{Activities: true, CurrentUserID: currentUserID, IsAdmin: isAdmin}
	return listOperationLogsWithScope(req, scope)
}

func ListAppOperationLogs(appID string, req models.OperationLogListRequest) (int64, []models.OperationLogItem, error) {
	return listOperationLogsWithScope(req, operationLogQueryScope{AppID: appID})
}

func ListCodeRepositoryOperationLogs(repoID string, req models.OperationLogListRequest) (int64, []models.OperationLogItem, error) {
	return listOperationLogsWithScope(req, operationLogQueryScope{RepoID: repoID})
}

func GetOperationLogRetentionDays() (int, error) {
	setting, err := getSystemSetting(operationLogRetentionSettingKey)
	if err != nil {
		return 0, err
	}
	if setting == nil || strings.TrimSpace(setting.Value) == "" {
		return defaultOperationLogRetentionDays, nil
	}
	days, err := strconv.Atoi(setting.Value)
	if err != nil || days <= 0 {
		return defaultOperationLogRetentionDays, nil
	}
	return days, nil
}

func UpdateOperationLogRetentionDays(days int) error {
	if days < minOperationLogRetentionDays || days > maxOperationLogRetentionDays {
		return errors.New("retention_days must be between 1 and 3650")
	}
	value := strconv.Itoa(days)
	setting, err := getSystemSetting(operationLogRetentionSettingKey)
	if err != nil {
		return err
	}
	if setting == nil {
		return db.DB.Create(&entities.SystemSetting{Base: entities.Base{ID: uuid.New()}, Key: operationLogRetentionSettingKey, Value: value}).Error
	}
	return db.DB.Model(&entities.SystemSetting{}).Where("id = ?", setting.ID).Update("value", value).Error
}

func CleanupExpiredOperationLogs() (int64, error) {
	days, err := GetOperationLogRetentionDays()
	if err != nil {
		return 0, err
	}
	deadline := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	result := db.DB.Where("created_at < ?", deadline).Delete(&entities.OperationLog{})
	return result.RowsAffected, result.Error
}

type operationLogQueryScope struct {
	AdminGlobal   bool
	Activities    bool
	CurrentUserID string
	IsAdmin       bool
	AppID         string
	RepoID        string
}

func listOperationLogsWithScope(req models.OperationLogListRequest, scope operationLogQueryScope) (int64, []models.OperationLogItem, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	query := db.DB.Model(&entities.OperationLog{})

	if scope.AppID != "" {
		query = query.Where("app_id = ?", scope.AppID)
	}
	if scope.RepoID != "" {
		query = query.Where("repo_id = ?", scope.RepoID)
	}

	if scope.Activities {
		if scope.IsAdmin {
			if req.UserID != "" {
				query = query.Where("user_id = ?", req.UserID)
			}
		} else {
			query = query.Where("(user_id = ? OR sensitivity = ?)", scope.CurrentUserID, entities.OperationLogSensitivityPublic)
		}
	}

	if req.Search != "" {
		search := "%" + req.Search + "%"
		query = query.Where("username LIKE ? OR action LIKE ? OR resource_type LIKE ? OR resource_id LIKE ? OR request_summary LIKE ?", search, search, search, search, search)
	}
	if req.Action != "" {
		query = query.Where("action = ?", req.Action)
	}
	if req.ResourceType != "" {
		query = query.Where("resource_type = ?", req.ResourceType)
	}
	if req.Sensitivity != "" {
		query = query.Where("sensitivity = ?", req.Sensitivity)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.UserID != "" && scope.AdminGlobal {
		query = query.Where("user_id = ?", req.UserID)
	}

	if t, ok := parseOperationLogTime(req.Start); ok {
		query = query.Where("created_at >= ?", t)
	}
	if t, ok := parseOperationLogTime(req.End); ok {
		query = query.Where("created_at <= ?", t)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	var rows []entities.OperationLog
	if err := query.Order("created_at DESC").Offset((req.Page - 1) * req.PageSize).Limit(req.PageSize).Find(&rows).Error; err != nil {
		return 0, nil, err
	}

	items := make([]models.OperationLogItem, 0, len(rows))
	for i := range rows {
		items = append(items, toOperationLogItem(&rows[i]))
	}
	return total, items, nil
}

func toOperationLogItem(row *entities.OperationLog) models.OperationLogItem {
	item := models.OperationLogItem{
		ID:             row.ID,
		CreatedAt:      row.CreatedAt,
		Username:       row.Username,
		Action:         row.Action,
		ResourceType:   row.ResourceType,
		ResourceID:     row.ResourceID,
		Status:         row.Status,
		StatusCode:     row.StatusCode,
		Sensitivity:    row.Sensitivity,
		RequestSummary: row.RequestSummary,
		ClientIP:       row.ClientIP,
	}
	if row.UserID != nil {
		item.UserID = *row.UserID
	}
	if row.ProjectID != nil {
		item.ProjectID = *row.ProjectID
	}
	if row.EnvID != nil {
		item.EnvID = *row.EnvID
	}
	if row.AppID != nil {
		item.AppID = *row.AppID
	}
	if row.RepoID != nil {
		item.RepoID = *row.RepoID
	}
	return item
}

func getSystemSetting(key string) (*entities.SystemSetting, error) {
	var setting entities.SystemSetting
	err := db.DB.Where("key = ?", key).First(&setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

func parseOperationLogTime(value string) (time.Time, bool) {
	v := strings.TrimSpace(value)
	if v == "" {
		return time.Time{}, false
	}

	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, true
	}

	layouts := []string{
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, v, time.Local)
		if err == nil {
			return t, true
		}
	}

	return time.Time{}, false
}

func nullableString(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func truncateOperationLogSummary(s string) string {
	if len(s) <= maxOperationLogSummaryLength {
		return s
	}
	return s[:maxOperationLogSummaryLength]
}

func normalizeOperationLogSensitivity(v string) string {
	s := strings.TrimSpace(v)
	if s == "" {
		return entities.OperationLogSensitivityPublic
	}
	if s == entities.OperationLogSensitivityPublic || s == entities.OperationLogSensitivityInternal || s == entities.OperationLogSensitivitySensitive {
		return s
	}
	return entities.OperationLogSensitivityInternal
}

func normalizeOperationLogStatus(v string) string {
	if strings.TrimSpace(v) == entities.OperationLogStatusSuccess {
		return entities.OperationLogStatusSuccess
	}
	return entities.OperationLogStatusFailure
}
