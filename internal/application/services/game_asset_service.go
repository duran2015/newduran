package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"control-panel/internal/domain/gameasset"
	"control-panel/internal/domain/gamepackage"
	"control-panel/internal/domain/gameproject"
	osscontract "control-panel/pkg/oss"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrGameAssetNotFound = errors.New("游戏资源不存在")
	ErrGameAssetInUse    = errors.New("该资源正在被场景使用，请先从场景中移除")
)

const (
	maxImageAssetBytes = int64(10 << 20)
	maxAudioAssetBytes = int64(30 << 20)
)

type GameAssetDTO struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"projectId"`
	Name         string    `json:"name"`
	Kind         string    `json:"kind"`
	OriginalName string    `json:"originalName"`
	MimeType     string    `json:"mimeType"`
	SizeBytes    int64     `json:"sizeBytes"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	DurationMs   int64     `json:"durationMs"`
	ContentHash  string    `json:"contentHash"`
	URL          string    `json:"url"`
	UsedBy       []string  `json:"usedBy"`
	Available    bool      `json:"available"`
	Problem      string    `json:"problem,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type GameAssetService struct {
	db       *gorm.DB
	uploader osscontract.OSSUploader
	localDir string
}

func NewGameAssetService(db *gorm.DB, uploader osscontract.OSSUploader, localDir string) *GameAssetService {
	return &GameAssetService{db: db, uploader: uploader, localDir: filepath.Clean(localDir)}
}

func validGameAssetKind(kind string) bool {
	switch kind {
	case gameasset.KindBackground, gameasset.KindPortrait, gameasset.KindMusic, gameasset.KindSound:
		return true
	default:
		return false
	}
}

func mediaAllowed(kind, mediaType string) bool {
	if kind == gameasset.KindBackground || kind == gameasset.KindPortrait {
		return mediaType == "image/png" || mediaType == "image/jpeg" || mediaType == "image/gif" || mediaType == "image/webp"
	}
	return mediaType == "audio/mpeg" || mediaType == "audio/wav" || mediaType == "audio/x-wav" || mediaType == "audio/ogg" || mediaType == "audio/mp4"
}

func assetSizeLimit(kind string) int64 {
	if kind == gameasset.KindBackground || kind == gameasset.KindPortrait {
		return maxImageAssetBytes
	}
	return maxAudioAssetBytes
}

func (s *GameAssetService) requireProject(tenantID, projectID string) error {
	var count int64
	if err := s.db.Model(&gameproject.Project{}).Where("tenant_id=? AND id=?", tenantID, projectID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrGameProjectNotFound
	}
	return nil
}

func (s *GameAssetService) OpenRelease(ctx context.Context, tenantID, releaseID, assetID string) (*gameasset.Asset, io.ReadCloser, error) {
	var release gameproject.Release
	if err := s.db.Where("tenant_id=? AND id=?", tenantID, releaseID).First(&release).Error; err != nil {
		return nil, nil, ErrGameAssetNotFound
	}
	var pkg gamepackage.Package
	if err := s.db.Where("tenant_id=? AND id=?", tenantID, release.PackageID).First(&pkg).Error; err != nil {
		return nil, nil, ErrGameAssetNotFound
	}
	raw, _ := json.Marshal(pkg.Manifest)
	var manifest GamePackageManifest
	if err := json.Unmarshal(raw, &manifest); err != nil || manifest.Content == nil {
		return nil, nil, ErrGameAssetNotFound
	}
	found := false
	for _, item := range manifest.Content.Assets {
		if item.ID == assetID {
			found = true
			break
		}
	}
	if !found {
		return nil, nil, ErrGameAssetNotFound
	}
	return s.Open(ctx, tenantID, release.ProjectID, assetID)
}

func (s *GameAssetService) Upload(ctx context.Context, tenantID, projectID, actor, name, kind, originalName string, input io.Reader, declaredSize int64) (*GameAssetDTO, error) {
	name, kind = strings.TrimSpace(name), strings.TrimSpace(kind)
	if name == "" || len([]rune(name)) > 160 {
		return nil, fmt.Errorf("请填写 1–160 个字的资源名称")
	}
	if !validGameAssetKind(kind) {
		return nil, fmt.Errorf("资源类型必须是背景图、人物立绘、音乐或音效")
	}
	if err := s.requireProject(tenantID, projectID); err != nil {
		return nil, err
	}
	limit := assetSizeLimit(kind)
	if declaredSize > limit {
		return nil, fmt.Errorf("文件过大：该类型最多上传 %d MB", limit>>20)
	}
	data, err := io.ReadAll(io.LimitReader(input, limit+1))
	if err != nil {
		return nil, fmt.Errorf("读取上传文件失败: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("上传文件不能为空")
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("文件过大：该类型最多上传 %d MB", limit>>20)
	}
	mediaType := strings.Split(http.DetectContentType(data), ";")[0]
	if !mediaAllowed(kind, mediaType) {
		return nil, fmt.Errorf("文件内容与所选资源类型不匹配")
	}

	id := uuid.NewString()
	safeOriginalName := filepath.Base(strings.ReplaceAll(originalName, "\\", "/"))
	if safeOriginalName == "." || safeOriginalName == "" {
		safeOriginalName = "upload"
	}
	ext := strings.ToLower(filepath.Ext(safeOriginalName))
	if guessed, _ := mime.ExtensionsByType(mediaType); len(guessed) > 0 {
		ext = guessed[0]
	}
	if len(ext) > 8 || strings.ContainsAny(ext, `/\\`) {
		ext = ""
	}
	key := fmt.Sprintf("game-assets/%s/%s/%s%s", tenantID, projectID, id, ext)
	hash := sha256.Sum256(data)
	row := &gameasset.Asset{ID: id, TenantID: tenantID, ProjectID: projectID, Name: name, Kind: kind, OriginalName: safeOriginalName, MimeType: mediaType, SizeBytes: int64(len(data)), ContentHash: hex.EncodeToString(hash[:]), StorageProvider: "local", StorageKey: id, CreatedBy: actor}
	if kind == gameasset.KindBackground || kind == gameasset.KindPortrait {
		if cfg, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil {
			row.Width, row.Height = cfg.Width, cfg.Height
		}
	}
	if s.uploader != nil {
		if _, err := s.uploader.Upload(ctx, key, bytes.NewReader(data), int64(len(data))); err != nil {
			return nil, err
		}
		row.StorageProvider, row.StorageKey = "oss", key
		if err := s.db.Create(row).Error; err != nil {
			_ = s.uploader.Delete(ctx, key)
			return nil, err
		}
	} else {
		if s.localDir == "." || s.localDir == "" {
			return nil, fmt.Errorf("本地资源目录未配置")
		}
		localPath := filepath.Join(s.localDir, projectID, id+ext)
		if err := os.MkdirAll(filepath.Dir(localPath), 0o750); err != nil {
			return nil, err
		}
		tmp, err := os.CreateTemp(filepath.Dir(localPath), ".upload-*")
		if err != nil {
			return nil, err
		}
		tmpName := tmp.Name()
		defer os.Remove(tmpName)
		if _, err := tmp.Write(data); err != nil {
			tmp.Close()
			return nil, err
		}
		if err := tmp.Chmod(0o640); err != nil {
			tmp.Close()
			return nil, err
		}
		if err := tmp.Close(); err != nil {
			return nil, err
		}
		if err := os.Rename(tmpName, localPath); err != nil {
			return nil, err
		}
		row.StorageProvider, row.StorageKey = "local", filepath.ToSlash(filepath.Join(projectID, id+ext))
		if err := s.db.Create(row).Error; err != nil {
			_ = os.Remove(localPath)
			return nil, err
		}
	}
	return s.dto(ctx, row), nil
}

func (s *GameAssetService) dto(ctx context.Context, row *gameasset.Asset) *GameAssetDTO {
	url := fmt.Sprintf("/api/v1/admin/game-projects/%s/assets/%s/content?v=%s", row.ProjectID, row.ID, row.ContentHash[:min(12, len(row.ContentHash))])
	available := true
	problem := ""
	if row.StorageProvider == "oss" && s.uploader != nil {
		if signed, err := s.uploader.GetPresignedURL(ctx, row.StorageKey); err == nil {
			url = signed
		} else {
			available, problem = false, "文件暂时无法读取，请重新上传"
		}
	} else if row.StorageProvider == "local" {
		path, err := s.localPath(row.StorageKey)
		if err != nil {
			available, problem = false, "文件保存位置无效，请重新上传"
		} else if _, err := os.Stat(path); err != nil {
			available, problem = false, "原文件已丢失，请重新上传"
		}
	}
	return &GameAssetDTO{ID: row.ID, ProjectID: row.ProjectID, Name: row.Name, Kind: row.Kind, OriginalName: row.OriginalName, MimeType: row.MimeType, SizeBytes: row.SizeBytes, Width: row.Width, Height: row.Height, DurationMs: row.DurationMs, ContentHash: row.ContentHash, URL: url, UsedBy: []string{}, Available: available, Problem: problem, CreatedAt: row.CreatedAt}
}

func (s *GameAssetService) List(ctx context.Context, tenantID, projectID string) ([]GameAssetDTO, error) {
	if err := s.requireProject(tenantID, projectID); err != nil {
		return nil, err
	}
	var rows []gameasset.Asset
	if err := s.db.Where("tenant_id=? AND project_id=?", tenantID, projectID).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]GameAssetDTO, 0, len(rows))
	var project gameproject.Project
	if err := s.db.Where("tenant_id=? AND id=?", tenantID, projectID).First(&project).Error; err != nil {
		return nil, err
	}
	draft, err := decodeProjectDraft(project.Draft)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		item := s.dto(ctx, &rows[i])
		for _, scene := range draft.Scenes {
			if sceneUsesAsset(scene, rows[i].ID) {
				item.UsedBy = append(item.UsedBy, scene.Name)
			}
		}
		result = append(result, *item)
	}
	return result, nil
}

// Replace changes the underlying file while keeping the resource ID stable,
// so every scene using it immediately receives the new picture or sound.
func (s *GameAssetService) Replace(ctx context.Context, tenantID, projectID, assetID, actor, name, originalName string, input io.Reader, declaredSize int64) (*GameAssetDTO, error) {
	old, err := s.find(tenantID, projectID, assetID)
	if err != nil {
		return nil, err
	}
	if version, used, err := s.publishedUsage(tenantID, projectID, assetID); err != nil {
		return nil, err
	} else if used {
		return nil, fmt.Errorf("该资源已被发布版本 %s 使用，不能直接替换；请上传新资源并发布新版本", version)
	}
	if strings.TrimSpace(name) == "" {
		name = old.Name
	}
	temporary, err := s.Upload(ctx, tenantID, projectID, actor, name, old.Kind, originalName, input, declaredSize)
	if err != nil {
		return nil, err
	}
	newRow, err := s.find(tenantID, projectID, temporary.ID)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{
		"name": newRow.Name, "original_name": newRow.OriginalName, "mime_type": newRow.MimeType,
		"size_bytes": newRow.SizeBytes, "width": newRow.Width, "height": newRow.Height,
		"duration_ms": newRow.DurationMs, "content_hash": newRow.ContentHash,
		"storage_provider": newRow.StorageProvider, "storage_key": newRow.StorageKey, "created_by": actor,
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&gameasset.Asset{}).Where("tenant_id=? AND project_id=? AND id=?", tenantID, projectID, assetID).Updates(updates).Error; err != nil {
			return err
		}
		return tx.Where("tenant_id=? AND project_id=? AND id=?", tenantID, projectID, temporary.ID).Delete(&gameasset.Asset{}).Error
	}); err != nil {
		_ = s.removeStored(ctx, newRow)
		return nil, err
	}
	_ = s.removeStored(ctx, old)
	replaced, err := s.find(tenantID, projectID, assetID)
	if err != nil {
		return nil, err
	}
	return s.dto(ctx, replaced), nil
}

func (s *GameAssetService) publishedUsage(tenantID, projectID, assetID string) (string, bool, error) {
	var releases []gameproject.Release
	if err := s.db.Where("tenant_id=? AND project_id=?", tenantID, projectID).Find(&releases).Error; err != nil {
		return "", false, err
	}
	for _, release := range releases {
		var pkg gamepackage.Package
		if err := s.db.Where("tenant_id=? AND id=?", tenantID, release.PackageID).First(&pkg).Error; err != nil {
			return "", false, err
		}
		manifest, err := decodeGameManifest(pkg)
		if err != nil {
			return "", false, err
		}
		if manifest.Content == nil {
			continue
		}
		for _, asset := range manifest.Content.Assets {
			if asset.ID == assetID {
				return release.Version, true, nil
			}
		}
		for _, scene := range manifest.Content.Scenes {
			if scene.BackgroundAssetID == assetID || scene.MusicAssetID == assetID || scene.BackgroundRef == assetID || scene.MusicRef == assetID {
				return release.Version, true, nil
			}
			for _, id := range append(append([]string{}, scene.SoundEffectAssetIDs...), scene.SoundEffectRefs...) {
				if id == assetID {
					return release.Version, true, nil
				}
			}
			for _, layer := range scene.Characters {
				if layer.PortraitAssetID == assetID {
					return release.Version, true, nil
				}
				for _, id := range layer.EmotionPortraitAssetIDs {
					if id == assetID {
						return release.Version, true, nil
					}
				}
			}
		}
	}
	return "", false, nil
}

func (s *GameAssetService) removeStored(ctx context.Context, row *gameasset.Asset) error {
	if row.StorageProvider == "oss" && s.uploader != nil {
		return s.uploader.Delete(ctx, row.StorageKey)
	}
	if row.StorageProvider == "local" {
		path, err := s.localPath(row.StorageKey)
		if err != nil {
			return err
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (s *GameAssetService) find(tenantID, projectID, assetID string) (*gameasset.Asset, error) {
	var row gameasset.Asset
	if err := s.db.Where("tenant_id=? AND project_id=? AND id=?", tenantID, projectID, assetID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGameAssetNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (s *GameAssetService) Open(ctx context.Context, tenantID, projectID, assetID string) (*gameasset.Asset, io.ReadCloser, error) {
	row, err := s.find(tenantID, projectID, assetID)
	if err != nil {
		return nil, nil, err
	}
	if row.StorageProvider == "oss" {
		if s.uploader == nil {
			return nil, nil, fmt.Errorf("资源存储暂不可用")
		}
		body, err := s.uploader.Download(ctx, row.StorageKey)
		return row, body, err
	}
	path, err := s.localPath(row.StorageKey)
	if err != nil {
		return nil, nil, err
	}
	body, err := os.Open(path)
	return row, body, err
}

func (s *GameAssetService) localPath(key string) (string, error) {
	root, err := filepath.Abs(s.localDir)
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(key)))
	if err != nil {
		return "", err
	}
	if path != root && !strings.HasPrefix(path, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("无效的资源存储路径")
	}
	return path, nil
}

func sceneUsesAsset(scene GameProjectScene, id string) bool {
	if scene.BackgroundAssetID == id || scene.MusicAssetID == id {
		return true
	}
	for _, ref := range scene.SoundEffectAssetIDs {
		if ref == id {
			return true
		}
	}
	for _, item := range scene.Characters {
		if item.PortraitAssetID == id {
			return true
		}
		for _, variantID := range item.EmotionPortraitAssetIDs {
			if variantID == id {
				return true
			}
		}
	}
	// Compatibility with early E0 drafts.
	if scene.BackgroundRef == id || scene.MusicRef == id {
		return true
	}
	for _, ref := range scene.SoundEffectRefs {
		if ref == id {
			return true
		}
	}
	for _, ref := range scene.PortraitRefs {
		if ref == id {
			return true
		}
	}
	return false
}

func (s *GameAssetService) Delete(ctx context.Context, tenantID, projectID, assetID string) error {
	row, err := s.find(tenantID, projectID, assetID)
	if err != nil {
		return err
	}
	var project gameproject.Project
	if err := s.db.Where("tenant_id=? AND id=?", tenantID, projectID).First(&project).Error; err != nil {
		return err
	}
	draft, err := decodeProjectDraft(project.Draft)
	if err != nil {
		return err
	}
	for _, scene := range draft.Scenes {
		if sceneUsesAsset(scene, assetID) {
			return fmt.Errorf("%w：场景“%s”正在使用它", ErrGameAssetInUse, scene.Name)
		}
	}
	var releases []gameproject.Release
	if err := s.db.Where("tenant_id=? AND project_id=?", tenantID, projectID).Find(&releases).Error; err != nil {
		return err
	}
	for _, release := range releases {
		var pkg gamepackage.Package
		if err := s.db.Where("tenant_id=? AND id=?", tenantID, release.PackageID).First(&pkg).Error; err != nil {
			return err
		}
		manifest, err := decodeGameManifest(pkg)
		if err != nil {
			return err
		}
		if manifest.Content == nil {
			continue
		}
		for _, asset := range manifest.Content.Assets {
			if asset.ID == assetID {
				return fmt.Errorf("该资源已被发布版本 %s 使用，不能删除", release.Version)
			}
		}
		// Compatibility with releases made before asset snapshots existed.
		for _, scene := range manifest.Content.Scenes {
			if scene.BackgroundAssetID == assetID || scene.MusicAssetID == assetID || scene.BackgroundRef == assetID || scene.MusicRef == assetID {
				return fmt.Errorf("该资源已被发布版本 %s 使用，不能删除", release.Version)
			}
			for _, id := range append(append([]string{}, scene.SoundEffectAssetIDs...), scene.SoundEffectRefs...) {
				if id == assetID {
					return fmt.Errorf("该资源已被发布版本 %s 使用，不能删除", release.Version)
				}
			}
			for _, layer := range scene.Characters {
				if layer.PortraitAssetID == assetID {
					return fmt.Errorf("该资源已被发布版本 %s 使用，不能删除", release.Version)
				}
				for _, variantID := range layer.EmotionPortraitAssetIDs {
					if variantID == assetID {
						return fmt.Errorf("该资源已被发布版本 %s 使用，不能删除", release.Version)
					}
				}
			}
			for _, id := range scene.PortraitRefs {
				if id == assetID {
					return fmt.Errorf("该资源已被发布版本 %s 使用，不能删除", release.Version)
				}
			}
		}
	}
	if err := s.db.Where("tenant_id=? AND project_id=? AND id=?", tenantID, projectID, assetID).Delete(&gameasset.Asset{}).Error; err != nil {
		return err
	}
	if row.StorageProvider == "oss" && s.uploader != nil {
		return s.uploader.Delete(ctx, row.StorageKey)
	}
	if row.StorageProvider == "local" {
		if path, err := s.localPath(row.StorageKey); err == nil {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	return nil
}
