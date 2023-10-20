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

package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/global"
	"github.com/ketches/ketches/internal/model"
	"github.com/ketches/ketches/internal/service"
)

func CreateAudit(c *gin.Context) {
	var req = new(model.CreateAuditRequest)
	operator := c.GetString(global.ContextKeyAccountID)
	if len(operator) == 0 {
		operator = global.SystemOperator
	}

	if v, ok := c.GetQuery("application_id"); ok && len(v) > 0 {
		req.SourceKey = global.AuditSourceKeyApplication.String()
		req.SourceValue = v
	} else if v, ok := c.GetQuery("space_id"); ok && len(v) > 0 {
		req.SourceKey = global.AuditSourceKeySpace.String()
		req.SourceValue = v
	} else {
		req.SourceKey = global.AuditSourceKeyPlatform.String()
		req.SourceValue = ""
	}

	req.RequestMethod = c.Request.Method
	req.RequestPath = c.Request.URL.Path

	service.NewAuditService().CreateAudit(c, req)
}

func ListAudits(c *gin.Context) {
	var req = new(model.ListAuditsRequest)
	req.SourceKey = c.Query("source_key")
	req.SourceValue = c.Query("source_value")

	service.NewAuditService().ListAudits(c, req)
}
