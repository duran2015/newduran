package services

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"path/filepath"
	"testing"

	"control-panel/internal/domain/gameasset"
	"control-panel/internal/domain/gamepackage"
	"control-panel/internal/domain/gameproject"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func gameAssetTestService(t *testing.T) (*GameAssetService, *gorm.DB, string, string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&gameproject.Project{}, &gameproject.Release{}, &gamepackage.Package{}, &gameasset.Asset{}))
	projectID := "project-a"
	require.NoError(t, db.Create(&gameproject.Project{ID: projectID, TenantID: "tenant-a", Key: "demo", Name: "Demo", Genre: "story", PlayGoal: "survive", Status: "draft", Revision: 1, Draft: encodeProjectDraft(emptyGameProjectDraft())}).Error)
	return NewGameAssetService(db, nil, t.TempDir()), db, "tenant-a", projectID
}

func tinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 3))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func TestGameAssetUploadLocalStorageAndTenantIsolation(t *testing.T) {
	svc, db, tenantID, projectID := gameAssetTestService(t)
	data := tinyPNG(t)
	asset, err := svc.Upload(context.Background(), tenantID, projectID, "pm", "会议室", gameasset.KindBackground, `..\meeting.png`, bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	require.Equal(t, "meeting.png", asset.OriginalName)
	require.Equal(t, 2, asset.Width)
	require.Equal(t, 3, asset.Height)
	require.NotEmpty(t, asset.ContentHash)

	items, err := svc.List(context.Background(), tenantID, projectID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	_, _, err = svc.Open(context.Background(), "tenant-b", projectID, asset.ID)
	require.ErrorIs(t, err, ErrGameAssetNotFound)

	row := gameasset.Asset{}
	require.NoError(t, db.First(&row, "id=?", asset.ID).Error)
	require.Equal(t, "local", row.StorageProvider)
	require.False(t, filepath.IsAbs(row.StorageKey))
	_, body, err := svc.Open(context.Background(), tenantID, projectID, asset.ID)
	require.NoError(t, err)
	got, err := io.ReadAll(body)
	require.NoError(t, err)
	require.NoError(t, body.Close())
	require.Equal(t, data, got)
}

func TestGameAssetRejectsTypeMismatchAndReferencedDelete(t *testing.T) {
	svc, db, tenantID, projectID := gameAssetTestService(t)
	data := tinyPNG(t)
	_, err := svc.Upload(context.Background(), tenantID, projectID, "pm", "假音乐", gameasset.KindMusic, "music.mp3", bytes.NewReader(data), int64(len(data)))
	require.Error(t, err)

	asset, err := svc.Upload(context.Background(), tenantID, projectID, "pm", "背景", gameasset.KindBackground, "bg.png", bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	var project gameproject.Project
	require.NoError(t, db.First(&project, "id=?", projectID).Error)
	draft, err := decodeProjectDraft(project.Draft)
	require.NoError(t, err)
	draft.Scenes = []GameProjectScene{{ID: "office", Name: "办公室", BackgroundAssetID: asset.ID}}
	raw, err := json.Marshal(draft)
	require.NoError(t, err)
	require.NoError(t, db.Model(&project).Update("draft_json", string(raw)).Error)
	items, err := svc.List(context.Background(), tenantID, projectID)
	require.NoError(t, err)
	require.Equal(t, []string{"办公室"}, items[0].UsedBy)
	require.ErrorIs(t, svc.Delete(context.Background(), tenantID, projectID, asset.ID), ErrGameAssetInUse)
}

func TestGameAssetReplaceKeepsSceneReferenceAndRemovesOldFile(t *testing.T) {
	svc, db, tenantID, projectID := gameAssetTestService(t)
	oldData := tinyPNG(t)
	asset, err := svc.Upload(context.Background(), tenantID, projectID, "pm", "会议室", gameasset.KindBackground, "old.png", bytes.NewReader(oldData), int64(len(oldData)))
	require.NoError(t, err)
	var oldRow gameasset.Asset
	require.NoError(t, db.First(&oldRow, "id=?", asset.ID).Error)
	oldPath, err := svc.localPath(oldRow.StorageKey)
	require.NoError(t, err)

	img := image.NewRGBA(image.Rect(0, 0, 5, 4))
	var replacement bytes.Buffer
	require.NoError(t, png.Encode(&replacement, img))
	replaced, err := svc.Replace(context.Background(), tenantID, projectID, asset.ID, "pm", "新会议室", "new.png", bytes.NewReader(replacement.Bytes()), int64(replacement.Len()))
	require.NoError(t, err)
	require.Equal(t, asset.ID, replaced.ID)
	require.Equal(t, "新会议室", replaced.Name)
	require.Equal(t, 5, replaced.Width)
	require.Equal(t, 4, replaced.Height)
	require.NotEqual(t, asset.URL, replaced.URL, "替换后地址版本必须变化，避免浏览器继续显示旧文件")
	require.NoFileExists(t, oldPath)

	items, err := svc.List(context.Background(), tenantID, projectID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	_, body, err := svc.Open(context.Background(), tenantID, projectID, asset.ID)
	require.NoError(t, err)
	got, err := io.ReadAll(body)
	require.NoError(t, err)
	require.NoError(t, body.Close())
	require.Equal(t, replacement.Bytes(), got)
}

func TestGameAssetPublishedFileCannotBeReplacedInPlace(t *testing.T) {
	svc, db, tenantID, projectID := gameAssetTestService(t)
	data := tinyPNG(t)
	asset, err := svc.Upload(context.Background(), tenantID, projectID, "pm", "已发布背景", gameasset.KindBackground, "published.png", bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	manifest := GamePackageManifest{SchemaVersion: GamePackageSchemaV1, Package: GamePackageMetadata{Key: "game.demo", Version: "1.0.0", DisplayName: "Demo"}, Content: &GamePackageContent{ProjectID: projectID, Assets: []GameAssetSnapshot{{ID: asset.ID, Kind: gameasset.KindBackground, ContentHash: asset.ContentHash}}}}
	raw, err := json.Marshal(manifest)
	require.NoError(t, err)
	var manifestMap map[string]any
	require.NoError(t, json.Unmarshal(raw, &manifestMap))
	require.NoError(t, db.Create(&gamepackage.Package{ID: "package-1", TenantID: tenantID, Key: "game.demo", Version: "1.0.0", DisplayName: "Demo", Status: gamepackage.StatusEnabled, Manifest: manifestMap, ContentHash: "hash"}).Error)
	require.NoError(t, db.Create(&gameproject.Release{ID: "release-1", TenantID: tenantID, ProjectID: projectID, Version: "1.0.0", PackageID: "package-1", ContentHash: "hash"}).Error)

	_, err = svc.Replace(context.Background(), tenantID, projectID, asset.ID, "pm", "新背景", "new.png", bytes.NewReader(data), int64(len(data)))
	require.Error(t, err)
	require.Contains(t, err.Error(), "不能直接替换")
}
