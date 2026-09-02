package entities

import (
	"database/sql"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestEnvClusterNamespaceNullableSerializer(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&Env{}))

	require.NoError(t, testDB.Exec(`INSERT INTO envs (id, slug, name, project_id, cluster_id, cluster_namespace, is_build_env) VALUES (?, ?, ?, ?, ?, NULL, ?)`,
		"env-null", "null", "Null", "project-1", "cluster-1", false).Error)

	var env Env
	require.NoError(t, testDB.First(&env, "id = ?", "env-null").Error)
	assert.Empty(t, env.ClusterNamespace)

	emptyNamespace := Env{
		Base:             Base{ID: "env-empty"},
		Slug:             "empty",
		Name:             "Empty",
		ProjectID:        "project-1",
		ClusterID:        "cluster-1",
		ClusterNamespace: "",
	}
	require.NoError(t, testDB.Create(&emptyNamespace).Error)

	var raw sql.NullString
	require.NoError(t, testDB.Raw("SELECT cluster_namespace FROM envs WHERE id = ?", emptyNamespace.ID).Scan(&raw).Error)
	assert.False(t, raw.Valid, "an empty namespace should be stored as SQL NULL")

	namedNamespace := Env{
		Base:             Base{ID: "env-named"},
		Slug:             "named",
		Name:             "Named",
		ProjectID:        "project-1",
		ClusterID:        "cluster-1",
		ClusterNamespace: "production",
	}
	require.NoError(t, testDB.Create(&namedNamespace).Error)

	var loaded Env
	require.NoError(t, testDB.First(&loaded, "id = ?", namedNamespace.ID).Error)
	assert.Equal(t, "production", loaded.ClusterNamespace)
}
