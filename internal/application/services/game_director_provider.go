package services

import (
	"context"
	"control-panel/internal/domain/agent"
	"gorm.io/gorm"
)

// GameAgentModelRunner references the actual Hub Agent configuration while
// intentionally exposing no tools, inbox, history or other agents' knowledge.
type GameAgentModelRunner struct {
	db        *gorm.DB
	providers *ProviderService
}

func NewGameAgentModelRunner(db *gorm.DB, providers *ProviderService) *GameAgentModelRunner {
	return &GameAgentModelRunner{db: db, providers: providers}
}

func (r *GameAgentModelRunner) RunOneShot(ctx context.Context, tenantID, name, prompt string) (string, error) {
	var row agent.AgentConfig
	if err := r.db.Where("tenant_id=? AND name=?", tenantID, name).First(&row).Error; err != nil {
		return "", err
	}
	model := NewProviderGameActionModel(r.providers, tenantID)
	model.providerID, model.modelID = row.ProviderID, row.ModelID
	model.systemPrompt = row.SystemPrompt + "\n" + row.PersonalityPrompt
	result, err := model.InterpretGameAction(ctx, prompt)
	return string(result), err
}
