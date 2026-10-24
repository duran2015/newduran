package services

import (
	"context"
	"encoding/json"
	"time"

	rundomain "control-panel/internal/domain/run"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *GamePlayerSessionService) SetDirectorBridge(bridge *GameDirectorBridge) { s.director = bridge }

// Invoked only after the first successful plan commit. Retries of the same
// plan never repeat model calls or game effects. Models run outside DB locks.
func (s *GamePlayerSessionService) participateAfterConfirmation(in GameDirectorTurnInput) {
	if s.director == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	_, _ = s.runs.AppendActivity(in.TenantID, in.RunID, AppendRunActivityInput{Kind: "game.characters.started", Status: "running", Name: "人物正在回应", Output: map[string]any{"summary": "行动已结算，导演正在安排人物回应。"}})
	result, err := s.director.Participate(ctx, in)
	if err != nil {
		_, _ = s.runs.AppendActivity(in.TenantID, in.RunID, AppendRunActivityInput{Kind: "game.director.failed", Status: "failed", Name: "人物暂时无法回应", Output: map[string]any{"summary": "行动已结算，但人物本轮未能回应。可以继续游戏。"}, Error: err.Error()})
		return
	}
	s.activateOfferedObjectives(in, result)
	_, _ = s.runs.AppendActivity(in.TenantID, in.RunID, AppendRunActivityInput{Kind: "game.characters.completed", Status: "completed", Name: "本轮人物回应结束", Output: map[string]any{"summary": "本轮人物回应已记录。"}})
}

func (s *GamePlayerSessionService) activateOfferedObjectives(in GameDirectorTurnInput, result *GameDirectorTurnResult) {
	if result == nil {
		return
	}
	for _, response := range result.Responses {
		if response.ObjectiveID == "" {
			continue
		}
		_ = s.db.Transaction(func(tx *gorm.DB) error {
			var state rundomain.RunState
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id=? AND run_id=? AND namespace=?", in.TenantID, in.RunID, gameSessionNamespace).First(&state).Error; err != nil {
				return err
			}
			statuses := stringMap(state.Data["objectiveStatuses"])
			if statuses[response.ObjectiveID] != "locked" && statuses[response.ObjectiveID] != "rejected" {
				return nil
			}
			statuses[response.ObjectiveID] = "offered"
			next := normalizeJSONMap(state.Data)
			next["objectiveStatuses"] = statuses
			encoded, err := json.Marshal(next)
			if err != nil {
				return err
			}
			now := time.Now().UTC()
			revision := state.Revision + 1
			res := tx.Model(&rundomain.RunState{}).Where("tenant_id=? AND id=? AND revision=?", in.TenantID, state.ID, state.Revision).Updates(map[string]any{"data": string(encoded), "revision": revision, "updated_at": now})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected != 1 {
				return rundomain.ErrConflict
			}
			var objective *GameObjectiveDefinition
			for i := range in.Draft.Objectives {
				if in.Draft.Objectives[i].ID == response.ObjectiveID {
					objective = &in.Draft.Objectives[i]
					break
				}
			}
			output := map[string]any{"summary": response.DisplayName + "向你交付了新任务，等待你接受或拒绝。", "objectiveId": response.ObjectiveID, "deliveryStatus": "offered"}
			if objective != nil {
				output["objectiveName"] = objective.Name
				if objective.PromiseText != "" {
					output["promise"] = objective.PromiseText
				}
			}
			return tx.Create(&rundomain.RunActivity{ID: uuid.NewString(), TenantID: in.TenantID, RunID: in.RunID, Kind: "game.objective.offered", Status: "completed", ActorType: "agent", ActorID: response.CharacterID, Name: "收到任务邀请", Output: output, OccurredAt: now}).Error
		})
	}
}
