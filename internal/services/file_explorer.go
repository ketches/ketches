package services

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/secrets"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

var newRemoteCommandExecutor = remotecommand.NewSPDYExecutor

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func appContextRESTConfig(appCtx *models.AppContext) (*rest.Config, error) {
	plaintextKubeConfig, err := secrets.DecryptString(appCtx.EnvContext.Cluster.KubeConfig)
	if err != nil {
		return nil, err
	}

	return clientcmd.RESTConfigFromKubeConfig([]byte(plaintextKubeConfig))
}

// execCommand executes a non-interactive command in a container and returns stdout/stderr
func execCommand(appCtx *models.AppContext, instanceName, containerName string, command []string) (string, string, error) {
	return execCommandWithContext(context.Background(), appCtx, instanceName, containerName, command)
}

func execCommandWithContext(ctx context.Context, appCtx *models.AppContext, instanceName, containerName string, command []string) (string, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	config, err := appContextRESTConfig(appCtx)
	if err != nil {
		return "", "", err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", "", err
	}
	if _, err := validateAppPodContainer(ctx, client, appCtx, instanceName, containerName); err != nil {
		return "", "", err
	}

	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(instanceName).
		Namespace(appCtx.EnvContext.Env.ClusterNamespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   command,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := newRemoteCommandExecutor(config, "POST", req.URL())
	if err != nil {
		return "", "", err
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if ctx.Err() != nil {
		return stdout.String(), stderr.String(), ctx.Err()
	}

	return stdout.String(), stderr.String(), err
}

// execCommandWithStdin executes a command in a container with stdin input
func execCommandWithStdin(appCtx *models.AppContext, instanceName, containerName string, command []string, stdin io.Reader) (string, string, error) {
	config, err := appContextRESTConfig(appCtx)
	if err != nil {
		return "", "", err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", "", err
	}
	if _, err := validateAppPodContainer(context.Background(), client, appCtx, instanceName, containerName); err != nil {
		return "", "", err
	}

	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(instanceName).
		Namespace(appCtx.EnvContext.Env.ClusterNamespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := newRemoteCommandExecutor(config, "POST", req.URL())
	if err != nil {
		return "", "", err
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: &stdout,
		Stderr: &stderr,
	})

	return stdout.String(), stderr.String(), err
}

// execCommandStreamStdout executes a command and streams stdout to the provided writer
func execCommandStreamStdout(appCtx *models.AppContext, instanceName, containerName string, command []string, stdout io.Writer) error {
	return execCommandStreamStdoutWithContext(context.Background(), appCtx, instanceName, containerName, command, stdout)
}

func execCommandStreamStdoutWithContext(ctx context.Context, appCtx *models.AppContext, instanceName, containerName string, command []string, stdout io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}

	config, err := appContextRESTConfig(appCtx)
	if err != nil {
		return err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	if _, err := validateAppPodContainer(ctx, client, appCtx, instanceName, containerName); err != nil {
		return err
	}

	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(instanceName).
		Namespace(appCtx.EnvContext.Env.ClusterNamespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   command,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := newRemoteCommandExecutor(config, "POST", req.URL())
	if err != nil {
		return err
	}

	var stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: stdout,
		Stderr: &stderr,
	})
	if ctx.Err() != nil {
		return ctx.Err()
	}

	return err
}

// execCommandWithStdinStream executes a command with stdin stream and stdout writer
func execCommandWithStdinStream(appCtx *models.AppContext, instanceName, containerName string, command []string, stdin io.Reader, stdout io.Writer) error {
	config, err := appContextRESTConfig(appCtx)
	if err != nil {
		return err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	if _, err := validateAppPodContainer(context.Background(), client, appCtx, instanceName, containerName); err != nil {
		return err
	}

	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(instanceName).
		Namespace(appCtx.EnvContext.Env.ClusterNamespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := newRemoteCommandExecutor(config, "POST", req.URL())
	if err != nil {
		return err
	}

	var stderr bytes.Buffer
	return exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: &stderr,
	})
}

// GetHomeDir returns the home directory ($HOME) of the container's default user
func GetHomeDir(appCtx *models.AppContext, instanceName, containerName string) (string, error) {
	stdout, _, err := execCommand(appCtx, instanceName, containerName, []string{"sh", "-c", `echo "$HOME"`})
	if err != nil {
		if errors.Is(err, errPodAccessDenied) {
			return "", err
		}
		return "/", nil // fallback to root when the container shell cannot report HOME
	}
	home := strings.TrimSpace(stdout)
	if home == "" {
		return "/", nil
	}
	return home, nil
}

// CheckWritable checks if a path is writable in the container
func CheckWritable(appCtx *models.AppContext, instanceName, containerName, path string) error {
	path = filepath.Clean(path)
	dir := filepath.Dir(path)

	// Check if the target directory is writable
	script := fmt.Sprintf(`if [ -e %s ]; then
  if [ -w %s ]; then exit 0; else echo "READONLY_FILE"; exit 1; fi
elif [ -w %s ]; then exit 0
else echo "READONLY_DIR"; exit 1; fi`, shellQuote(path), shellQuote(path), shellQuote(dir))

	stdout, _, err := execCommand(appCtx, instanceName, containerName, []string{"sh", "-c", script})
	if err != nil {
		msg := strings.TrimSpace(stdout)
		if msg == "READONLY_FILE" {
			return app.NewErrorf("file is on a read-only file system")
		}
		if msg == "READONLY_DIR" {
			return app.NewErrorf("directory is on a read-only file system")
		}
		return app.NewErrorf("path is not writable")
	}
	return nil
}

// CompressFiles compresses multiple files into a tar.gz archive inside the container
func CompressFiles(appCtx *models.AppContext, instanceName, containerName, baseDir string, fileNames []string, destPath string) error {
	baseDir = filepath.Clean(baseDir)
	destPath = filepath.Clean(destPath)

	command := []string{"tar", "czf", destPath, "-C", baseDir}
	for _, name := range fileNames {
		command = append(command, filepath.Clean(name))
	}

	_, stderr, err := execCommand(appCtx, instanceName, containerName, command)
	if err != nil {
		return app.NewErrorf("failed to compress files: %v, stderr: %s", err, stderr)
	}
	return nil
}

// CompressAndDownloadFiles compresses multiple files and streams the tar.gz to the writer
func CompressAndDownloadFiles(appCtx *models.AppContext, instanceName, containerName, baseDir string, fileNames []string, writer io.Writer) error {
	baseDir = filepath.Clean(baseDir)

	command := []string{"tar", "czf", "-", "-C", baseDir}
	for _, name := range fileNames {
		command = append(command, filepath.Clean(name))
	}

	return execCommandStreamStdout(
		appCtx, instanceName, containerName,
		command,
		writer,
	)
}

// ListFiles lists files in a directory inside a container
func ListFiles(appCtx *models.AppContext, instanceName, containerName, path string) (*models.ListFilesResponse, error) {
	if path == "" {
		path = "/"
	}
	// Normalize path to avoid path traversal
	path = filepath.Clean(path)

	// Use a shell script that works across most containers (bash, ash, sh)
	// Output format: name\ttype\tsize\tmodtime\tpermissions
	script := fmt.Sprintf(`dir=%s
if [ ! -d "$dir" ]; then echo "ERROR: not a directory"; exit 1; fi
for f in "$dir"/* "$dir"/.*; do
  [ -e "$f" ] || [ -L "$f" ] || continue
  name=$(basename "$f")
  [ "$name" = "." ] || [ "$name" = ".." ] && continue
  if [ -L "$f" ]; then t="link"
  elif [ -d "$f" ]; then t="dir"
  else t="file"
  fi
  s=$(stat -c %%s "$f" 2>/dev/null || stat -f %%z "$f" 2>/dev/null || echo 0)
  m=$(stat -c %%Y "$f" 2>/dev/null || stat -f %%m "$f" 2>/dev/null || echo 0)
  p=$(stat -c %%a "$f" 2>/dev/null || stat -f %%Lp "$f" 2>/dev/null || echo 644)
  printf '%%s\t%%s\t%%s\t%%s\t%%s\n' "$name" "$t" "$s" "$m" "$p"
done`, shellQuote(path))

	stdout, stderr, err := execCommand(appCtx, instanceName, containerName, []string{"sh", "-c", script})
	if err != nil {
		if strings.Contains(stderr, "ERROR: not a directory") {
			return nil, app.NewErrorf("path is not a directory: %s", path)
		}
		return nil, app.NewErrorf("failed to list files: %v, stderr: %s", err, stderr)
	}

	files := []models.FileInfo{}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 5)
		if len(parts) < 5 {
			continue
		}

		size, _ := strconv.ParseInt(parts[2], 10, 64)
		modTime, _ := strconv.ParseInt(parts[3], 10, 64)

		files = append(files, models.FileInfo{
			Name:        parts[0],
			Type:        parts[1],
			Size:        size,
			ModTime:     modTime,
			Permissions: parts[4],
		})
	}

	return &models.ListFilesResponse{
		Path:  path,
		Files: files,
	}, nil
}

