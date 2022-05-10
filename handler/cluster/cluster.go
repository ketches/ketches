package cluster

func GetCluster(req GetClusterRequest) (GetClusterResponse, error) {
	var (
		cluster entities.Cluster
		res     GetClusterResponse
	)
	err := repos.NsoKube.Get(req.ClusterId, &cluster)
	if err != nil {
		return res, errs.ResourceNotFound()
	}

	var lastOperator entities.User
	repos.User().Get(cluster.LastOperatorId, &lastOperator)
	res = GetClusterResponse{
		SingleClusterResponse{
			Id:           cluster.Id,
			Description:  cluster.Description,
			Formatting:   cluster.Formatting,
			KubeConfig:   cluster.KubeConfig,
			LastOperator: lastOperator.FullName(),
		},
	}

	return res, nil
}

func CreateCluster(r CreateClusterRequest) (CreateClusterResponse, error) {
	var (
		res CreateClusterResponse
	)
	if len(r.Name) == 0 {
		return res, errs.InvalidRequest()
	}
	cluster := &entities.Cluster{
		Name:           r.Name,
		Description:    r.Description,
		KubeConfig:     r.KubeConfig,
		Formatting:     r.Formatting,
		LastOperatorId: r.LastOperatorId, //uint64(util.StringToInt64((ctx.Value("UserId")).(string))),
	}
	err := repos.Cluster().Create(cluster)
	if err != nil {
		return res, err
	}

	var lastOperator entities.User
	repos.User().Get(cluster.LastOperatorId, &lastOperator)

	res = CreateClusterResponse{
		SingleClusterResponse: SingleClusterResponse{
			Id:           cluster.Id,
			Name:         cluster.Name,
			Description:  cluster.Description,
			KubeConfig:   cluster.KubeConfig,
			Formatting:   cluster.Formatting,
			LastOperator: lastOperator.FullName(),
		},
	}

	return res, nil
}

func UpdateCluster(r UpdateClusterRequest) (UpdateClusterResponse, error) {
	var (
		res UpdateClusterResponse
	)
	if len(r.Name) == 0 {
		return res, errs.InvalidRequest()
	}
	cluster := entities.Cluster{
		Name:           r.Name,
		Description:    r.Description,
		KubeConfig:     r.KubeConfig,
		Formatting:     r.Formatting,
		LastOperatorId: r.LastOperatorId,
	}
	err := repos.Cluster().Update(&cluster, orm.UpdateOption{
		Columns: []string{"name", "display_name", "kube_config", "description", "formatting", "last_operator_id"},
	})
	if err != nil {
		return res, err
	}

	var lastOperatorId entities.User
	repos.User().Get(cluster.LastOperatorId, &lastOperatorId)

	res = UpdateClusterResponse{
		SingleClusterResponse{
			Id:           cluster.Id,
			Name:         cluster.Name,
			Description:  cluster.Description,
			KubeConfig:   cluster.KubeConfig,
			Formatting:   cluster.Formatting,
			LastOperator: lastOperatorId.FullName(),
		},
	}

	return res, nil
}

func PagingClusters(r PagingClusterRequest) (PagingClusterResponse, error) {
	var (
		clusters []entities.Cluster
		res      PagingClusterResponse
	)
	total, err := repos.Cluster().Page(&clusters, orm.PageOption{
		No:        r.No,
		Size:      r.Size,
		OrderBy:   r.OrderBy,
		Desc:      r.Desc,
		Condition: orm.Condition("name like ?", "%"+r.Filter+"%"),
	})
	if err != nil {
		return res, err
	}

	var pagingClusters []SingleClusterResponse
	for _, cluster := range clusters {
		var lastOperatorId entities.User
		repos.User().Get(cluster.LastOperatorId, &lastOperatorId)
		pagingClusters = append(pagingClusters, SingleClusterResponse{
			Id:           cluster.Id,
			Name:         cluster.Name,
			Description:  cluster.Description,
			KubeConfig:   cluster.KubeConfig,
			Formatting:   cluster.Formatting,
			LastOperator: lastOperatorId.FullName(),
		})
	}

	res = PagingClusterResponse{
		Total:    total,
		Clusters: pagingClusters,
	}

	return res, nil
}