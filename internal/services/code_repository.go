package services

import (
	"fmt"
	"net/url"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

// RepoNameFromURL derives a short name from git repo URL (e.g. last path segment without .git).
func RepoNameFromURL(gitRepoURL string) string {
	u, err := url.Parse(gitRepoURL)
	if err != nil {
		// fallback: take after last /
		if idx := strings.LastIndex(gitRepoURL, "/"); idx >= 0 && idx < len(gitRepoURL)-1 {
			s := gitRepoURL[idx+1:]
			s = strings.TrimSuffix(s, ".git")
			return sanitizeRepoName(s)
		}
		return "repo"
	}
	seg := path.Base(u.Path)
	seg = strings.TrimSuffix(seg, ".git")
	return sanitizeRepoName(seg)
}

func sanitizeRepoName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "repo"
	}
	if len(s) > 128 {
		s = s[:128]
	}
	return s
}

// RepoSlugFromName returns a URL-safe slug from a name (lowercase, non-alphanumeric to hyphen).
func RepoSlugFromName(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return "repo"
	}
	s = strings.ToLower(s)
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "repo"
	}
	if len(s) > 128 {
		s = s[:128]
	}
	return s
}

func ListCodeRepositories(projectID string, page, pageSize int, search string) (int64, []entities.CodeRepository, error) {
	var repos []entities.CodeRepository
	var total int64
	query := db.DB.Model(&entities.CodeRepository{}).Where("project_id = ?", projectID)
	if search != "" {
		query = query.Where("name LIKE ? OR slug LIKE ? OR git_repo_url LIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if err := query.Order("created_at asc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&repos).Error; err != nil {
		return 0, nil, err
	}
	return total, repos, nil
}

func ListCodeRepositoriesSimple(projectID string) ([]entities.CodeRepository, error) {
	var repos []entities.CodeRepository
	if err := db.DB.Select("id, slug, name, description").Where("project_id = ?", projectID).Order("name").Find(&repos).Error; err != nil {
		return nil, err
	}
	return repos, nil
}

func GetCodeRepository(id string) (*entities.CodeRepository, error) {
	var repo entities.CodeRepository
	if err := db.DB.Preload("Project").
		First(&repo, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &repo, nil
}

func CreateCodeRepository(projectID string, req *models.CreateCodeRepositoryRequest) (*entities.CodeRepository, error) {
	secret, _ := generateWebhookSecret()
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = RepoNameFromURL(req.GitRepoURL)
	}
	slug := strings.TrimSpace(req.Slug)
	if slug == "" {
		slug = RepoSlugFromName(name)
	} else {
		slug = RepoSlugFromName(slug)
	}

	var existing entities.CodeRepository
	if err := db.DB.Where("project_id = ? AND slug = ?", projectID, slug).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("code repository with slug %s already exists in the project", slug)
	}

	repo := &entities.CodeRepository{
		Base:           entities.Base{ID: uuid.New()},
		ProjectID:      projectID,
		Name:           name,
		Slug:           slug,
		GitRepoURL:     req.GitRepoURL,
		GitUsername:    req.GitUsername,
		GitPassword:    req.GitPassword,
		WebhookSecret:  secret,
		WebhookEnabled: false,
	}
	if err := db.DB.Create(repo).Error; err != nil {
		return nil, err
	}
	return GetCodeRepository(repo.ID)
}

func UpdateCodeRepository(id string, req *models.UpdateCodeRepositoryRequest) (*entities.CodeRepository, error) {
	repo, err := GetCodeRepository(id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		repo.Name = sanitizeRepoName(req.Name)
	}
	if req.GitRepoURL != "" {
		repo.GitRepoURL = req.GitRepoURL
	}
	repo.GitUsername = req.GitUsername
	repo.GitPassword = req.GitPassword
	if req.WebhookEnabled != nil {
		repo.WebhookEnabled = *req.WebhookEnabled
	}
	if err := db.DB.Save(repo).Error; err != nil {
		return nil, err
	}
	return GetCodeRepository(id)
}

func DeleteCodeRepository(id string) error {
	var configCount int64
	if err := db.DB.Model(&entities.CodeRepositoryBuildConfig{}).Where("code_repository_id = ?", id).Count(&configCount).Error; err != nil {
		return err
	}
	if configCount > 0 {
		return fmt.Errorf("cannot delete code repository: remove %d build config(s) first", configCount)
	}
	var buildCount int64
	if err := db.DB.Model(&entities.Build{}).Where("code_repository_id = ?", id).Count(&buildCount).Error; err != nil {
		return err
	}
	if buildCount > 0 {
		return fmt.Errorf("cannot delete code repository: it has %d build(s)", buildCount)
	}
	return db.DB.Delete(&entities.CodeRepository{}, "id = ?", id).Error
}

