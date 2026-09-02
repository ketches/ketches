package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupProjectMembershipRaceTestDB(t *testing.T) {
	t.Helper()
	previous := db.DB
	t.Cleanup(func() { db.DB = previous })

	testDB, err := gorm.Open(sqlite.Open(t.TempDir()+"/membership.db"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(
		&entities.Project{},
		&entities.ProjectMember{},
		&entities.Notification{},
	))
	db.DB = testDB
}

func TestProjectMemberCompositeUniqueIndex(t *testing.T) {
	setupProjectMembershipRaceTestDB(t)
	project := &entities.Project{Base: entities.Base{ID: "project-1"}, Slug: "project-1", Name: "Project"}
	require.NoError(t, db.DB.Create(project).Error)
	require.NoError(t, db.DB.Create(&entities.ProjectMember{
		ID: "member-1", ProjectID: project.ID, UserID: "user-1", ProjectRole: app.ProjectRoleViewer,
	}).Error)

	err := db.DB.Create(&entities.ProjectMember{
		ID: "member-2", ProjectID: project.ID, UserID: "user-1", ProjectRole: app.ProjectRoleDeveloper,
	}).Error
	assert.Error(t, err)
}

func TestInviteProjectMembersIsIdempotent(t *testing.T) {
	setupProjectMembershipRaceTestDB(t)
	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"}, Slug: "project-1", Name: "Project",
	}).Error)

	require.NoError(t, InviteProjectMembers("project-1", []string{"user-1"}, app.ProjectRoleDeveloper, "sender-1"))
	require.NoError(t, InviteProjectMembers("project-1", []string{"user-1"}, app.ProjectRoleViewer, "sender-1"))

	var invitations []entities.Notification
	require.NoError(t, db.DB.Where("recipient_id = ? AND resource_id = ? AND status = ?", "user-1", "project-1", "pending").Find(&invitations).Error)
	require.Len(t, invitations, 1)
	assert.JSONEq(t, `{"role":"viewer"}`, invitations[0].ActionData)
}

func TestHandleNotificationActionAcceptIsIdempotent(t *testing.T) {
	setupProjectMembershipRaceTestDB(t)
	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"}, Slug: "project-1", Name: "Project",
	}).Error)
	require.NoError(t, InviteProjectMembers("project-1", []string{"user-1"}, app.ProjectRoleDeveloper, "sender-1"))

	var invitation entities.Notification
	require.NoError(t, db.DB.Where("recipient_id = ?", "user-1").First(&invitation).Error)
	require.NoError(t, HandleNotificationAction(invitation.ID, "user-1", "accept"))
	// A retry after a lost response must not create a second member or fail.
	require.NoError(t, HandleNotificationAction(invitation.ID, "user-1", "accept"))

	var memberCount int64
	require.NoError(t, db.DB.Model(&entities.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", "project-1", "user-1").Count(&memberCount).Error)
	assert.Equal(t, int64(1), memberCount)
	var accepted entities.Notification
	require.NoError(t, db.DB.First(&accepted, "id = ?", invitation.ID).Error)
	assert.Equal(t, "accepted", accepted.Status)
}

func TestOwnerMutationsRejectRemovingLastOwner(t *testing.T) {
	setupProjectMembershipRaceTestDB(t)
	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"}, Slug: "project-1", Name: "Project",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.ProjectMember{
		ID: "member-owner", ProjectID: "project-1", UserID: "owner-1", ProjectRole: app.ProjectRoleOwner,
	}).Error)

	assert.ErrorContains(t, UpdateProjectMemberRole("project-1", "owner-1", app.ProjectRoleViewer), "at least one owner")
	assert.ErrorContains(t, RemoveProjectMember("project-1", "owner-1"), "at least one owner")
}

func TestProjectMemberMutationsRejectInvalidRole(t *testing.T) {
	setupProjectMembershipRaceTestDB(t)
	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"}, Slug: "project-1", Name: "Project",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.ProjectMember{
		ID: "member-1", ProjectID: "project-1", UserID: "user-1", ProjectRole: app.ProjectRoleDeveloper,
	}).Error)

	assert.ErrorIs(t, InviteProjectMembers("project-1", []string{"user-2"}, "administrator", "owner-1"), ErrInvalidProjectRole)
	assert.ErrorIs(t, UpdateProjectMemberRole("project-1", "user-1", "administrator"), ErrInvalidProjectRole)

	var member entities.ProjectMember
	require.NoError(t, db.DB.First(&member, "project_id = ? AND user_id = ?", "project-1", "user-1").Error)
	assert.Equal(t, app.ProjectRoleDeveloper, member.ProjectRole)

	var notificationCount int64
	require.NoError(t, db.DB.Model(&entities.Notification{}).Count(&notificationCount).Error)
	assert.Zero(t, notificationCount)
}

func TestInvitationAcceptanceRejectsInvalidRole(t *testing.T) {
	setupProjectMembershipRaceTestDB(t)
	require.NoError(t, db.DB.Create(&entities.Notification{
		Base:        entities.Base{ID: "notification-1"},
		RecipientID: "user-1",
		Category:    "invitation",
		EventType:   "project_invitation",
		ResourceID:  "project-1",
		Status:      "pending",
		ActionData:  `{"role":"administrator"}`,
	}).Error)

	assert.ErrorIs(t, HandleNotificationAction("notification-1", "user-1", "accept"), ErrInvalidProjectRole)

	var memberCount int64
	require.NoError(t, db.DB.Model(&entities.ProjectMember{}).Count(&memberCount).Error)
	assert.Zero(t, memberCount)
}
