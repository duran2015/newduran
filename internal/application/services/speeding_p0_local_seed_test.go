package services

import (
	"testing"

	"control-panel/internal/domain/agent"
	"control-panel/internal/domain/gamepackage"
	"control-panel/internal/domain/gameproject"
	"github.com/stretchr/testify/require"
)

func TestSeedSpeedingP0LocalIsIdempotentAndLeavesAssetsPending(t *testing.T) {
	db := setupToolTenantServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&gamepackage.Package{}, &gameproject.Project{}, &gameproject.Release{}))
	for _, name := range []string{"director", "cfo", "counsel"} {
		require.NoError(t, db.Create(&agent.AgentConfig{TenantID: "default", Name: name, SystemPrompt: name, ContentHash: name}).Error)
	}
	projects := NewGameProjectService(db, NewGamePackageService(db))
	agents := NewAgentService("", "")

	first, err := SeedSpeedingP0Local(db, projects, agents, SpeedingP0SeedOptions{TenantID: "default"})
	require.NoError(t, err)
	require.True(t, first.ProjectCreated)
	require.Equal(t, []string{"board-rival"}, first.CreatedAgents)
	require.NotEmpty(t, first.PendingAssets)
	require.NotZero(t, first.AgentIDs["board_rival"])

	detail, err := projects.Get("default", first.ProjectID)
	require.NoError(t, err)
	require.Len(t, detail.Draft.NPCs, 4)
	require.NotEmpty(t, detail.Draft.Events)
	require.NotEmpty(t, detail.Draft.Actions)
	require.NotEmpty(t, detail.Draft.Endings)
	for _, scene := range detail.Draft.Scenes {
		require.Empty(t, scene.BackgroundRef)
		require.Empty(t, scene.PortraitRefs)
		require.Empty(t, scene.MusicRef)
		require.Empty(t, scene.SoundEffectRefs)
	}

	second, err := SeedSpeedingP0Local(db, projects, agents, SpeedingP0SeedOptions{TenantID: "default"})
	require.NoError(t, err)
	require.True(t, second.AlreadyExists)
	require.Equal(t, first.ProjectID, second.ProjectID)
	var count int64
	require.NoError(t, db.Model(&gameproject.Project{}).Where("tenant_id=? AND `key`=?", "default", SpeedingP0LocalProjectKey).Count(&count).Error)
	require.EqualValues(t, 1, count)

	updated, err := SeedSpeedingP0Local(db, projects, agents, SpeedingP0SeedOptions{TenantID: "default", UpdateExisting: true})
	require.NoError(t, err)
	require.True(t, updated.ProjectUpdated)
	require.True(t, updated.AlreadyExists)
	require.Equal(t, first.ProjectID, updated.ProjectID)
	refreshed, err := projects.Get("default", first.ProjectID)
	require.NoError(t, err)
	require.Equal(t, float64(30), refreshed.Draft.Actions[0].Effects[0].Value)
	require.Equal(t, 4, refreshed.Draft.Objectives[0].DeadlineTurns)
}

func TestSeedSpeedingP0LocalDryRunDoesNotWrite(t *testing.T) {
	db := setupToolTenantServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&gamepackage.Package{}, &gameproject.Project{}, &gameproject.Release{}))
	for _, name := range []string{"director", "cfo", "counsel"} {
		require.NoError(t, db.Create(&agent.AgentConfig{TenantID: "default", Name: name}).Error)
	}
	result, err := SeedSpeedingP0Local(db, NewGameProjectService(db, NewGamePackageService(db)), NewAgentService("", ""), SpeedingP0SeedOptions{TenantID: "default", DryRun: true})
	require.NoError(t, err)
	require.Contains(t, result.CreatedAgents, "board-rival (planned)")
	var projects, rivals int64
	require.NoError(t, db.Model(&gameproject.Project{}).Count(&projects).Error)
	require.NoError(t, db.Model(&agent.AgentConfig{}).Where("tenant_id=? AND name=?", "default", "board-rival").Count(&rivals).Error)
	require.Zero(t, projects)
	require.Zero(t, rivals)
}
