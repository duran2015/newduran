package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"control-panel/internal/domain/agent"
	"control-panel/internal/domain/gameasset"
	"control-panel/internal/domain/gameproject"
)

func e4AssetService(t *testing.T) (*GameAssetService, *GameProjectService) {
	t.Helper()
	projects, db := gameProjectTestService(t)
	if err := db.AutoMigrate(&gameasset.Asset{}); err != nil {
		t.Fatal(err)
	}
	return NewGameAssetService(db, nil, t.TempDir()), projects
}

func onePixelPNG(t *testing.T) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestE4AssetUploadChecksContentSizeTenantAndSafeName(t *testing.T) {
	assets, projects := e4AssetService(t)
	project := createGameProjectForTest(t, projects)
	ctx := context.Background()

	got, err := assets.Upload(ctx, "tenant-a", project.Project.ID, "editor", "董事会议室", gameasset.KindBackground, "../素材/会议室.png", bytes.NewReader(onePixelPNG(t)), int64(len(onePixelPNG(t))))
	if err != nil {
		t.Fatal(err)
	}
	if got.OriginalName != "会议室.png" || got.MimeType != "image/png" || got.Width != 1 || got.Height != 1 {
		t.Fatalf("上传后的安全文件名或图片信息不正确: %#v", got)
	}
	if got.URL == "" || strings.Contains(got.URL, "../") {
		t.Fatalf("资源地址不应暴露原文件路径: %q", got.URL)
	}
	// Reconstructing the service simulates an application/container restart.
	// Metadata stays in the database and bytes stay in the mounted media path.
	restarted := NewGameAssetService(assets.db, nil, assets.localDir)
	_, restartedBody, err := restarted.Open(ctx, "tenant-a", project.Project.ID, got.ID)
	if err != nil {
		t.Fatalf("服务重启后仍应能读取已上传素材: %v", err)
	}
	restartedBody.Close()

	if _, err := assets.Upload(ctx, "tenant-a", project.Project.ID, "editor", "伪装音乐", gameasset.KindMusic, "music.mp3", bytes.NewReader(onePixelPNG(t)), int64(len(onePixelPNG(t)))); err == nil {
		t.Fatal("图片内容不能伪装成音乐上传")
	}
	if _, err := assets.Upload(ctx, "tenant-a", project.Project.ID, "editor", "过大背景", gameasset.KindBackground, "huge.png", bytes.NewReader(onePixelPNG(t)), maxImageAssetBytes+1); err == nil || !strings.Contains(err.Error(), "10 MB") {
		t.Fatalf("超过图片上限应给出可理解原因，err=%v", err)
	}
	if _, err := assets.List(ctx, "tenant-b", project.Project.ID); err == nil {
		t.Fatal("其他租户不能读取本租户项目的资源")
	}
	if _, _, err := assets.Open(ctx, "tenant-b", project.Project.ID, got.ID); err == nil {
		t.Fatal("其他租户不能读取资源文件内容")
	}
	if err := assets.Delete(ctx, "tenant-b", project.Project.ID, got.ID); err == nil {
		t.Fatal("其他租户不能删除资源")
	}
}

func TestE4ReferencedAssetCannotBeDeleted(t *testing.T) {
	assets, projects := e4AssetService(t)
	project := createGameProjectForTest(t, projects)
	ctx := context.Background()
	asset, err := assets.Upload(ctx, "tenant-a", project.Project.ID, "editor", "办公室", gameasset.KindBackground, "office.png", bytes.NewReader(onePixelPNG(t)), int64(len(onePixelPNG(t))))
	if err != nil {
		t.Fatal(err)
	}

	draft := project.Draft
	draft.Scenes = []GameProjectScene{{ID: "office", Name: "办公室", BackgroundAssetID: asset.ID}}
	if _, err := projects.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{
		Revision: project.Project.Revision, Name: project.Project.Name, Description: project.Project.Description,
		Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft,
	}); err != nil {
		t.Fatal(err)
	}
	if err := assets.Delete(ctx, "tenant-a", project.Project.ID, asset.ID); err == nil || !strings.Contains(err.Error(), "正在被场景使用") {
		t.Fatalf("正在使用的资源必须阻止删除并说明原因，err=%v", err)
	}
	opened, body, err := assets.Open(ctx, "tenant-a", project.Project.ID, asset.ID)
	if err != nil {
		t.Fatalf("删除失败后资源仍应存在: %v", err)
	}
	body.Close()
	if opened.ID != asset.ID {
		t.Fatalf("删除保护后读到了错误资源: %#v", opened)
	}
}

