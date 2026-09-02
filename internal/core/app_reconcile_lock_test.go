package core

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReconcileLatestAppContextReloadsAndRepeatsWhenDesiredStateChanges(t *testing.T) {
	policy := &models.PodAccessPolicy{RequiredLabels: map[string]string{"scope": "workspace"}}
	original := &models.AppContext{
		App:             entities.App{Base: entities.Base{ID: "app-1"}, ContainerImage: "stale:v0"},
		PodAccessPolicy: policy,
	}
	snapshots := []models.AppContext{
		{App: entities.App{Base: entities.Base{ID: "app-1"}, ContainerImage: "current:v1"}},
		{App: entities.App{Base: entities.Base{ID: "app-1"}, ContainerImage: "newer:v2"}},
		{App: entities.App{Base: entities.Base{ID: "app-1"}, ContainerImage: "newer:v2"}},
	}
	loadCalls := 0
	loader := func(_ context.Context, appID string) (*models.AppContext, error) {
		require.Equal(t, "app-1", appID)
		require.Less(t, loadCalls, len(snapshots))
		result := snapshots[loadCalls]
		loadCalls++
		return &result, nil
	}

	var appliedImages []string
	err := reconcileLatestAppContext(context.Background(), original, loader, func(latest *models.AppContext) error {
		assert.Same(t, policy, latest.PodAccessPolicy)
		appliedImages = append(appliedImages, latest.App.ContainerImage)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"current:v1", "newer:v2"}, appliedImages)
	assert.Equal(t, 3, loadCalls)
	assert.Equal(t, "newer:v2", original.App.ContainerImage)
	assert.Same(t, policy, original.PodAccessPolicy)
}

func TestReconcileLatestAppContextNeverAppliesPreLockContext(t *testing.T) {
	original := &models.AppContext{App: entities.App{
		Base:           entities.Base{ID: "app-1"},
		ContainerImage: "stale:v1",
	}}
	fresh := models.AppContext{App: entities.App{
		Base:           entities.Base{ID: "app-1"},
		ContainerImage: "fresh:v2",
	}}
	loader := func(context.Context, string) (*models.AppContext, error) {
		result := fresh
		return &result, nil
	}

	var applied string
	require.NoError(t, reconcileLatestAppContext(context.Background(), original, loader, func(latest *models.AppContext) error {
		applied = latest.App.ContainerImage
		return nil
	}))
	assert.Equal(t, "fresh:v2", applied)
}

func TestAppContextReconcileRevisionIgnoresTransientPodAccessPolicy(t *testing.T) {
	first := &models.AppContext{
		App:             entities.App{Base: entities.Base{ID: "app-1"}},
		PodAccessPolicy: &models.PodAccessPolicy{RequiredLabels: map[string]string{"workspace": "one"}},
	}
	second := &models.AppContext{
		App:             entities.App{Base: entities.Base{ID: "app-1"}},
		PodAccessPolicy: &models.PodAccessPolicy{RequiredLabels: map[string]string{"workspace": "two"}},
	}

	firstRevision, err := appContextReconcileRevision(first)
	require.NoError(t, err)
	secondRevision, err := appContextReconcileRevision(second)
	require.NoError(t, err)
	assert.Equal(t, firstRevision, secondRevision)
}

func TestAppReconcileFenceSerializesMutationAndReconcile(t *testing.T) {
	originalDB := db.DB
	db.DB = nil
	t.Cleanup(func() { db.DB = originalDB })

	fenceHeld := make(chan struct{})
	releaseFence := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- WithAppReconcileFence(context.Background(), "app-serialized", func() error {
			close(fenceHeld)
			<-releaseFence
			return nil
		})
	}()

	select {
	case <-fenceHeld:
	case <-time.After(5 * time.Second):
		t.Fatal("mutation fence was not acquired")
	}

	reconcileStarted := make(chan struct{})
	secondDone := make(chan error, 1)
	var startOnce sync.Once
	go func() {
		secondDone <- withAppReconcileLockContext(context.Background(), &models.AppContext{
			App: entities.App{Base: entities.Base{ID: "app-serialized"}},
		}, func(*models.AppContext) error {
			startOnce.Do(func() { close(reconcileStarted) })
			return nil
		})
	}()

	select {
	case <-reconcileStarted:
		t.Fatal("reconcile entered while the mutation fence was held")
	case <-time.After(100 * time.Millisecond):
	}

	close(releaseFence)
	require.NoError(t, <-firstDone)
	select {
	case <-reconcileStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("reconcile did not enter after the mutation fence was released")
	}
	require.NoError(t, <-secondDone)
}
