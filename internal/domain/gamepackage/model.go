// Package gamepackage owns the generic, tenant-scoped game package registry.
// It deliberately contains no game-specific vocabulary: a package is an
// immutable manifest version plus a tenant-local activation flag.
package gamepackage

import "time"

const (
	StatusEnabled  = "enabled"
	StatusDisabled = "disabled"
)

// Package is one immutable GamePackageManifest version. Key is stable across
// versions while ID identifies the exact version installed by a tenant.
type Package struct {
	ID          string         `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID    string         `gorm:"type:varchar(64);not null;uniqueIndex:uk_game_packages,priority:1;index" json:"-"`
	Key         string         `gorm:"type:varchar(253);not null;uniqueIndex:uk_game_packages,priority:2;index" json:"key"`
	Version     string         `gorm:"type:varchar(64);not null;uniqueIndex:uk_game_packages,priority:3" json:"version"`
	DisplayName string         `gorm:"type:varchar(160);not null" json:"displayName"`
	Description string         `gorm:"type:text;not null" json:"description"`
	Status      string         `gorm:"type:varchar(24);not null;default:'disabled';index" json:"status"`
	Manifest    map[string]any `gorm:"column:manifest_json;type:json;serializer:json" json:"manifest"`
	ContentHash string         `gorm:"type:char(64);not null" json:"contentHash"`
	CreatedBy   string         `gorm:"type:varchar(160);not null;default:''" json:"createdBy"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

func (Package) TableName() string { return "game_packages" }