// ReadFile reads the content of a text file inside a container
func ReadFile(appCtx *models.AppContext, instanceName, containerName, path string) (*models.ReadFileResponse, error) {
	path = filepath.Clean(path)

	// First check if it's a file and get its size
	checkScript := fmt.Sprintf(`f=%s
if [ ! -e "$f" ]; then echo "ERROR: file not found"; exit 1; fi
if [ -d "$f" ]; then echo "ERROR: is a directory"; exit 1; fi
s=$(stat -c %%s "$f" 2>/dev/null || stat -f %%z "$f" 2>/dev/null || echo 0)
echo "$s"`, shellQuote(path))

	stdout, stderr, err := execCommand(appCtx, instanceName, containerName, []string{"sh", "-c", checkScript})
	if err != nil {
		if strings.Contains(stderr, "ERROR:") || strings.Contains(stdout, "ERROR:") {
			msg := strings.TrimSpace(stderr)
			if msg == "" {
				msg = strings.TrimSpace(stdout)
			}
			return nil, app.NewErrorf("%s", msg)
		}
		return nil, app.NewErrorf("failed to check file: %v", err)
	}

	size, _ := strconv.ParseInt(strings.TrimSpace(stdout), 10, 64)

	// Limit file size to 5MB for text reading
	const maxReadSize = 5 * 1024 * 1024
	if size > maxReadSize {
		return nil, app.NewErrorf("file too large to read: %d bytes (max %d bytes)", size, maxReadSize)
	}

	// Read the file content
	content, stderr, err := execCommand(appCtx, instanceName, containerName, []string{"cat", path})
	if err != nil {
		return nil, app.NewErrorf("failed to read file: %v, stderr: %s", err, stderr)
	}

	return &models.ReadFileResponse{
		Path:    path,
		Content: content,
		Size:    size,
	}, nil
}

