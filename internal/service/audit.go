/*
Copyright 2023 The Ketches Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	corev1alpha1 "github.com/ketches/ketches/api/core/v1alpha1"
	"github.com/ketches/ketches/internal/model"
	"github.com/ketches/ketches/pkg/ketches"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/strings"
)

type AuditService interface {
	CreateAudit(ctx context.Context, audit *model.CreateAuditRequest) error
	ListAudits(ctx context.Context, req *model.ListAuditsRequest) (*model.ListAuditsResponse, error)
}

type auditService struct {
	Service
}

func NewAuditService() AuditService {
	return &auditService{
		Service: LoadService(),
	}
}

var _ AuditService = (*auditService)(nil)

func (s *auditService) CreateAudit(ctx context.Context, req *model.CreateAuditRequest) error {
	_, err := s.KetchesClient().CoreV1alpha1().Audits(req.SpaceID).Create(ctx, &corev1alpha1.Audit{
		ObjectMeta: metav1.ObjectMeta{
			Name: req.SourceValue + "-" + strings.ShortenString(uuid.NewString(), 8),
			Labels: map[string]string{
				"ketches.io/source-key":   req.SourceKey,
				"ketches.io/source-value": req.SourceValue,
			},
		},
		Spec: corev1alpha1.AuditSpec{
			SourceKey:     req.SourceKey,
			SourceValue:   req.SourceValue,
			RequestMethod: req.RequestMethod,
			RequestPath:   req.RequestPath,
			Operator:      req.Operator,
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *auditService) ListAudits(ctx context.Context, req *model.ListAuditsRequest) (*model.ListAuditsResponse, error) {
	list, err := ketches.Store().AuditLister().List(labels.SelectorFromSet(labels.Set{
		"ketches.io/source-key":   req.SourceKey,
		"ketches.io/source-value": req.SourceValue,
	}))
	if err != nil {
		slog.Error("list audits error", err)
		return nil, err
	}

	audits := make([]model.AuditResponse, len(list))
	for _, audit := range list {
		audits = append(audits, model.AuditResponse{
			AuditModel: model.AuditModel{
				SourceKey:     audit.Spec.SourceKey,
				SourceValue:   audit.Spec.SourceValue,
				RequestMethod: audit.Spec.RequestMethod,
				RequestPath:   audit.Spec.RequestPath,
				Operator:      audit.Spec.Operator,
			},
			CreatedAt: audit.CreationTimestamp.Format(time.DateTime),
		})
	}
	return &model.ListAuditsResponse{
		Audits: audits,
	}, nil
}
