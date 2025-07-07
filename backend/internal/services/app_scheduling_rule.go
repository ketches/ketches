package services

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/db/orm"
	"github.com/ketches/ketches/internal/models"
)

type AppSchedulingRuleService interface {
	GetAppSchedulingRule(ctx context.Context, req *models.GetAppSchedulingRuleRequest) (*models.AppSchedulingRuleModel, app.Error)
	SetAppSchedulingRule(ctx context.Context, req *models.SetAppSchedulingRuleRequest) (*models.AppSchedulingRuleModel, app.Error)
	DeleteAppSchedulingRule(ctx context.Context, appID string) app.Error
}

type appSchedulingRuleService struct {
	Service
}

var appSchedulingRuleServiceInstance = &appSchedulingRuleService{
	Service: LoadService(),
}

func NewAppSchedulingRuleService() AppSchedulingRuleService {
	return appSchedulingRuleServiceInstance
}

func (s *appSchedulingRuleService) GetAppSchedulingRule(ctx context.Context, req *models.GetAppSchedulingRuleRequest) (*models.AppSchedulingRuleModel, app.Error) {
	// 验证应用是否存在
	_, err := orm.GetAppByID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	var rule entities.AppSchedulingRule
	if err := db.Instance().Where("app_id = ?", req.AppID).First(&rule).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			return nil, nil // 没有调度规则返回nil
		}
		log.Printf("failed to get app scheduling rule: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	var (
		nodeName     string
		nodeSelector []string
		nodeAffinity []string
		tolerations  []models.AppSchedulingRuleTolerationModel
	)
	switch rule.RuleType {
	case app.SchedulingRuleTypeNodeName:
		nodeName = rule.NodeName
	case app.SchedulingRuleTypeNodeSelector:
		nodeSelector = strings.Split(rule.NodeSelector, ",")
	case app.SchedulingRuleTypeNodeAffinity:
		nodeAffinity = strings.Split(rule.NodeAffinity, ",")
	}

	if rule.Tolerations != "" {
		if err := json.Unmarshal([]byte(rule.Tolerations), &tolerations); err != nil {
			log.Printf("failed to unmarshal app scheduling rule tolerations: %v", err)
			return nil, app.NewError(http.StatusInternalServerError, "无法解析调度规则容忍设置")
		}
	}

	result := &models.AppSchedulingRuleModel{
		RuleID:       rule.ID,
		AppID:        rule.AppID,
		RuleType:     rule.RuleType,
		NodeName:     nodeName,
		NodeSelector: nodeSelector,
		NodeAffinity: nodeAffinity,
		Tolerations:  tolerations,
	}

	return result, nil
}

func (s *appSchedulingRuleService) SetAppSchedulingRule(ctx context.Context, req *models.SetAppSchedulingRuleRequest) (*models.AppSchedulingRuleModel, app.Error) {
	// 验证应用是否存在
	_, err := orm.GetAppByID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	switch req.RuleType {
	case app.SchedulingRuleTypeNodeName:
		if req.NodeName == "" {
			return nil, app.NewError(http.StatusBadRequest, "节点名称不能为空")
		}
	case app.SchedulingRuleTypeNodeSelector:
		if len(req.NodeSelector) == 0 {
			return nil, app.NewError(http.StatusBadRequest, "节点标签不能为空")
		}
	case app.SchedulingRuleTypeNodeAffinity:
		if len(req.NodeAffinity) == 0 {
			return nil, app.NewError(http.StatusBadRequest, "亲和节点不能为空")
		}
	case "":
		if len(req.Tolerations) == 0 {
			return nil, app.NewError(http.StatusBadRequest, "未设置有效的调度规则")
		}
	default:
		return nil, app.NewError(http.StatusBadRequest, "无效的调度规则类型")
	}

	nodeSelector := strings.Join(req.NodeSelector, ",")
	nodeAffinity := strings.Join(req.NodeAffinity, ",")

	var (
		tolerations []byte
		e           error
	)

	if len(req.Tolerations) > 0 {
		for _, toleration := range req.Tolerations {
			if toleration.Key == "" || toleration.Operator == "" || toleration.Value == "" || toleration.Effect == "" {
				return nil, app.NewError(http.StatusBadRequest, "包含无效的调度规则容忍设置")
			}
			if toleration.Key == "" {
				if toleration.Operator == app.SchedulingRuleTolerationOperatorEqual {
					return nil, app.NewError(http.StatusBadRequest, "当前容忍设置的操作符为 Equal 时，容忍键不能为空")
				}
				if toleration.Value != "" {
					return nil, app.NewError(http.StatusBadRequest, "当前容忍设置的操作符为 Equal 时，容忍键不能为空")
				}
			}

		}
		tolerations, e = json.Marshal(req.Tolerations)
		if e != nil {
			log.Printf("failed to marshal tolerations: %v", e)
			return nil, app.NewError(http.StatusInternalServerError, "无法解析调度规则容忍设置")
		}
	}

	// 查找现有规则
	var existingRule entities.AppSchedulingRule
	found := true
	if err := db.Instance().Where("app_id = ?", req.AppID).First(&existingRule).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			found = false
		} else {
			log.Printf("failed to query existing scheduling rule: %v", err)
			return nil, app.ErrDatabaseOperationFailed
		}
	}

	if found {
		// 更新现有规则
		existingRule.RuleType = req.RuleType
		existingRule.NodeName = req.NodeName
		existingRule.NodeSelector = nodeSelector
		existingRule.NodeAffinity = nodeAffinity
		existingRule.Tolerations = string(tolerations)
		existingRule.UpdatedBy = api.UserID(ctx)

		if err := db.Instance().Save(&existingRule).Error; err != nil {
			log.Printf("failed to update app scheduling rule: %v", err)
			return nil, app.ErrDatabaseOperationFailed
		}
	} else {
		// 创建新规则
		existingRule = entities.AppSchedulingRule{
			AppID:        req.AppID,
			RuleType:     req.RuleType,
			NodeName:     req.NodeName,
			NodeSelector: nodeSelector,
			NodeAffinity: nodeAffinity,
			Tolerations:  string(tolerations),
			AuditBase: entities.AuditBase{
				CreatedBy: api.UserID(ctx),
				UpdatedBy: api.UserID(ctx),
			},
		}

		if err := db.Instance().Create(&existingRule).Error; err != nil {
			log.Printf("failed to create app scheduling rule: %v", err)
			return nil, app.ErrDatabaseOperationFailed
		}
	}

	result := &models.AppSchedulingRuleModel{
		RuleID:       existingRule.ID,
		AppID:        existingRule.AppID,
		RuleType:     existingRule.RuleType,
		NodeName:     existingRule.NodeName,
		NodeSelector: req.NodeSelector,
		NodeAffinity: req.NodeAffinity,
	}

	return result, nil
}

func (s *appSchedulingRuleService) DeleteAppSchedulingRule(ctx context.Context, appID string) app.Error {
	// 验证应用是否存在
	_, err := orm.GetAppByID(ctx, appID)
	if err != nil {
		return err
	}

	if err := db.Instance().Where("app_id = ?", appID).Delete(&entities.AppSchedulingRule{}).Error; err != nil {
		log.Printf("failed to delete app scheduling rule: %v", err)
		return app.ErrDatabaseOperationFailed
	}

	return nil
}
