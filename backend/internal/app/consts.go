package app

const (
	UserRoleAdmin = "admin"
	UserRoleUser  = "user"
)

var UserRoles = []string{UserRoleAdmin, UserRoleUser}

const (
	ProjectRoleOwner     = "owner"
	ProjectRoleDeveloper = "developer"
	ProjectRoleViewer    = "viewer"
)

var ProjectRoles = []string{ProjectRoleOwner, ProjectRoleDeveloper, ProjectRoleViewer}
