package services

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
	"testing"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type fakeRemoteCommandExecutor struct {
	seenCtx context.Context
}

type tarStreamExecutor struct {
	stdout []byte
	stdin  []byte
}

func (f *tarStreamExecutor) Stream(options remotecommand.StreamOptions) error {
	return f.StreamWithContext(context.Background(), options)
}

func (f *tarStreamExecutor) StreamWithContext(_ context.Context, options remotecommand.StreamOptions) error {
	if options.Stdout != nil && len(f.stdout) > 0 {
		if _, err := options.Stdout.Write(f.stdout); err != nil {
			return err
		}
	}
	if options.Stdin != nil {
		data, err := io.ReadAll(options.Stdin)
		if err != nil {
			return err
		}
		f.stdin = data
	}
	return nil
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
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "workspace-pod",
			Namespace: "workspace-ns",
			Labels: map[string]string{
				kube.LabelManagedBy:         "true",
				"ketches.cn/workspace-role": "interactive",
				"ketches.cn/workspace-id":   "workspace-1",
			},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "workspace"}}},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/namespaces/workspace-ns/pods/workspace-pod" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(pod))
	}))
	t.Cleanup(server.Close)

	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})
	app.Config.SecretEncryptionKey = "test-master-key"

	kubeConfig := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: %s
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
`, server.URL)
	encryptedKubeConfig, err := secrets.EncryptString(kubeConfig)
	require.NoError(t, err)

	return &models.AppContext{
		PodAccessPolicy: &models.PodAccessPolicy{RequiredLabels: map[string]string{
			"ketches.cn/workspace-role": "interactive",
			"ketches.cn/workspace-id":   "workspace-1",
		}},
		EnvContext: models.EnvContext{
			Env:     entities.Env{ClusterNamespace: "workspace-ns"},
			Cluster: entities.Cluster{KubeConfig: encryptedKubeConfig},
		},
	}
}

func TestExecCommandStreamStdoutWithContext_StopsBeforeExecWhenContextCanceled(t *testing.T) {
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
	err := execCommandStreamStdoutWithContext(ctx, appCtx, "workspace-pod", "workspace", []string{"cat", "/workspace/dist/index.html"}, &stdout)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, fakeExecutor.seenCtx)
	assert.Empty(t, stdout.String())
}

func TestExecCommandRejectsPodOutsideApplicationBeforeExec(t *testing.T) {
	appCtx := buildFileExplorerTestAppContext(t)
	appCtx.App.ID = "app-1"

	executorCreated := false
	originalFactory := newRemoteCommandExecutor
	newRemoteCommandExecutor = func(_ *rest.Config, _ string, _ *url.URL) (remotecommand.Executor, error) {
		executorCreated = true
		return &fakeRemoteCommandExecutor{}, nil
	}
	t.Cleanup(func() {
		newRemoteCommandExecutor = originalFactory
	})

	_, _, err := execCommand(appCtx, "workspace-pod", "workspace", []string{"id"})
	require.ErrorIs(t, err, errPodAccessDenied)
	assert.False(t, executorCreated)
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

	err := WriteFile(appCtx, "workspace-pod", "workspace", "/workspace/src/main.tsx", "export {}")
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

	err := CompressFiles(appCtx, "workspace-pod", "workspace", "/workspace/app", []string{"src/main.tsx", "package.json"}, "/workspace/archive.tar.gz")
	require.NoError(t, err)
	require.Equal(t, []string{"tar", "czf", "/workspace/archive.tar.gz", "-C", "/workspace/app", "src/main.tsx", "package.json"}, seenCommands)
}

func TestDownloadFileContentsStreamsTarEntry(t *testing.T) {
	appCtx := buildFileExplorerTestAppContext(t)
	var archive bytes.Buffer
	tarWriter := tar.NewWriter(&archive)
	content := bytes.Repeat([]byte("download-data"), 1024)
	require.NoError(t, tarWriter.WriteHeader(&tar.Header{Name: "artifact.bin", Mode: 0600, Size: int64(len(content))}))
	_, err := tarWriter.Write(content)
	require.NoError(t, err)
	require.NoError(t, tarWriter.Close())

	executor := &tarStreamExecutor{stdout: archive.Bytes()}
	originalFactory := newRemoteCommandExecutor
	newRemoteCommandExecutor = func(_ *rest.Config, _ string, _ *url.URL) (remotecommand.Executor, error) {
		return executor, nil
	}
	t.Cleanup(func() { newRemoteCommandExecutor = originalFactory })

	var output bytes.Buffer
	size, err := DownloadFileContents(appCtx, "workspace-pod", "workspace", "/workspace/artifact.bin", &output)
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), size)
	assert.Equal(t, content, output.Bytes())
}

func TestUploadFileStreamsTarArchive(t *testing.T) {
	appCtx := buildFileExplorerTestAppContext(t)
	content := bytes.Repeat([]byte("upload-data"), 1024)
	executor := &tarStreamExecutor{}
	originalFactory := newRemoteCommandExecutor
	newRemoteCommandExecutor = func(_ *rest.Config, _ string, _ *url.URL) (remotecommand.Executor, error) {
		return executor, nil
	}
	t.Cleanup(func() { newRemoteCommandExecutor = originalFactory })

	err := UploadFile(appCtx, "workspace-pod", "workspace", "/workspace", "artifact.bin", bytes.NewReader(content), int64(len(content)))
	require.NoError(t, err)

	tarReader := tar.NewReader(bytes.NewReader(executor.stdin))
	header, err := tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "artifact.bin", header.Name)
	assert.Equal(t, int64(len(content)), header.Size)
	uploaded, err := io.ReadAll(tarReader)
	require.NoError(t, err)
	assert.Equal(t, content, uploaded)
}