// WriteFile writes content to a file inside a container
func WriteFile(appCtx *models.AppContext, instanceName, containerName, path, content string) error {
	path = filepath.Clean(path)

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if dir != "/" && dir != "." {
		_, stderr, err := execCommand(appCtx, instanceName, containerName, []string{"mkdir", "-p", dir})
		if err != nil {
			if strings.Contains(stderr, "Read-only file system") {
				return app.NewErrorf("read-only file system: cannot create %s", dir)
			}
			return app.NewErrorf("failed to create parent directory: %v, stderr: %s", err, stderr)
		}
	}

	// Use tee with stdin so the target path stays an argv entry instead of part
	// of a shell command string.
	_, stderr, err := execCommandWithStdin(
		appCtx, instanceName, containerName,
		[]string{"tee", path},
		strings.NewReader(content),
	)
	if err != nil {
		if strings.Contains(stderr, "Read-only file system") {
			return app.NewErrorf("read-only file system: cannot write to %s", path)
		}
		return app.NewErrorf("failed to write file: %v, stderr: %s", err, stderr)
	}

	return nil
}

// MkdirInContainer creates a directory inside a container
func MkdirInContainer(appCtx *models.AppContext, instanceName, containerName, path string) error {
	path = filepath.Clean(path)

	_, stderr, err := execCommand(appCtx, instanceName, containerName, []string{"mkdir", "-p", path})
	if err != nil {
		return app.NewErrorf("failed to create directory: %v, stderr: %s", err, stderr)
	}

	return nil
}

