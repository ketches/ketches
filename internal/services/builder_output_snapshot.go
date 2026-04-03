package services

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

var streamBuilderOutputSnapshotSourceFile = func(ctx context.Context, workspace *entities.BuilderWorkspace, relativePath string, dst io.Writer) error {
	if workspace == nil {
		return errors.New("builder workspace is required")
	}

	appCtx, err := buildBuilderWorkspaceAppContext(workspace)
	if err != nil {
		return err
	}

	resolvedPath, err := resolveBuilderWorkspacePath(workspace.WorkspaceRoot, relativePath)
	if err != nil {
		return err
	}

	return execCommandStreamStdoutWithContext(ctx, appCtx, workspace.PodName, workspace.ContainerName, []string{"cat", resolvedPath}, dst)
}

func PublishBuilderOutputSnapshot(ctx context.Context, workspace *entities.BuilderWorkspace, run *entities.BuilderRun) (*entities.BuilderOutputSnapshot, error) {
	if workspace == nil {
		return nil, errors.New("builder workspace is required")
	}
	if run == nil {
		return nil, errors.New("builder run is required")
	}
	if workspace.SessionID != run.SessionID {
		return nil, errors.New("builder workspace session id must match builder run session id")
	}
	if run.WorkspaceID == nil || strings.TrimSpace(*run.WorkspaceID) == "" || *run.WorkspaceID != workspace.ID {
		return nil, errors.New("builder workspace id must match builder run workspace id")
	}
	if run.Status != entities.BuilderRunStatusSucceeded {
		return nil, app.NewErrorf("builder run must be succeeded to publish output snapshot: %s", run.Status)
	}

	existingSnapshot, err := getBuilderOutputSnapshotByRunID(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	if existingSnapshot != nil {
		return existingSnapshot, nil
	}

	artifacts, err := listBuilderOutputArtifacts(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	if len(artifacts) == 0 {
		return nil, errors.New("builder output artifacts are required")
	}

	publicationTime := time.Now().UTC()
	outputRoot, filePlans, totalSizeBytes, defaultEntryPath, err := planBuilderOutputSnapshotFiles(artifacts)
	if err != nil {
		return nil, err
	}

	status := entities.BuilderOutputSnapshotStatusDeliveryOnly
	if defaultEntryPath != "" {
		status = entities.BuilderOutputSnapshotStatusPreviewable
	}

	snapshot := &entities.BuilderOutputSnapshot{
		ID:               uuid.New(),
		SessionID:        workspace.SessionID,
		RunID:            run.ID,
		WorkspaceID:      workspace.ID,
		Status:           status,
		OutputRoot:       outputRoot,
		DefaultEntryPath: defaultEntryPath,
		FileCount:        len(filePlans),
		TotalSizeBytes:   totalSizeBytes,
		PublishedAt:      publicationTime,
	}

	storagePath, absPath, tmpPath, err := builderOutputSnapshotPaths(snapshot)
	if err != nil {
		return nil, err
	}
	snapshot.StoragePath = storagePath

	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return nil, app.WrapErrorf(err, "create snapshot directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(tmpPath), 0o755); err != nil {
		return nil, app.WrapErrorf(err, "create temporary snapshot directory: %w", err)
	}
	_ = os.RemoveAll(tmpPath)
	defer os.RemoveAll(tmpPath)

	if err := writeBuilderOutputSnapshotFiles(ctx, workspace, tmpPath, storagePath, snapshot.ID, filePlans); err != nil {
		return nil, err
	}
	if err := os.Rename(tmpPath, absPath); err != nil {
		return nil, app.WrapErrorf(err, "finalize snapshot directory: %w", err)
	}

	if err := db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(snapshot).Error; err != nil {
			return err
		}

		files := make([]entities.BuilderOutputSnapshotFile, 0, len(filePlans))
		for _, filePlan := range filePlans {
			files = append(files, entities.BuilderOutputSnapshotFile{
				ID:             uuid.New(),
				SnapshotID:     snapshot.ID,
				RelativePath:   filePlan.RelativePath,
				StoragePath:    filePlan.StoragePath,
				SizeBytes:      filePlan.SizeBytes,
				ContentType:    filePlan.ContentType,
				IsDefaultEntry: filePlan.IsDefaultEntry,
			})
		}

		return tx.Create(&files).Error
	}); err != nil {
		cleanupErr := os.RemoveAll(absPath)
		if cleanupErr != nil && !errors.Is(cleanupErr, os.ErrNotExist) {
			return nil, app.WrapErrorf(cleanupErr, "cleanup failed publish snapshot directory: %w", cleanupErr)
		}

		existingSnapshot, existingErr := getBuilderOutputSnapshotByRunID(ctx, run.ID)
		if existingErr != nil {
			return nil, errors.Join(err, existingErr)
		}
		if existingSnapshot != nil {
			return existingSnapshot, nil
		}

		return nil, err
	}

	return snapshot, nil
}

