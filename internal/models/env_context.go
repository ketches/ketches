package models

import (
	"github.com/ketches/ketches/internal/db/entities"
)

// EnvContext encapsulates an environment and its project/cluster data
type EnvContext struct {
	Env     entities.Env
	Project entities.Project
	Cluster entities.Cluster
}
