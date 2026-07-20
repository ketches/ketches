package services

import (
	"context"
	"errors"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/egress"
)

var ErrInvalidGitRepositoryURL = errors.New("git repository URL must use https, ssh, or git@")

const (
	gitCommandTimeout     = 2 * time.Minute
	gitCommandOutputLimit = 1 << 20
)

type validatedGitEndpoint struct {
	URL       *url.URL
	Host      string
	Port      string
	Addresses []net.IP
	Scheme    string
}

func validateGitRepositoryURL(repoURL string) error {
	trimmed := strings.TrimSpace(repoURL)
	if trimmed == "" {
		return ErrInvalidGitRepositoryURL
	}
	normalized, err := normalizeGitRepositoryURL(trimmed)
	if err != nil {
		return ErrInvalidGitRepositoryURL
	}
	parsed, err := egress.CurrentPolicy().ValidateURLSyntax(normalized, "https", "ssh")
	if err != nil {
		return ErrInvalidGitRepositoryURL
	}
	if parsed.Scheme == "https" && parsed.User != nil {
		return ErrInvalidGitRepositoryURL
	}
	if parsed.Fragment != "" {
		return ErrInvalidGitRepositoryURL
	}
	return nil
}

func normalizeGitRepositoryURL(repoURL string) (string, error) {
	trimmed := strings.TrimSpace(repoURL)
	if strings.HasPrefix(trimmed, "git@") {
		at := strings.IndexByte(trimmed, '@')
		if at < 0 {
			return "", ErrInvalidGitRepositoryURL
		}
		colon := strings.IndexByte(trimmed[at+1:], ':')
		if colon < 1 {
			return "", ErrInvalidGitRepositoryURL
		}
		colon += at + 1
		host := trimmed[at+1 : colon]
		repositoryPath := strings.TrimPrefix(trimmed[colon+1:], "/")
		if repositoryPath == "" || strings.ContainsAny(host, "/?#") {
			return "", ErrInvalidGitRepositoryURL
		}
		return "ssh://git@" + host + "/" + repositoryPath, nil
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", ErrInvalidGitRepositoryURL
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	if parsed.Scheme != "https" && parsed.Scheme != "ssh" {
		return "", ErrInvalidGitRepositoryURL
	}
	return parsed.String(), nil
}

func resolveGitRepositoryEndpoint(ctx context.Context, repoURL string) (*validatedGitEndpoint, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	normalized, err := normalizeGitRepositoryURL(repoURL)
	if err != nil {
		return nil, err
	}
	endpoint, err := egress.CurrentPolicy().ValidateURL(ctx, normalized, "https", "ssh")
	if err != nil {
		return nil, app.WrapErrorf(err, "unsafe git repository URL: %w", err)
	}
	if endpoint.URL.Scheme == "https" && endpoint.URL.User != nil {
		return nil, ErrInvalidGitRepositoryURL
	}
	port := endpoint.URL.Port()
	if port == "" {
		port = "443"
		if endpoint.URL.Scheme == "ssh" {
			port = "22"
		}
	}
	addresses := make([]net.IP, 0, len(endpoint.Addresses))
	for _, address := range endpoint.Addresses {
		addresses = append(addresses, net.ParseIP(address.String()))
	}
	return &validatedGitEndpoint{
		URL:       endpoint.URL,
		Host:      endpoint.Host,
		Port:      port,
		Addresses: addresses,
		Scheme:    endpoint.URL.Scheme,
	}, nil
}

func runGitRemoteCommand(ctx context.Context, dir string, endpoint *validatedGitEndpoint, args ...string) ([]byte, error) {
	if endpoint == nil {
		return nil, ErrInvalidGitRepositoryURL
	}
	commandArgs := []string{"-c", "http.followRedirects=false", "-c", "protocol.allow=never", "-c", "protocol.https.allow=always", "-c", "protocol.ssh.allow=always"}
	if endpoint.Scheme == "https" {
		resolved := make([]string, 0, len(endpoint.Addresses))
		for _, address := range endpoint.Addresses {
			value := address.String()
			if address.To4() == nil {
				value = "[" + value + "]"
			}
			resolved = append(resolved, value)
		}
		if net.ParseIP(endpoint.Host) == nil {
			commandArgs = append(commandArgs, "-c", "http.curloptResolve="+endpoint.Host+":"+endpoint.Port+":"+strings.Join(resolved, ","))
		}
	} else {
		commandArgs = append(commandArgs, "-c", "ssh.variant=ssh")
	}
	commandArgs = append(commandArgs, args...)
	output, err := runGitCommand(ctx, dir, endpoint, commandArgs...)
	return redactGitCommandOutput(output, args), err
}

func runGitCommand(ctx context.Context, dir string, endpoint *validatedGitEndpoint, args ...string) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	commandCtx, cancel := context.WithTimeout(ctx, gitCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(commandCtx, "git", args...)
	cmd.Dir = dir
	cmd.Env = hardenedGitEnvironment(endpoint)
	var stdout, stderr limitedGitBuffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if commandCtx.Err() != nil {
		return append(stdout.Bytes(), stderr.Bytes()...), commandCtx.Err()
	}
	return append(stdout.Bytes(), stderr.Bytes()...), err
}

func hardenedGitEnvironment(endpoint *validatedGitEndpoint) []string {
	env := append([]string(nil), os.Environ()...)
	env = append(env,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_PROTOCOL_FROM_USER=0",
		"GIT_ASKPASS=/bin/false",
		"SSH_ASKPASS=/bin/false",
	)
	if endpoint != nil && endpoint.Scheme == "ssh" && len(endpoint.Addresses) > 0 {
		env = append(env, "GIT_SSH_COMMAND=ssh -o BatchMode=yes -o StrictHostKeyChecking=accept-new -o ConnectTimeout=10 -o ConnectionAttempts=1 -o HostKeyAlias="+endpoint.Host+" -o HostName="+endpoint.Addresses[0].String()+" -o Port="+endpoint.Port)
	}
	return env
}

type limitedGitBuffer struct {
	bytes []byte
	full  bool
}

func (b *limitedGitBuffer) Write(p []byte) (int, error) {
	remaining := gitCommandOutputLimit - len(b.bytes)
	if remaining <= 0 {
		b.full = true
		return len(p), nil
	}
	if len(p) > remaining {
		b.bytes = append(b.bytes, p[:remaining]...)
		b.full = true
		return len(p), nil
	}
	b.bytes = append(b.bytes, p...)
	return len(p), nil
}

func (b *limitedGitBuffer) Bytes() []byte {
	if b.full {
		return append(b.bytes, []byte("\n[git output truncated]")...)
	}
	return b.bytes
}

func redactGitCommandOutput(output []byte, args []string) []byte {
	redacted := string(output)
	for _, arg := range args {
		parsed, err := url.Parse(arg)
		if err != nil || parsed.User == nil {
			continue
		}
		if password, ok := parsed.User.Password(); ok && password != "" {
			redacted = strings.ReplaceAll(redacted, password, "***")
			redacted = strings.ReplaceAll(redacted, url.PathEscape(password), "***")
		}
		withoutCredentials := *parsed
		withoutCredentials.User = nil
		redacted = strings.ReplaceAll(redacted, arg, withoutCredentials.String())
	}
	return []byte(redacted)
}
