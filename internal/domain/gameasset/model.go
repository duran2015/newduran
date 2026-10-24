// Package gameasset stores reusable 2D media owned by one game project.
// It is intentionally separate from domain/scene, whose "scene" means an
// Agent prompt preset rather than a visual game scene.
package gameasset

import "time"

const (
	KindBackground = "background"
	KindPortrait   = "portrait"
	KindMusic      = "music"
	KindSound      = "sound"
)

// Asset is metadata only. Object bytes are accessed through a storage
// provider so moving from the built-in database provider to OSS later does
// not change project drafts or published game packages.
type Asset struct {
	ID              string    `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID        string    `gorm:"type:varchar(64);not null;index:idx_game_assets_project,priority:1" json:"-"`
	ProjectID       string    `gorm:"type:char(36);not null;index:idx_game_assets_project,priority:2" json:"projectId"`
	Name            string    `gorm:"type:varchar(160);not null" json:"name"`
	Kind            string    `gorm:"type:varchar(24);not null;index" json:"kind"`
	OriginalName    string    `gorm:"type:varchar(255);not null" json:"originalName"`
	MimeType        string    `gorm:"type:varchar(96);not null" json:"mimeType"`
	SizeBytes       int64     `gorm:"not null" json:"sizeBytes"`
	Width           int       `gorm:"not null;default:0" json:"width"`
	Height          int       `gorm:"not null;default:0" json:"height"`
	DurationMs      int64     `gorm:"not null;default:0" json:"durationMs"`
	ContentHash     string    `gorm:"type:char(64);not null" json:"contentHash"`
	StorageProvider string    `gorm:"type:varchar(32);not null" json:"-"`
	StorageKey      string    `gorm:"type:varchar(512);not null" json:"-"`
	CreatedBy       string    `gorm:"type:varchar(160);not null;default:''" json:"createdBy,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

func (Asset) TableName() string { return "game_assets" }
