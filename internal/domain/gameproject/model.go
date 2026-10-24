// Package gameproject stores editable game projects and their immutable releases.
package gameproject

import "time"

const StatusDraft = "draft"

// Project is the creator-facing source of truth. Draft contains authored
// rules, Agent references, story events and media references. Published game
// packages are snapshots and are never used as an editing source.
type Project struct {
	ID          string         `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID    string         `gorm:"type:varchar(64);not null;uniqueIndex:uk_game_projects_key,priority:1;index" json:"-"`
	Key         string         `gorm:"type:varchar(128);not null;uniqueIndex:uk_game_projects_key,priority:2" json:"key"`
	Name        string         `gorm:"type:varchar(160);not null" json:"name"`
	Description string         `gorm:"type:text;not null" json:"description"`
	Genre       string         `gorm:"type:varchar(96);not null" json:"genre"`
	PlayGoal    string         `gorm:"type:text;not null" json:"playGoal"`
	Status      string         `gorm:"type:varchar(24);not null;default:'draft';index" json:"status"`
	Revision    int            `gorm:"not null;default:1" json:"revision"`
	Draft       map[string]any `gorm:"column:draft_json;type:json;serializer:json" json:"-"`
	CreatedBy   string         `gorm:"type:varchar(160);not null;default:''" json:"createdBy"`
	UpdatedBy   string         `gorm:"type:varchar(160);not null;default:''" json:"updatedBy"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

func (Project) TableName() string { return "game_projects" }

// Release records exactly which immutable game-package version was generated
// from a project revision.
type Release struct {
	ID              string    `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID        string    `gorm:"type:varchar(64);not null;uniqueIndex:uk_game_project_release,priority:1;index" json:"-"`
	ProjectID       string    `gorm:"type:char(36);not null;uniqueIndex:uk_game_project_release,priority:2;index" json:"projectId"`
	Version         string    `gorm:"type:varchar(64);not null;uniqueIndex:uk_game_project_release,priority:3" json:"version"`
	ProjectRevision int       `gorm:"not null" json:"projectRevision"`
	PackageID       string    `gorm:"type:char(36);not null;index" json:"packageId"`
	ContentHash     string    `gorm:"type:char(64);not null" json:"contentHash"`
	ReleaseNotes    string    `gorm:"type:text;not null" json:"releaseNotes"`
	PublishedBy     string    `gorm:"type:varchar(160);not null;default:''" json:"publishedBy"`
	PublishedAt     time.Time `json:"publishedAt"`
}

func (Release) TableName() string { return "game_project_releases" }

// Session is the game-specific projection of a generic Run. The Run owns
// lifecycle, participants, mutable state and the append-only timeline; this
// row only pins the authored game revision used by that play-through.
type Session struct {
	ID                  string         `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID            string         `gorm:"type:varchar(64);not null;index;uniqueIndex:uk_game_session_run,priority:1" json:"-"`
	RunID               string         `gorm:"type:char(36);not null;uniqueIndex:uk_game_session_run,priority:2;index" json:"runId"`
	ProjectID           string         `gorm:"type:char(36);not null;index:idx_game_sessions_project,priority:1" json:"projectId"`
	ProjectRevision     int            `gorm:"not null" json:"projectRevision"`
	Source              string         `gorm:"type:varchar(24);not null;default:'draft'" json:"source"`
	ReleaseID           string         `gorm:"type:char(36);not null;default:'';index" json:"releaseId,omitempty"`
	StartTurn           int            `gorm:"not null;default:0" json:"startTurn"`
	StartEventID        string         `gorm:"type:varchar(128);not null;default:''" json:"startEventId,omitempty"`
	DirectorCharacterID string         `gorm:"type:varchar(128);not null" json:"directorCharacterId"`
	Snapshot            map[string]any `gorm:"type:json;serializer:json" json:"-"`
	CreatedBy           string         `gorm:"type:varchar(160);not null;default:''" json:"createdBy"`
	CreatedAt           time.Time      `gorm:"index:idx_game_sessions_project,priority:2" json:"createdAt"`
	UpdatedAt           time.Time      `json:"updatedAt"`
}

func (Session) TableName() string { return "game_sessions" }

// PlayerPlan is an interpreted player instruction awaiting explicit
// confirmation. Interpretation never mutates game state; only confirmation
// settles its frozen effects.
type PlayerPlan struct {
	ID             string         `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID       string         `gorm:"type:varchar(64);not null;index;uniqueIndex:uk_player_plan_request,priority:1" json:"-"`
	SessionID      string         `gorm:"type:char(36);not null;index;uniqueIndex:uk_player_plan_request,priority:2" json:"sessionId"`
	RequestKey     string         `gorm:"type:varchar(160);not null;uniqueIndex:uk_player_plan_request,priority:3" json:"-"`
	InputText      string         `gorm:"type:text;not null" json:"inputText"`
	Summary        string         `gorm:"type:text;not null" json:"summary"`
	ActionID       string         `gorm:"type:varchar(128);not null;default:''" json:"actionId,omitempty"`
	Effects        map[string]any `gorm:"type:json;serializer:json" json:"effects"`
	Status         string         `gorm:"type:varchar(24);not null;default:'pending';index" json:"status"`
	ConfirmedByKey string         `gorm:"type:varchar(160);not null;default:'';index" json:"-"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

func (PlayerPlan) TableName() string { return "game_player_plans" }
