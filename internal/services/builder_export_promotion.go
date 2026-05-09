package services

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
)

var downloadBuilderExportArchive = DownloadBuilderSessionExport
var runBuilderExportGitCommand = func(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

type builderExportPromotionMetadata struct {
	PromotedCodeRepositoryID string    `json:"promoted_code_repository_id,omitempty"`
	PromotedGitRepoURL       string    `json:"promoted_git_repo_url,omitempty"`
	PromotedAt               time.Time `json:"promoted_at,omitempty"`
}

func PromoteBuilderSessionExportToCodeRepository(ctx context.Context, projectID, sessionID, exportID string, req *models.PromoteBuilderExportToCodeRepositoryRequest) (*models.BuilderExportPromotionResponse, error) {
	if req == nil {
		return nil, app.NewErrorf("builder export promotion request is required")
	}

	session, err := loadBuilderSession(db.DB.WithContext(ctx), projectID, sessionID)
	if err != nil {
		return nil, err
	}

	var export entities.BuilderExport
	if err := db.DB.WithContext(ctx).
		Where("id = ? AND session_id = ?", exportID, session.ID).
		First(&export).Error; err != nil {
		return nil, err
	}

	workDir, err := os.MkdirTemp("", "ketches-builder-export-*")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = os.RemoveAll(workDir)
	}()

	if err := extractBuilderExportArchive(ctx, projectID, session.ID, export.ID, workDir); err != nil {
		return nil, err
	}

	if err := initializeBuilderPromotionRepository(workDir); err != nil {
		return nil, err
	}

	repoURL := req.GitRepoURL
	if req.GitUsername != "" && req.GitPassword != "" {
		repoURL = injectGitCredentials(repoURL, req.GitUsername, req.GitPassword)
	}
	if err := pushBuilderPromotionRepository(workDir, repoURL); err != nil {
		return nil, err
	}

	repository, err := CreateCodeRepository(projectID, &models.CreateCodeRepositoryRequest{
		Name:        req.Name,
		Slug:        req.Slug,
		GitRepoURL:  req.GitRepoURL,
		GitUsername: req.GitUsername,
		GitPassword: req.GitPassword,
	})
	if err != nil {
		return nil, err
	}

	metadataJSON, err := json.Marshal(builderExportPromotionMetadata{
		PromotedCodeRepositoryID: repository.ID,
		PromotedGitRepoURL:       req.GitRepoURL,
		PromotedAt:               time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}
	if err := db.DB.WithContext(ctx).
		Model(&entities.BuilderExport{}).
		Where("id = ?", export.ID).
		Update("metadata_json", string(metadataJSON)).Error; err != nil {
		return nil, err
	}
	export.MetadataJSON = string(metadataJSON)

	return &models.BuilderExportPromotionResponse{
		Export:     *toBuilderExportResponse(&export),
		Repository: ToCodeRepositoryResponse(&repository.CodeRepository),
	}, nil
}

func extractBuilderExportArchive(ctx context.Context, projectID, sessionID, exportID, destDir string) error {
	if strings.TrimSpace(destDir) == "" {
		return app.NewErrorf("builder export destination directory is required")
	}

	var archive bytes.Buffer
	if err := downloadBuilderExportArchive(ctx, projectID, sessionID, exportID, &archive); err != nil {
		return err
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(archive.Bytes()))
	if err != nil {
		return err
	}
	defer func() {
		_ = gzipReader.Close()
	}()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(destDir, filepath.Clean(header.Name))
		if !strings.HasPrefix(targetPath, destDir) {
			return app.NewErrorf("unsafe builder export archive path %q", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return err
			}
			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(file, tarReader); err != nil {
				_ = file.Close()
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}
		}
	}
}

func initializeBuilderPromotionRepository(dir string) error {
	commands := [][]string{
		{"init", "-b", "main"},
		{"config", "user.email", "builder@ketches.local"},
		{"config", "user.name", "Ketches Builder"},
		{"add", "."},
		{"commit", "--allow-empty", "-m", "chore: import builder export"},
	}
	for _, command := range commands {
		if output, err := runBuilderExportGitCommand(dir, command...); err != nil {
			return app.NewErrorf("git %s failed: %s", strings.Join(command, " "), strings.TrimSpace(string(output)))
		}
	}
	return nil
}

func pushBuilderPromotionRepository(dir, repoURL string) error {
	if output, err := runBuilderExportGitCommand(dir, "remote", "add", "origin", repoURL); err != nil {
		return app.NewErrorf("git remote add failed: %s", strings.TrimSpace(string(output)))
	}
	if output, err := runBuilderExportGitCommand(dir, "push", "-u", "origin", "main"); err != nil {
		return app.NewErrorf("git push failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}