func TestE4SceneCRUDValidatesGameCharactersAssetKindsAndOrder(t *testing.T) {
	assets, projects := e4AssetService(t)
	project := createGameProjectForTest(t, projects)
	ctx := context.Background()
	agentRow := agent.AgentConfig{Name: "cfo-e4", TenantID: "tenant-a", ContentHash: "v1", SystemPrompt: "finance"}
	if err := assets.db.Create(&agentRow).Error; err != nil {
		t.Fatal(err)
	}
	draft := project.Draft
	draft.NPCs = []GameProjectNPC{{ID: "cfo", AgentID: agentRow.ID, DisplayName: "莫云岑", Role: "首席财务官", CharacterType: GameCharacterNPC}}
	withCharacter, err := projects.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{
		Revision: project.Project.Revision, Name: project.Project.Name, Description: project.Project.Description,
		Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft,
	})
	if err != nil {
		t.Fatal(err)
	}
	background, err := assets.Upload(ctx, "tenant-a", project.Project.ID, "editor", "办公室", gameasset.KindBackground, "office.png", bytes.NewReader(onePixelPNG(t)), int64(len(onePixelPNG(t))))
	if err != nil {
		t.Fatal(err)
	}
	portrait, err := assets.Upload(ctx, "tenant-a", project.Project.ID, "editor", "莫云岑", gameasset.KindPortrait, "cfo.png", bytes.NewReader(onePixelPNG(t)), int64(len(onePixelPNG(t))))
	if err != nil {
		t.Fatal(err)
	}

	invalid := GameProjectScene{ID: "wrong-kind", Name: "错误场景", BackgroundAssetID: portrait.ID, Characters: []GameSceneCharacter{}}
	if _, err := projects.CreateScene("tenant-a", project.Project.ID, "editor", SaveGameSceneInput{Revision: withCharacter.Project.Revision, Scene: invalid}); err == nil || !strings.Contains(err.Error(), "资源类型不正确") {
		t.Fatalf("人物立绘不能当背景图使用，err=%v", err)
	}
	invalid.BackgroundAssetID = background.ID
	invalid.Characters = []GameSceneCharacter{{ID: "outsider-layer", CharacterID: "outsider", PortraitAssetID: portrait.ID, X: 50, Y: 100, Scale: 1, Visible: true}}
	if _, err := projects.CreateScene("tenant-a", project.Project.ID, "editor", SaveGameSceneInput{Revision: withCharacter.Project.Revision, Scene: invalid}); err == nil || !strings.Contains(err.Error(), "不存在的游戏人物") {
		t.Fatalf("场景人物必须来自本游戏绑定，err=%v", err)
	}

	first := GameProjectScene{ID: "office", Name: "办公室", Order: 0, BackgroundAssetID: background.ID, SoundEffectAssetIDs: []string{}, Characters: []GameSceneCharacter{{ID: "cfo-layer", CharacterID: "cfo", PortraitAssetID: portrait.ID, X: 18, Y: 100, Scale: 1, Layer: 1, Visible: true}}}
	created, err := projects.CreateScene("tenant-a", project.Project.ID, "editor", SaveGameSceneInput{Revision: withCharacter.Project.Revision, Scene: first})
	if err != nil {
		t.Fatal(err)
	}
	copyScene := first
	copyScene.ID, copyScene.Name, copyScene.Order = "office-copy", "办公室（副本）", 1
	copyScene.Characters = append([]GameSceneCharacter(nil), first.Characters...)
	copyScene.Characters[0].ID = "cfo-layer-copy"
	createdCopy, err := projects.CreateScene("tenant-a", project.Project.ID, "editor", SaveGameSceneInput{Revision: created.Project.Revision, Scene: copyScene})
	if err != nil {
		t.Fatal(err)
	}
	if len(createdCopy.Draft.Scenes) != 2 || createdCopy.Draft.Scenes[1].BackgroundAssetID != background.ID || createdCopy.Draft.Scenes[1].Characters[0].PortraitAssetID != portrait.ID {
		t.Fatalf("复制场景应保留资源引用: %#v", createdCopy.Draft.Scenes)
	}
	deleted, err := projects.DeleteScene("tenant-a", project.Project.ID, "office-copy", "editor", createdCopy.Project.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted.Draft.Scenes) != 1 || deleted.Draft.Scenes[0].ID != "office" {
		t.Fatalf("删除一个场景不应影响其他场景: %#v", deleted.Draft.Scenes)
	}
	if _, body, err := assets.Open(ctx, "tenant-a", project.Project.ID, background.ID); err != nil {
		t.Fatalf("删除场景不能删除资源文件: %v", err)
	} else {
		body.Close()
	}
}

