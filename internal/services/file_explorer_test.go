package services

import (
	"bytes"
	"context"
	"net/url"
	"testing"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type fakeRemoteCommandExecutor struct {
	seenCtx context.Context
}

func (f *fakeRemoteCommandExecutor) Stream(_ remotecommand.StreamOptions) error {
	return nil
}

func (f *fakeRemoteCommandExecutor) StreamWithContext(ctx context.Context, options remotecommand.StreamOptions) error {
	f.seenCtx = ctx
	if options.Stdout != nil {
		_, _ = options.Stdout.Write([]byte("streamed-output"))
	}
	return nil
}

func TestExecCommandStreamStdoutWithContext_UsesPassedContext(t *testing.T) {
	appCtx := &models.AppContext{
		EnvContext: models.EnvContext{
			Env: entities.Env{ClusterNamespace: "builder-ns"},
			Cluster: entities.Cluster{KubeConfig: `apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: https://127.0.0.1
users:
- name: test
  user:
    token: test-token
contexts:
- name: test
  context:
    cluster: test
    user: test
current-context: test
`},
		},
	}

	fakeExecutor := &fakeRemoteCommandExecutor{}
	originalFactory := newRemoteCommandExecutor
	newRemoteCommandExecutor = func(_ *rest.Config, _ string, _ *url.URL) (remotecommand.Executor, error) {
		return fakeExecutor, nil
	}
	t.Cleanup(func() {
		newRemoteCommandExecutor = originalFactory
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var stdout bytes.Buffer
	err := execCommandStreamStdoutWithContext(ctx, appCtx, "builder-pod", "workspace", []string{"cat", "/workspace/dist/index.html"}, &stdout)
	require.ErrorIs(t, err, context.Canceled)
	assert.Same(t, ctx, fakeExecutor.seenCtx)
	assert.Equal(t, "streamed-output", stdout.String())
}
