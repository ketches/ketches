package app

import (
	dbtypes "github.com/ketches/ketches/db/types"
)

func CreateApplication(req CreateAppReq) (CreateAppResp, error) {
	var (
		res CreateAppResp
	)

	// if err := utils.ValidateResourceName(r.Name); err != nil {
	// 	return res, err
	// }

	app := &dbtypes.App{
		Name:        req.Name,
		ClusterID:   req.ClusterID,
		NamespaceId: r.NamespaceId,
		Description: r.Description,
		//Tags:           "", // TODO
		Status:         enums.ApplicationStatusNascent,
		CreatorId:      0, // TODO
		LastOperatorId: 0, // TODO
		TenantId:       r.TenantId,
	}

	repos.Application().Create(app)

	res = CreateAppResp{
		Id:        app.Id,
		Name:      app.Name,
		Desc:      app.Desc,
		ClusterID: app.ClusterID,
		Namespace: app.Namespace,
		Status:    uint8(app.Status),
		TenantID:  app.TenantID,
	}

	return res, nil
}