func TestE4PublishedSceneSnapshotDoesNotChangeWithDraft(t *testing.T) {
	assets, projects := e4AssetService(t)
	project := createGameProjectForTest(t, projects)
	ctx := context.Background()
	background, err := assets.Upload(ctx, "tenant-a", project.Project.ID, "editor", "会议室", gameasset.KindBackground, "boardroom.png", bytes.NewReader(onePixelPNG(t)), int64(len(onePixelPNG(t))))
	if err != nil {
		t.Fatal(err)
	}

	// Publication requires the earlier editor steps as well; this acceptance
	// test focuses on the scene snapshot and uses the normal complete fixture.
	draft := completeGameDraft(0)
	draft.NPCs = nil
	draft.Events[0].ParticipantIDs = nil
	draft.Scenes = []GameProjectScene{{ID: "boardroom", Name: "董事会议室", Order: 1, BackgroundAssetID: background.ID}}
	updated, err := projects.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{
		Revision: project.Project.Revision, Name: project.Project.Name, Description: project.Project.Description,
		Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft,
	})
	if err != nil {
		t.Fatal(err)
	}
	release, err := projects.Publish("tenant-a", project.Project.ID, "publisher", PublishGameProjectInput{Version: "0.4.0"})
	if err != nil {
		t.Fatal(err)
	}
	published, err := projects.packages.Get("tenant-a", release.PackageID)
	if err != nil {
		t.Fatal(err)
	}
	if published.Manifest.Content == nil || len(published.Manifest.Content.Scenes) != 1 || published.Manifest.Content.Scenes[0].BackgroundAssetID != background.ID {
		t.Fatalf("发布版本没有冻结场景资源引用: %#v", published.Manifest.Content)
	}

	changed := updated.Draft
	changed.Scenes[0].Name = "草稿中的新名称"
	changed.Scenes[0].BackgroundAssetID = ""
	if _, err := projects.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{
		Revision: updated.Project.Revision, Name: updated.Project.Name, Description: updated.Project.Description,
		Genre: updated.Project.Genre, PlayGoal: updated.Project.PlayGoal, Draft: &changed,
	}); err != nil {
		t.Fatal(err)
	}
	publishedAgain, err := projects.packages.Get("tenant-a", release.PackageID)
	if err != nil {
		t.Fatal(err)
	}
	if publishedAgain.Manifest.Content.Scenes[0].Name != "董事会议室" || publishedAgain.Manifest.Content.Scenes[0].BackgroundAssetID != background.ID {
		t.Fatalf("修改草稿不应污染旧发布版本: %#v", publishedAgain.Manifest.Content.Scenes[0])
	}
	if err := assets.Delete(ctx, "tenant-a", project.Project.ID, background.ID); err == nil || !strings.Contains(err.Error(), "发布") {
		t.Fatalf("旧发布版本仍在使用的文件不能删除，否则无法完整回放，err=%v", err)
	}
}

// Compile-time guard: E4 scenes live inside the project draft rather than a
// second independently editable source of truth.
var _ = gameproject.Project{}
