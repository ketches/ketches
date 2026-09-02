package services

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreateNotification inserts a new notification record.
func CreateNotification(recipientID, senderID, category, eventType, title, message, resourceType, resourceID, projectID, actionData string) error {
	notif := newNotification(recipientID, senderID, category, eventType, title, message, resourceType, resourceID, projectID, actionData)
	return db.DB.Create(notif).Error
}

func newNotification(recipientID, senderID, category, eventType, title, message, resourceType, resourceID, projectID, actionData string) *entities.Notification {
	return &entities.Notification{
		Base:         entities.Base{ID: uuid.New()},
		RecipientID:  recipientID,
		SenderID:     senderID,
		Category:     category,
		EventType:    eventType,
		Title:        title,
		Message:      message,
		Status:       "pending",
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ProjectID:    projectID,
		ActionData:   actionData,
	}
}

// ListNotifications returns paginated notifications for a user with joined sender/project names.
func ListNotifications(userID string, req *models.PaginationRequest) (int64, []models.NotificationJoinRow, error) {
	var total int64
	countQ := db.DB.Model(&entities.Notification{}).Where("recipient_id = ?", userID)
	if err := countQ.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	var rows []models.NotificationJoinRow
	dataQ := db.DB.Table("notifications").
		Select(`notifications.id, notifications.sender_id,
			COALESCE(u.fullname, u.username, '') AS sender_name,
			notifications.category, notifications.event_type,
			notifications.title, notifications.message, notifications.status,
			notifications.read_at, notifications.resource_type, notifications.resource_id,
			notifications.project_id,
			COALESCE(p.name, '') AS project_name,
			notifications.action_data, notifications.created_at`).
		Joins("LEFT JOIN users u ON u.id = notifications.sender_id").
		Joins("LEFT JOIN projects p ON p.id = notifications.project_id").
		Where("notifications.recipient_id = ?", userID).
		Where("notifications.deleted_at IS NULL").
		Order("notifications.created_at DESC").
		Offset(req.GetOffset()).Limit(req.PageSize)

	if err := dataQ.Find(&rows).Error; err != nil {
		return 0, nil, err
	}

	return total, rows, nil
}

// GetUnreadCount returns the count of unread (pending) notifications for a user.
func GetUnreadCount(userID string) (int64, error) {
	var count int64
	err := db.DB.Model(&entities.Notification{}).
		Where("recipient_id = ? AND status = 'pending'", userID).
		Count(&count).Error
	return count, err
}

// HandleNotificationAction processes an action on a notification.
func HandleNotificationAction(notifID string, userID string, action string) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var notif entities.Notification
		if err := lockForUpdate(tx).Where("id = ? AND recipient_id = ?", notifID, userID).First(&notif).Error; err != nil {
			return errors.New("notification not found")
		}

		// Replaying an already completed action is intentionally idempotent. This
		// is important when a client retries after a response timeout.
		if action == "accept" && notif.Status == "accepted" {
			return nil
		}
		if action == "refuse" && notif.Status == "refused" {
			return nil
		}

		now := time.Now()
		switch action {
		case "accept":
			if notif.Category != "invitation" || notif.Status != "pending" {
				return errors.New("notification cannot be accepted")
			}
			// Parse action data for role.
			role := "developer"
			if notif.ActionData != "" {
				var data map[string]string
				if err := json.Unmarshal([]byte(notif.ActionData), &data); err == nil {
					if r, ok := data["role"]; ok && r != "" {
						role = r
					}
				}
			}
			if err := validateProjectRole(role); err != nil {
				return err
			}

			// The notification row lock serializes retries for one invitation. A
			// unique project/user index plus DoNothing makes acceptance safe even
			// when multiple invitation records exist due to legacy data.
			member := &entities.ProjectMember{
				ID:          uuid.New(),
				ProjectID:   notif.ResourceID,
				UserID:      userID,
				ProjectRole: role,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "project_id"}, {Name: "user_id"}},
				DoNothing: true,
			}).Create(member).Error; err != nil {
				return err
			}
			result := tx.Model(&entities.Notification{}).
				Where("id = ? AND recipient_id = ? AND status = ?", notifID, userID, "pending").
				Updates(map[string]any{"status": "accepted", "read_at": now})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errors.New("notification state changed")
			}

		case "refuse":
			if notif.Category != "invitation" || notif.Status != "pending" {
				return errors.New("notification cannot be refused")
			}
			result := tx.Model(&entities.Notification{}).
				Where("id = ? AND recipient_id = ? AND status = ?", notifID, userID, "pending").
				Updates(map[string]any{"status": "refused", "read_at": now})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errors.New("notification state changed")
			}

		case "read", "dismiss":
			status := "read"
			if action == "dismiss" {
				status = "dismissed"
			}
			result := tx.Model(&entities.Notification{}).
				Where("id = ? AND recipient_id = ? AND status = ?", notifID, userID, notif.Status).
				Updates(map[string]any{"status": status, "read_at": now})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errors.New("notification state changed")
			}

		default:
			return errors.New("invalid action")
		}
		return nil
	})
}

// MarkAllRead marks all pending notifications as read for a user.
func MarkAllRead(userID string) error {
	now := time.Now()
	return db.DB.Model(&entities.Notification{}).
		Where("recipient_id = ? AND status = 'pending' AND category != 'invitation'", userID).
		Updates(map[string]any{"status": "read", "read_at": now}).Error
}

// HasPendingInvitation checks if a pending invitation notification already exists.
func HasPendingInvitation(recipientID, projectID string) (bool, string, error) {
	var notif entities.Notification
	err := db.DB.Where("recipient_id = ? AND resource_id = ? AND event_type = 'project_invitation' AND status = 'pending'",
		recipientID, projectID).First(&notif).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, notif.ID, nil
}
