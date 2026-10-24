package services

import (
	"strings"
	"testing"

	"control-panel/internal/domain/agent"
	"control-panel/internal/domain/gamepackage"
	"control-panel/internal/domain/gameproject"
)

func publishableE5Project(t *testing.T) (*GameProjectService, *GameProjectDetail) {
	t.Helper()
	service, db := gameProjectTestService(t)
	actor := agent.AgentConfig{Name: "e5-director", TenantID: "tenant-a", ContentHash: "director-v1", SystemPrompt: "推进合法事件"}
	if err := db.Create(&actor).Error; err != nil {
		t.Fatal(err)
	}
	project := createGameProjectForTest(t, service)
	draft := completeGameDraft(actor.ID)
	draft.NPCs[0].CharacterType = GameCharacterDirector
	draft.NPCs[0].Permissions = []string{"advance_turn"}
	updated, err := service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{
		Revision: project.Project.Revision, Name: project.Project.Name, Description: project.Project.Description,
		Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft,
	})
	if err != nil {
		t.Fatal(err)
	}
	return service, updated
}

func TestE5PublishingSameVersionAndContentIsIdempotent(t *testing.T) {
	service, project := publishableE5Project(t)
	first, err := service.Publish("tenant-a", project.Project.ID, "publisher", PublishGameProjectInput{Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Publish("tenant-a", project.Project.ID, "publisher", PublishGameProjectInput{Version: "1.0.0"})
	if err != nil {
		t.Fatalf("重复提交同一次发布应返回原版本，而不是产生错误: %v", err)
	}
	if first.ID != second.ID || first.PackageID != second.PackageID || first.ContentHash != second.ContentHash {
		t.Fatalf("重复发布产生了不同版本: first=%#v second=%#v", first, second)
	}
	var releases, packages int64
	service.db.Model(&gameproject.Release{}).Where("tenant_id=? AND project_id=? AND version=?", "tenant-a", project.Project.ID, "1.0.0").Count(&releases)
	service.db.Model(&gamepackage.Package{}).Where("tenant_id=? AND id=?", "tenant-a", first.PackageID).Count(&packages)
	if releases != 1 || packages != 1 {
		t.Fatalf("一次发布请求只能留下一个版本和一个游戏包, releases=%d packages=%d", releases, packages)
	}
}

func TestE5SameVersionCannotBeReplacedByChangedDraft(t *testing.T) {
	service, project := publishableE5Project(t)
	release, err := service.Publish("tenant-a", project.Project.ID, "publisher", PublishGameProjectInput{Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	original, err := service.packages.Get("tenant-a", release.PackageID)
	if err != nil {
		t.Fatal(err)
	}

	draft := project.Draft
	draft.Metrics[0].Initial = float64(42)
	draft.Events[0].Name = "草稿中的新事件文案"
	changed, err := service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{
		Revision: project.Project.Revision, Name: project.Project.Name, Description: project.Project.Description,
		Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Publish("tenant-a", changed.Project.ID, "publisher", PublishGameProjectInput{Version: "1.0.0"}); err == nil || !strings.Contains(err.Error(), "different content") {
		t.Fatalf("同一个版本号不能被不同草稿覆盖, err=%v", err)
	}
	stillOriginal, err := service.packages.Get("tenant-a", release.PackageID)
	if err != nil {
		t.Fatal(err)
	}
	if stillOriginal.Package.ContentHash != original.Package.ContentHash || stillOriginal.Manifest.Registry.Metrics[0].Initial != float64(100) {
		t.Fatalf("失败的覆盖发布污染了旧版本: %#v", stillOriginal.Manifest.Registry.Metrics[0])
	}
	var releases int64
	service.db.Model(&gameproject.Release{}).Where("tenant_id=? AND project_id=?", "tenant-a", project.Project.ID).Count(&releases)
	if releases != 1 {
		t.Fatalf("覆盖失败后只能保留原版本, releases=%d", releases)
	}
}

func TestE5PublishAndReleaseListRespectTenantBoundary(t *testing.T) {
	service, project := publishableE5Project(t)
	release, err := service.Publish("tenant-a", project.Project.ID, "publisher", PublishGameProjectInput{Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get("tenant-b", project.Project.ID); err == nil {
		t.Fatal("其他组织不应读取项目及其版本列表")
	}
	if _, err := service.packages.Get("tenant-b", release.PackageID); err == nil {
		t.Fatal("其他组织不应读取已发布游戏包")
	}
	if _, err := service.Publish("tenant-b", project.Project.ID, "intruder", PublishGameProjectInput{Version: "2.0.0"}); err == nil {
		t.Fatal("其他组织不应发布本组织项目")
	}
}

func TestE5FailedValidationLeavesNoPartialRelease(t *testing.T) {
	service, db := gameProjectTestService(t)
	project := createGameProjectForTest(t, service)
	if _, err := service.Publish("tenant-a", project.Project.ID, "publisher", PublishGameProjectInput{Version: "1.0.0"}); err == nil {
		t.Fatal("不完整草稿不应发布成功")
	}
	var releases, packages int64
	db.Model(&gameproject.Release{}).Where("tenant_id=? AND project_id=?", "tenant-a", project.Project.ID).Count(&releases)
	db.Model(&gamepackage.Package{}).Where("tenant_id=?", "tenant-a").Count(&packages)
	if releases != 0 || packages != 0 {
		t.Fatalf("发布检查失败后不能留下半个版本, releases=%d packages=%d", releases, packages)
	}
}