func GetLatestSuccessfulBuilderOutputSnapshot(ctx context.Context, sessionID string) (*entities.BuilderOutputSnapshot, error) {
	return getLatestSuccessfulBuilderOutputSnapshot(db.DB.WithContext(ctx), sessionID)
}

func GetBuilderOutputSnapshotByRunID(ctx context.Context, runID string) (*entities.BuilderOutputSnapshot, error) {
	return getBuilderOutputSnapshotByRunID(ctx, runID)
}

func getLatestSuccessfulBuilderOutputSnapshot(tx *gorm.DB, sessionID string) (*entities.BuilderOutputSnapshot, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("builder session id is required")
	}

	var snapshot entities.BuilderOutputSnapshot
	err := tx.
		Table("builder_output_snapshots").
		Joins("JOIN builder_runs AS br ON br.id = builder_output_snapshots.run_id").
		Where("builder_output_snapshots.session_id = ? AND br.status = ?", sessionID, entities.BuilderRunStatusSucceeded).
		Order("br.created_at DESC").
		Order("br.id DESC").
		Take(&snapshot).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &snapshot, nil
}

func OpenBuilderOutputSnapshotFile(snapshotFile *entities.BuilderOutputSnapshotFile) (io.ReadCloser, error) {
	if snapshotFile == nil {
		return nil, errors.New("builder output snapshot file is required")
	}
	if strings.TrimSpace(snapshotFile.StoragePath) == "" {
		return nil, os.ErrNotExist
	}

	return os.Open(builderOutputSnapshotAbsPath(snapshotFile.StoragePath))
}

func GetBuilderOutputSnapshotFile(ctx context.Context, snapshotID, relativePath string) (*entities.BuilderOutputSnapshotFile, error) {
	if strings.TrimSpace(snapshotID) == "" {
		return nil, errors.New("builder output snapshot id is required")
	}
	if strings.TrimSpace(relativePath) == "" {
		return nil, errors.New("builder output snapshot relative path is required")
	}

	validatedPath, err := validateBuilderAgentFilePath(relativePath)
	if err != nil {
		return nil, err
	}

	var snapshotFile entities.BuilderOutputSnapshotFile
	err = db.DB.WithContext(ctx).
		Where("snapshot_id = ? AND relative_path = ?", snapshotID, validatedPath).
		Take(&snapshotFile).Error
	if err != nil {
		return nil, err
	}

	return &snapshotFile, nil
}

func WriteBuilderOutputSnapshotArchive(ctx context.Context, snapshot *entities.BuilderOutputSnapshot, writer io.Writer) error {
	if snapshot == nil {
		return errors.New("builder output snapshot is required")
	}
	if writer == nil {
		return errors.New("archive writer is required")
	}

	files, err := listBuilderOutputSnapshotFiles(ctx, snapshot.ID)
	if err != nil {
		return err
	}

	gzipWriter := gzip.NewWriter(writer)
	tarWriter := tar.NewWriter(gzipWriter)

	for i := range files {
		snapshotFile := &files[i]
		reader, err := OpenBuilderOutputSnapshotFile(snapshotFile)
		if err != nil {
			_ = tarWriter.Close()
			_ = gzipWriter.Close()
			return err
		}

		fileInfo, err := os.Stat(builderOutputSnapshotAbsPath(snapshotFile.StoragePath))
		if err != nil {
			reader.Close()
			_ = tarWriter.Close()
			_ = gzipWriter.Close()
			return err
		}

		header, err := tar.FileInfoHeader(fileInfo, "")
		if err != nil {
			reader.Close()
			_ = tarWriter.Close()
			_ = gzipWriter.Close()
			return err
		}
		header.Name = snapshotFile.RelativePath
		if err := tarWriter.WriteHeader(header); err != nil {
			reader.Close()
			_ = tarWriter.Close()
			_ = gzipWriter.Close()
			return err
		}
		if _, err := io.Copy(tarWriter, reader); err != nil {
			reader.Close()
			_ = tarWriter.Close()
			_ = gzipWriter.Close()
			return err
		}
		if err := reader.Close(); err != nil {
			_ = tarWriter.Close()
			_ = gzipWriter.Close()
			return err
		}
	}

	if err := tarWriter.Close(); err != nil {
		_ = gzipWriter.Close()
		return err
	}
	return gzipWriter.Close()
}