func ToCodeRepositoryResponse(r *entities.CodeRepository, baseURL string) models.CodeRepositoryResponse {
	resp := models.CodeRepositoryResponse{
		ID:             r.ID,
		ProjectID:      r.ProjectID,
		Name:           r.Name,
		Slug:           r.Slug,
		GitRepoURL:     r.GitRepoURL,
		GitUsername:    r.GitUsername,
		GitPassword:    r.GitPassword,
		WebhookSecret:  r.WebhookSecret,
		WebhookEnabled: r.WebhookEnabled,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
	if baseURL != "" && r.WebhookSecret != "" {
		resp.WebhookURL = fmt.Sprintf("%s/api/v1/webhooks/git/repo/%s?secret=%s", baseURL, r.ID, r.WebhookSecret)
	}
	return resp
}

// --- Build configs ---

func ListCodeRepositoryBuildConfigs(repoID string) ([]entities.CodeRepositoryBuildConfig, error) {
	var configs []entities.CodeRepositoryBuildConfig
	if err := db.DB.Preload("Registry").
		Where("code_repository_id = ?", repoID).
		Order("created_at asc").
		Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

func GetCodeRepositoryBuildConfig(configID string) (*entities.CodeRepositoryBuildConfig, error) {
	var cfg entities.CodeRepositoryBuildConfig
	if err := db.DB.Preload("Registry").Preload("CodeRepository").
		First(&cfg, "id = ?", configID).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func CreateCodeRepositoryBuildConfig(repoID string, req *models.CreateCodeRepositoryBuildConfigRequest) (*entities.CodeRepositoryBuildConfig, error) {
	repo, err := GetCodeRepository(repoID)
	if err != nil {
		return nil, err
	}
	cfg := &entities.CodeRepositoryBuildConfig{
		Base:             entities.Base{ID: uuid.New()},
		CodeRepositoryID: repoID,
		Name:             req.Name,
		GitRef:           defaultStr(req.GitRef, "main"),
		DockerfilePath:   defaultStr(req.DockerfilePath, "Dockerfile"),
		BuildContext:     defaultStr(req.BuildContext, "."),
		ImageName:        req.ImageName,
		RegistryID:       req.RegistryID,
		BuildArgs:        req.BuildArgs,
		AutoBuild:        req.AutoBuild,
		AutoDeploy:       req.AutoDeploy,
		WebhookEnabled:   req.WebhookEnabled,
	}
	if err := db.DB.Create(cfg).Error; err != nil {
		return nil, err
	}
	_ = repo
	return GetCodeRepositoryBuildConfig(cfg.ID)
}

func UpdateCodeRepositoryBuildConfig(configID string, req *models.UpdateCodeRepositoryBuildConfigRequest) (*entities.CodeRepositoryBuildConfig, error) {
	cfg, err := GetCodeRepositoryBuildConfig(configID)
	if err != nil {
		return nil, err
	}
	if req.Name != "" {
		cfg.Name = req.Name
	}
	if req.GitRef != "" {
		cfg.GitRef = req.GitRef
	}
	if req.DockerfilePath != "" {
		cfg.DockerfilePath = req.DockerfilePath
	}
	if req.BuildContext != "" {
		cfg.BuildContext = req.BuildContext
	}
	if req.ImageName != "" {
		cfg.ImageName = req.ImageName
	}
	if req.RegistryID != "" {
		cfg.RegistryID = req.RegistryID
	}
	if req.BuildArgs != "" {
		cfg.BuildArgs = req.BuildArgs
	}
	if req.AutoBuild != nil {
		cfg.AutoBuild = *req.AutoBuild
	}
	if req.AutoDeploy != nil {
		cfg.AutoDeploy = *req.AutoDeploy
	}
	if req.WebhookEnabled != nil {
		cfg.WebhookEnabled = *req.WebhookEnabled
	}
	if err := db.DB.Save(cfg).Error; err != nil {
		return nil, err
	}
	return GetCodeRepositoryBuildConfig(configID)
}

func DeleteCodeRepositoryBuildConfig(configID string) error {
	var count int64
	if err := db.DB.Model(&entities.Build{}).Where("code_repository_build_config_id = ?", configID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("cannot delete build config: it has %d build(s)", count)
	}
	return db.DB.Delete(&entities.CodeRepositoryBuildConfig{}, "id = ?", configID).Error
}

func ToCodeRepositoryBuildConfigResponse(c *entities.CodeRepositoryBuildConfig) models.CodeRepositoryBuildConfigResponse {
	resp := models.CodeRepositoryBuildConfigResponse{
		ID:               c.ID,
		CodeRepositoryID: c.CodeRepositoryID,
		Name:             c.Name,
		GitRef:           c.GitRef,
		DockerfilePath:   c.DockerfilePath,
		BuildContext:     c.BuildContext,
		ImageName:        c.ImageName,
		RegistryID:       c.RegistryID,
		BuildArgs:        c.BuildArgs,
		AutoBuild:        c.AutoBuild,
		AutoDeploy:       c.AutoDeploy,
		WebhookEnabled:   c.WebhookEnabled,
		CreatedAt:        c.CreatedAt,
		UpdatedAt:        c.UpdatedAt,
	}
	if c.Registry.ID != "" {
		regResp := ToContainerRegistryResponse(&c.Registry)
		resp.Registry = &regResp
	}
	return resp
}

func ListAvailableContainerRegistriesForCodeRepository(repoID string) ([]entities.ContainerRegistry, error) {
	repo, err := GetCodeRepository(repoID)
	if err != nil {
		return nil, err
	}
	var env entities.Env
	if err := db.DB.Where("project_id = ?", repo.ProjectID).First(&env).Error; err != nil {
		return nil, err
	}
	return ListAvailableRegistries(env.ClusterID, repo.ProjectID)
}

func CodeRepositorySlugForJob(repoName, configName string) string {
	slug := repoName + "-" + configName
	s := regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(strings.ToLower(slug), "-")
	s = regexp.MustCompile(`-+`).ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 32 {
		s = s[:32]
	}
	if s == "" {
		s = "cr"
	}
	return "cr-" + s
}

func GetCodeRepositoryTopology(repoID string) (*models.AppTopologyResponse, error) {
	repo, err := GetCodeRepository(repoID)
	if err != nil {
		return nil, err
	}

	var nodes []models.AppTopologyNode
	var edges []models.AppTopologyEdge

	repoNodeID := "repo-" + repo.ID
	nodes = append(nodes, models.AppTopologyNode{
		ID:   repoNodeID,
		Type: "CodeRepository",
		Name: repo.Name,
	})

	// Get associated apps and their envs
	var apps []entities.App
	db.DB.Preload("Env").Where("code_repository_id = ?", repoID).Find(&apps)

	// Add env and app nodes
	for _, app := range apps {
		envNodeID := "env-" + app.Env.ID
		appNodeID := "app-" + app.ID

		// Add env node if not exists
		envExists := false
		for _, n := range nodes {
			if n.ID == envNodeID {
				envExists = true
				break
			}
		}
		if !envExists {
			nodes = append(nodes, models.AppTopologyNode{
				ID:   envNodeID,
				Type: "Environment",
				Name: app.Env.Name,
			})
		}

		// Add app node
		nodes = append(nodes, models.AppTopologyNode{
			ID:   appNodeID,
			Type: "Application",
			Name: app.Name,
		})
		// Connect Env -> App
		edges = append(edges, models.AppTopologyEdge{
			Source: envNodeID,
			Target: appNodeID,
		})
	}

	// Get build configs
	var buildConfigs []entities.CodeRepositoryBuildConfig
	if err := db.DB.Where("code_repository_id = ?", repoID).Order("created_at asc").Find(&buildConfigs).Error; err == nil {
		for _, bc := range buildConfigs {
			bcNodeID := "bc-" + bc.ID
			nodes = append(nodes, models.AppTopologyNode{
				ID:   bcNodeID,
				Type: "BuildConfig",
				Name: bc.Name,
			})
			edges = append(edges, models.AppTopologyEdge{
				Source: repoNodeID,
				Target: bcNodeID,
			})

			// Relationship to Envs
			// In Ketches, a BuildConfig produces an image that is usually intended for specific apps.
			// Since apps are linked to the repo, we can connect build configs to envs where these apps exist.
			for _, app := range apps {
				envNodeID := "env-" + app.Env.ID
				edgeExists := false
				for _, e := range edges {
					if e.Source == bcNodeID && e.Target == envNodeID {
						edgeExists = true
						break
					}
				}
				if !edgeExists {
					edges = append(edges, models.AppTopologyEdge{
						Source: bcNodeID,
						Target: envNodeID,
					})
				}
			}
		}
	}

	return &models.AppTopologyResponse{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

func ListCodeRepositoryRefs(repoID string) ([]models.GitRef, error) {
	repo, err := GetCodeRepository(repoID)
	if err != nil {
		return nil, err
	}

	repoURL := repo.GitRepoURL
	if repo.GitUsername != "" && repo.GitPassword != "" {
		repoURL = injectGitCredentials(repoURL, repo.GitUsername, repo.GitPassword)
	}

	cmd := exec.Command("git", "ls-remote", "--heads", "--tags", repoURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list refs: %s", strings.TrimSpace(string(output)))
	}

	var refs []models.GitRef
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		ref := parts[1]
		if strings.HasPrefix(ref, "refs/heads/") {
			refs = append(refs, models.GitRef{
				Name: strings.TrimPrefix(ref, "refs/heads/"),
				Type: "branch",
			})
		} else if strings.HasPrefix(ref, "refs/tags/") {
			if strings.HasSuffix(ref, "^{}") {
				continue
			}
			refs = append(refs, models.GitRef{
				Name: strings.TrimPrefix(ref, "refs/tags/"),
				Type: "tag",
			})
		}
	}

	return refs, nil
}
