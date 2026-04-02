package services

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"path"
	"strings"
	"testing"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/secrets"
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

type fakeWritableFSExecutor struct {
	commands []string
	dirs     map[string]struct{}
	files    map[string]string
}

func (f *fakeWritableFSExecutor) Stream(_ remotecommand.StreamOptions) error {
	return nil
}

func (f *fakeWritableFSExecutor) StreamWithContext(_ context.Context, options remotecommand.StreamOptions) error {
	if len(f.commands) == 0 {
		return nil
	}

	switch {
	case len(f.commands) >= 3 && f.commands[0] == "sh" && f.commands[1] == "-c" && strings.Contains(f.commands[2], "READONLY_DIR"):
		script := f.commands[2]
		matches := strings.Split(script, "'")
		if len(matches) < 6 {
			return errors.New("missing writable-check paths")
		}
		targetPath := path.Clean(matches[1])
		targetDir := path.Clean(matches[5])
		if _, ok := f.files[targetPath]; ok {
			return nil
		}
		if _, ok := f.dirs[targetDir]; ok {
			return nil
		}
		if options.Stdout != nil {
			_, _ = options.Stdout.Write([]byte("READONLY_DIR"))
		}
		return errors.New("directory not writable")
	case len(f.commands) >= 3 && f.commands[0] == "mkdir" && f.commands[1] == "-p":
		f.ensureDir(path.Clean(f.commands[2]))
		return nil
	case len(f.commands) >= 2 && f.commands[0] == "tee":
		if len(f.commands) < 2 {
			return errors.New("missing write path")
		}
		targetPath := path.Clean(f.commands[1])
		parentDir := path.Dir(targetPath)
		if _, ok := f.dirs[parentDir]; !ok {
			return errors.New("parent directory missing")
		}
		contentBytes, err := io.ReadAll(options.Stdin)
		if err != nil {
			return err
		}
		f.files[targetPath] = string(contentBytes)
		return nil
	default:
		return nil
	}
}

func (f *fakeWritableFSExecutor) ensureDir(dir string) {
	current := path.Clean(dir)
	for current != "/" && current != "." {
		f.dirs[current] = struct{}{}
		current = path.Dir(current)
	}
	f.dirs["/"] = struct{}{}
}

func buildFileExplorerTestAppContext(t *testing.T) *models.AppContext {
	t.Helper()

	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})
	app.Config.SecretEncryptionKey = "test-master-key"

	encryptedKubeConfig, err := secrets.EncryptString(`apiVersion: v1
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
`)
	require.NoError(t, err)

	return &models.AppContext{
		EnvContext: models.EnvContext{
			Env:     entities.Env{ClusterNamespace: "builder-ns"},
			Cluster: entities.Cluster{KubeConfig: encryptedKubeConfig},
		},
	}
}

func TestExecCommandStreamStdoutWithContext_UsesPassedContext(t *testing.T) {
	appCtx := buildFileExplorerTestAppContext(t)

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

func TestWriteFile_AllowsCreatingNestedParentDirectoriesInsideWritableWorkspace(t *testing.T) {
	appCtx := buildFileExplorerTestAppContext(t)
	fs := &fakeWritableFSExecutor{
		dirs: map[string]struct{}{
			"/":          {},
			"/workspace": {},
		},
		files: map[string]string{},
	}

	originalFactory := newRemoteCommandExecutor
	newRemoteCommandExecutor = func(_ *rest.Config, _ string, reqURL *url.URL) (remotecommand.Executor, error) {
		return &fakeWritableFSExecutor{
			commands: reqURL.Query()["command"],
			dirs:     fs.dirs,
			files:    fs.files,
		}, nil
	}
	t.Cleanup(func() {
		newRemoteCommandExecutor = originalFactory
	})

	err := WriteFile(appCtx, "builder-pod", "workspace", "/workspace/src/main.tsx", "export {}")
	require.NoError(t, err)
	assert.Contains(t, fs.files, "/workspace/src/main.tsx")
	assert.Equal(t, "export {}", fs.files["/workspace/src/main.tsx"])
}

func TestShellQuoteEscapesSingleQuotes(t *testing.T) {
	assert.Equal(t, `'a'"'"'b'`, shellQuote("a'b"))
}

func TestCompressFilesUsesTarArgumentsInsteadOfShellInterpolation(t *testing.T) {
	appCtx := buildFileExplorerTestAppContext(t)

	var seenCommands []string
	originalFactory := newRemoteCommandExecutor
	newRemoteCommandExecutor = func(_ *rest.Config, _ string, reqURL *url.URL) (remotecommand.Executor, error) {
		seenCommands = append(seenCommands, reqURL.Query()["command"]...)
		return &fakeRemoteCommandExecutor{}, nil
	}
	t.Cleanup(func() {
		newRemoteCommandExecutor = originalFactory
	})

	err := CompressFiles(appCtx, "builder-pod", "workspace", "/workspace/app", []string{"src/main.tsx", "package.json"}, "/workspace/archive.tar.gz")
	require.NoError(t, err)
	require.Equal(t, []string{"tar", "czf", "/workspace/archive.tar.gz", "-C", "/workspace/app", "src/main.tsx", "package.json"}, seenCommands)
}