func DeleteBuilderOutputSnapshotsByRunID(ctx context.Context, runID string) error {
	if strings.TrimSpace(runID) == "" {
		return errors.New("builder run id is required")
	}

	return db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return deleteBuilderOutputSnapshotsByRunIDTx(tx, runID)
	})
}

type builderOutputSnapshotFilePlan struct {
	RelativePath   string
	StoragePath    string
	SizeBytes      int64
	ContentType    string
	IsDefaultEntry bool
}

func getBuilderOutputSnapshotByRunID(ctx context.Context, runID string) (*entities.BuilderOutputSnapshot, error) {
	var snapshot entities.BuilderOutputSnapshot
	err := db.DB.WithContext(ctx).Where("run_id = ?", runID).Take(&snapshot).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &snapshot, nil
}

func listBuilderOutputArtifacts(ctx context.Context, runID string) ([]entities.BuilderArtifact, error) {
	var artifacts []entities.BuilderArtifact
	err := db.DB.WithContext(ctx).
		Where("run_id = ? AND kind = ?", runID, entities.BuilderArtifactKindBuildOutput).
		Order("path ASC, id ASC").
		Find(&artifacts).Error
	return artifacts, err
}

func deleteBuilderOutputSnapshotsByRunIDTx(tx *gorm.DB, runID string) error {
	var snapshotIDs []string
	if err := tx.Model(&entities.BuilderOutputSnapshot{}).
		Where("run_id = ?", runID).
		Pluck("id", &snapshotIDs).Error; err != nil {
		return err
	}
	if len(snapshotIDs) == 0 {
		return nil
	}
	if err := tx.Where("snapshot_id IN ?", snapshotIDs).Delete(&entities.BuilderOutputSnapshotFile{}).Error; err != nil {
		return err
	}
	if err := tx.Where("id IN ?", snapshotIDs).Delete(&entities.BuilderOutputSnapshot{}).Error; err != nil {
		return err
	}
	return nil
}

func listBuilderOutputSnapshotFiles(ctx context.Context, snapshotID string) ([]entities.BuilderOutputSnapshotFile, error) {
	var files []entities.BuilderOutputSnapshotFile
	err := db.DB.WithContext(ctx).
		Where("snapshot_id = ?", snapshotID).
		Order("relative_path ASC").
		Find(&files).Error
	return files, err
}

func planBuilderOutputSnapshotFiles(artifacts []entities.BuilderArtifact) (string, []builderOutputSnapshotFilePlan, int64, string, error) {
	if len(artifacts) == 0 {
		return "", nil, 0, "", errors.New("builder output artifacts are required")
	}

	outputRoot := ""
	defaultEntryPath := ""
	totalSizeBytes := int64(0)
	filePlans := make([]builderOutputSnapshotFilePlan, 0, len(artifacts))

	for _, artifact := range artifacts {
		validatedRelativePath, err := validateBuilderAgentFilePath(artifact.Path)
		if err != nil {
			return "", nil, 0, "", err
		}

		metadata, err := parseBuilderArtifactMetadata(artifact.MetadataJSON)
		if err != nil {
			return "", nil, 0, "", err
		}

		artifactOutputRoot := strings.TrimSpace(metadata.OutputRoot)
		if artifactOutputRoot == "" {
			artifactOutputRoot = strings.SplitN(validatedRelativePath, "/", 2)[0]
		}
		artifactOutputRoot, err = validateBuilderAgentFilePath(artifactOutputRoot)
		if err != nil {
			return "", nil, 0, "", err
		}

		if outputRoot == "" {
			outputRoot = artifactOutputRoot
		} else if outputRoot != artifactOutputRoot {
			return "", nil, 0, "", app.NewErrorf("builder output artifacts span multiple roots: %s and %s", outputRoot, artifactOutputRoot)
		}

		contentType := mime.TypeByExtension(path.Ext(validatedRelativePath))
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		isDefaultEntry := validatedRelativePath == path.Join(outputRoot, "index.html")
		if isDefaultEntry {
			defaultEntryPath = validatedRelativePath
		}

		filePlans = append(filePlans, builderOutputSnapshotFilePlan{
			RelativePath:   validatedRelativePath,
			SizeBytes:      metadata.SizeBytes,
			ContentType:    contentType,
			IsDefaultEntry: isDefaultEntry,
		})
		totalSizeBytes += metadata.SizeBytes
	}

	return outputRoot, filePlans, totalSizeBytes, defaultEntryPath, nil
}

