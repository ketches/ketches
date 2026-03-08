package models

import (
	"github.com/ketches/ketches/internal/db/entities"
)

type EnvBasic struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
}

// EnvContext encapsulates an environment and its project/cluster data
type EnvContext struct {
	Env     entities.Env
	Project entities.Project
	Cluster entities.Cluster
}