// DeleteFileInContainer deletes a file or directory inside a container
func DeleteFileInContainer(appCtx *models.AppContext, instanceName, containerName, path string) error {
	path = filepath.Clean(path)

	// Safety: prevent deleting root
	if path == "/" || path == "" {
		return app.NewErrorf("cannot delete root directory")
	}

	_, stderr, err := execCommand(appCtx, instanceName, containerName, []string{"rm", "-rf", path})
	if err != nil {
		return app.NewErrorf("failed to delete: %v, stderr: %s", err, stderr)
	}

	return nil
}

// MoveFileInContainer moves/renames a file or directory inside a container
func MoveFileInContainer(appCtx *models.AppContext, instanceName, containerName, source, destination string) error {
	source = filepath.Clean(source)
	destination = filepath.Clean(destination)

	_, stderr, err := execCommand(appCtx, instanceName, containerName, []string{"mv", source, destination})
	if err != nil {
		return app.NewErrorf("failed to move: %v, stderr: %s", err, stderr)
	}

	return nil
}

// CopyFileInContainer copies a file or directory inside a container
func CopyFileInContainer(appCtx *models.AppContext, instanceName, containerName, source, destination string) error {
	source = filepath.Clean(source)
	destination = filepath.Clean(destination)

	_, stderr, err := execCommand(appCtx, instanceName, containerName, []string{"cp", "-r", source, destination})
	if err != nil {
		return app.NewErrorf("failed to copy: %v, stderr: %s", err, stderr)
	}

	return nil
}

// DownloadFile streams a file from a container to the writer using tar
func DownloadFile(appCtx *models.AppContext, instanceName, containerName, path string, writer io.Writer) error {
	path = filepath.Clean(path)
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	return execCommandStreamStdout(
		appCtx, instanceName, containerName,
		[]string{"tar", "cf", "-", "-C", dir, base},
		writer,
	)
}

// DownloadFileContents streams the first tar entry without buffering the
// archive in memory.
func DownloadFileContents(appCtx *models.AppContext, instanceName, containerName, path string, writer io.Writer) (int64, error) {
	pipeReader, pipeWriter := io.Pipe()
	errorCh := make(chan error, 1)
	go func() {
		err := DownloadFile(appCtx, instanceName, containerName, path, pipeWriter)
		_ = pipeWriter.CloseWithError(err)
		errorCh <- err
	}()

	reader := tar.NewReader(pipeReader)
	header, err := reader.Next()
	if err != nil {
		_ = pipeReader.CloseWithError(err)
		<-errorCh
		return 0, err
	}
	written, copyErr := io.Copy(writer, reader)
	if copyErr != nil {
		_ = pipeReader.CloseWithError(copyErr)
		<-errorCh
		return written, copyErr
	}
	if _, err := io.Copy(io.Discard, pipeReader); err != nil {
		<-errorCh
		return written, err
	}
	if err := <-errorCh; err != nil {
		return written, err
	}
	return header.Size, nil
}

// UploadFile uploads a file to the container using tar
func UploadFile(appCtx *models.AppContext, instanceName, containerName, destDir, fileName string, fileContent io.Reader, fileSize int64) error {
	if fileSize < 0 {
		return app.NewErrorf("upload file size must be known")
	}
	destDir = filepath.Clean(destDir)
	fileName = filepath.Base(fileName)
	if fileName == "." || fileName == "/" || fileName == "" {
		fileName = "upload"
	}

	pipeReader, pipeWriter := io.Pipe()
	errorCh := make(chan error, 1)
	go func() {
		tw := tar.NewWriter(pipeWriter)
		err := tw.WriteHeader(&tar.Header{Name: fileName, Mode: 0644, Size: fileSize})
		if err == nil {
			_, err = io.CopyN(tw, fileContent, fileSize)
		}
		if closeErr := tw.Close(); err == nil {
			err = closeErr
		}
		_ = pipeWriter.CloseWithError(err)
		errorCh <- err
	}()

	// Extract the tar archive in the destination directory while the archive is produced.
	err := execCommandWithStdinStream(
		appCtx, instanceName, containerName,
		[]string{"tar", "xf", "-", "-C", destDir},
		pipeReader,
		io.Discard,
	)
	_ = pipeReader.CloseWithError(err)
	producerErr := <-errorCh
	if err != nil {
		return err
	}
	if producerErr != nil {
		return app.NewErrorf("failed to create upload archive: %v", producerErr)
	}
	return nil
}