func parseBuilderArtifactMetadata(metadataJSON string) (builderArtifactMetadata, error) {
	if strings.TrimSpace(metadataJSON) == "" {
		return builderArtifactMetadata{}, nil
	}

	var metadata builderArtifactMetadata
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		return builderArtifactMetadata{}, err
	}

	return metadata, nil
}

func writeBuilderOutputSnapshotFiles(ctx context.Context, workspace *entities.BuilderWorkspace, tmpRoot, storagePath, snapshotID string, filePlans []builderOutputSnapshotFilePlan) error {
	for i := range filePlans {
		plan := &filePlans[i]

		fileDir := filepath.Join(tmpRoot, filepath.FromSlash(path.Dir(plan.RelativePath)))
		if err := os.MkdirAll(fileDir, 0o755); err != nil {
			return app.WrapErrorf(err, "create snapshot file directory: %w", err)
		}

		finalTmpPath := filepath.Join(tmpRoot, filepath.FromSlash(plan.RelativePath))
		tempFilePath := finalTmpPath + ".tmp"
		_ = os.Remove(tempFilePath)

		tempFile, err := os.OpenFile(tempFilePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return app.WrapErrorf(err, "create snapshot file: %w", err)
		}

		streamErr := streamBuilderOutputSnapshotSourceFile(ctx, workspace, plan.RelativePath, tempFile)
		closeErr := tempFile.Close()
		if streamErr != nil {
			_ = os.Remove(tempFilePath)
			return app.WrapErrorf(streamErr, "copy snapshot source file %s: %w", plan.RelativePath, streamErr)
		}
		if closeErr != nil {
			_ = os.Remove(tempFilePath)
			return app.WrapErrorf(closeErr, "close snapshot file %s: %w", plan.RelativePath, closeErr)
		}
		if err := os.Rename(tempFilePath, finalTmpPath); err != nil {
			_ = os.Remove(tempFilePath)
			return app.WrapErrorf(err, "finalize snapshot file %s: %w", plan.RelativePath, err)
		}

		plan.StoragePath = path.Join(storagePath, plan.RelativePath)
	}

	return nil
}

func builderOutputSnapshotPaths(snapshot *entities.BuilderOutputSnapshot) (string, string, string, error) {
	if snapshot == nil {
		return "", "", "", errors.New("builder output snapshot is required")
	}

	baseDir := builderOutputSnapshotBaseDir()
	if strings.TrimSpace(baseDir) == "" {
		return "", "", "", errors.New("builder snapshot base dir is not configured")
	}

	relPath := filepath.ToSlash(filepath.Join("sessions", snapshot.SessionID, "runs", snapshot.RunID, snapshot.ID))
	absPath := builderOutputSnapshotAbsPath(relPath)
	tmpPath := filepath.Join(baseDir, "tmp", snapshot.ID)
	return relPath, absPath, tmpPath, nil
}

func builderOutputSnapshotAbsPath(relPath string) string {
	return filepath.Join(builderOutputSnapshotBaseDir(), filepath.FromSlash(relPath))
}

func builderOutputSnapshotBaseDir() string {
	if strings.TrimSpace(app.Config.BuilderSnapshotBaseDir) == "" {
		return "data/builder-previews"
	}
	return app.Config.BuilderSnapshotBaseDir
}
