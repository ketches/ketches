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

type CodeRepositoryWithProject struct {
	entities.CodeRepository
	Project entities.Project `gorm:"embedded;embeddedPrefix:project_"`
}

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
	if err := query.Order("created_at").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&repos).Error; err != nil {
		return 0, nil, err
	}
	return total, repos, nil
}

func ListCodeRepositoriesSimple(projectID string) ([]models.SimpleCodeRepository, error) {
	var repos []models.SimpleCodeRepository
	if err := db.DB.Model(&entities.CodeRepository{}).Select("id, slug, name").Where("project_id = ?", projectID).Order("created_at").Find(&repos).Error; err != nil {
		return nil, err
	}
	return repos, nil
}

func GetCodeRepository(id string) (*CodeRepositoryWithProject, error) {
	var repo CodeRepositoryWithProject
	if err := db.DB.Table("code_repositories").
		Select("code_repositories.*, projects.id AS project_id, projects.created_at AS project_created_at, projects.updated_at AS project_updated_at, projects.slug AS project_slug, projects.name AS project_name, projects.description AS project_description").
		Joins("LEFT JOIN projects ON projects.id = code_repositories.project_id").
		Where("code_repositories.id = ?", id).
		First(&repo).Error; err != nil {
		return nil, err
	}
	return &repo, nil
}

func CreateCodeRepository(projectID string, req *models.CreateCodeRepositoryRequest) (*CodeRepositoryWithProject, error) {
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
		Base:        entities.Base{ID: uuid.New()},
		ProjectID:   projectID,
		Name:        name,
		Slug:        slug,
		GitRepoURL:  req.GitRepoURL,
		GitUsername: req.GitUsername,
		GitPassword: req.GitPassword,
	}
	if err := db.DB.Create(repo).Error; err != nil {
		return nil, err
	}
	return GetCodeRepository(repo.ID)
}

func UpdateCodeRepository(id string, req *models.UpdateCodeRepositoryRequest) (*CodeRepositoryWithProject, error) {
	repo, err := GetCodeRepository(id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		repo.Name = sanitizeRepoName(req.Name)
	}
	if req.Slug != "" {
		repo.Slug = RepoSlugFromName(req.Slug)
	}
	if req.GitRepoURL != "" {
		repo.GitRepoURL = req.GitRepoURL
	}
	repo.GitUsername = req.GitUsername
	repo.GitPassword = req.GitPassword
	if err := db.DB.Save(&repo.CodeRepository).Error; err != nil {
		return nil, err
	}
	return GetCodeRepository(id)
}

func DeleteCodeRepository(id string) error {
	var buildSettingCount int64
	if err := db.DB.Model(&entities.BuildSetting{}).Where("code_repository_id = ?", id).Count(&buildSettingCount).Error; err != nil {
		return err
	}
	if buildSettingCount > 0 {
		return fmt.Errorf("cannot delete code repository: remove %d build setting(s) first", buildSettingCount)
	}
	var buildCount int64
	if err := db.DB.Model(&entities.Build{}).
		Joins("JOIN build_settings ON build_settings.id = builds.build_setting_id").
		Where("build_settings.code_repository_id = ?", id).
		Count(&buildCount).Error; err != nil {
		return err
	}
	if buildCount > 0 {
		return fmt.Errorf("cannot delete code repository: it has %d build(s)", buildCount)
	}
	return db.DB.Delete(&entities.CodeRepository{}, "id = ?", id).Error
}

func ToCodeRepositoryResponse(r *entities.CodeRepository) models.CodeRepositoryResponse {
	resp := models.CodeRepositoryResponse{
		ID:          r.ID,
		ProjectID:   r.ProjectID,
		Name:        r.Name,
		Slug:        r.Slug,
		GitRepoURL:  r.GitRepoURL,
		GitUsername: r.GitUsername,
		GitPassword: r.GitPassword,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
	return resp
}

func ToCodeRepositoryRowResponse(r *CodeRepositoryWithProject) models.CodeRepositoryResponse {
	return ToCodeRepositoryResponse(&r.CodeRepository)
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

func CodeRepositorySlugForJob(repoName, buildSettingName string) string {
	slug := repoName + "-" + buildSettingName
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
	db.DB.Where("code_repository_id = ?", repoID).Find(&apps)
	// Batch-fetch Env for each app
	envIDs := make(map[string]struct{})
	for _, a := range apps {
		envIDs[a.EnvID] = struct{}{}
	}
	envMap := make(map[string]entities.Env)
	if len(envIDs) > 0 {
		envIDList := make([]string, 0, len(envIDs))
		for id := range envIDs {
			envIDList = append(envIDList, id)
		}
		var envs []entities.Env
		db.DB.Where("id IN ?", envIDList).Find(&envs)
		for _, e := range envs {
			envMap[e.ID] = e
		}
	}

	// Add env and app nodes
	for _, app := range apps {
		env := envMap[app.EnvID]
		envNodeID := "env-" + env.ID
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
				Name: env.Name,
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

	var buildSettings []entities.BuildSetting
	if err := db.DB.Where("code_repository_id = ?", repoID).Order("created_at").Find(&buildSettings).Error; err == nil {
		for _, bs := range buildSettings {
			bsNodeID := "bs-" + bs.ID
			nodes = append(nodes, models.AppTopologyNode{
				ID:   bsNodeID,
				Type: "BuildSetting",
				Name: bs.Name,
			})
			edges = append(edges, models.AppTopologyEdge{
				Source: repoNodeID,
				Target: bsNodeID,
			})

			for _, app := range apps {
				env := envMap[app.EnvID]
				envNodeID := "env-" + env.ID
				edgeExists := false
				for _, e := range edges {
					if e.Source == bsNodeID && e.Target == envNodeID {
						edgeExists = true
						break
					}
				}
				if !edgeExists {
					edges = append(edges, models.AppTopologyEdge{
						Source: bsNodeID,
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
