package services

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"control-panel/internal/domain/agent"
	"control-panel/internal/domain/gameproject"
	"gorm.io/gorm"
)

const SpeedingP0LocalProjectKey = "speeding-p0-local"

//go:embed testdata/gameprojects/speeding-p0-draft.json
var speedingP0SeedFiles embed.FS

type SpeedingP0SeedOptions struct {
	TenantID       string
	Actor          string
	DryRun         bool
	UpdateExisting bool
	ProjectKey     string
	ProjectName    string
	AgentNames     map[string]string
}

type SpeedingP0SeedResult struct {
	DryRun         bool              `json:"dryRun"`
	ProjectID      string            `json:"projectId,omitempty"`
	ProjectKey     string            `json:"projectKey"`
	ProjectCreated bool              `json:"projectCreated"`
	ProjectUpdated bool              `json:"projectUpdated"`
	AlreadyExists  bool              `json:"alreadyExists"`
	CreatedAgents  []string          `json:"createdAgents"`
	AgentIDs       map[string]uint64 `json:"agentIds"`
	PendingAssets  []string          `json:"pendingAssets"`
}

// SeedSpeedingP0Local imports the checked-in P0 fixture through the normal
// project service. Existing projects are left untouched unless UpdateExisting
// is explicitly enabled; an update preserves all uploaded scene media.
func SeedSpeedingP0Local(db *gorm.DB, projects *GameProjectService, agents *AgentService, opts SpeedingP0SeedOptions) (*SpeedingP0SeedResult, error) {
	tenantID := strings.TrimSpace(opts.TenantID)
	if tenantID == "" {
		tenantID = "default"
	}
	actor := strings.TrimSpace(opts.Actor)
	if actor == "" {
		actor = "local-seed"
	}
	projectKey := strings.TrimSpace(opts.ProjectKey)
	if projectKey == "" {
		projectKey = SpeedingP0LocalProjectKey
	}
	projectName := strings.TrimSpace(opts.ProjectName)
	if projectName == "" {
		projectName = "Speeding：雨夜决议"
	}
	names := map[string]string{"director": "director", "finance": "cfo", "legal": "counsel", "board_rival": "board-rival"}
	for role, name := range opts.AgentNames {
		if strings.TrimSpace(name) != "" {
			names[role] = strings.TrimSpace(name)
		}
	}

	result := &SpeedingP0SeedResult{DryRun: opts.DryRun, ProjectKey: projectKey, AgentIDs: map[string]uint64{}, CreatedAgents: []string{}, PendingAssets: []string{}}
	var existing gameproject.Project
	err := db.Where("tenant_id=? AND `key`=?", tenantID, projectKey).First(&existing).Error
	existingFound := err == nil
	if existingFound && !opts.UpdateExisting {
		result.ProjectID, result.AlreadyExists = existing.ID, true
		return result, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("check Speeding P0 project: %w", err)
	}

	for _, role := range []string{"director", "finance", "legal", "board_rival"} {
		var row agent.AgentConfig
		err := db.Where("tenant_id=? AND name=?", tenantID, names[role]).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) && role == "board_rival" {
			if opts.DryRun {
				result.CreatedAgents = append(result.CreatedAgents, names[role]+" (planned)")
				continue
			}
			created, createErr := agents.CreateAgent(tenantID, &CreateAgentInput{Name: names[role], Config: map[string]interface{}{
				"systemPrompt": "You are Luo Chongxiao, a board rival protecting governance continuity. Challenge unsupported claims, negotiate conditions, and never decide for the player.",
			}})
			if createErr != nil {
				return nil, fmt.Errorf("create board rival Agent %q: %w", names[role], createErr)
			}
			row.ID = created.ID
			result.CreatedAgents = append(result.CreatedAgents, names[role])
		} else if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("required %s Agent %q does not exist", role, names[role])
			}
			return nil, fmt.Errorf("find %s Agent %q: %w", role, names[role], err)
		}
		result.AgentIDs[role] = row.ID
	}

	raw, err := speedingP0SeedFiles.ReadFile("testdata/gameprojects/speeding-p0-draft.json")
	if err != nil {
		return nil, err
	}
	var draft GameProjectDraft
	if err := json.Unmarshal(raw, &draft); err != nil {
		return nil, fmt.Errorf("decode Speeding P0 fixture: %w", err)
	}
	for i := range draft.NPCs {
		id, ok := result.AgentIDs[draft.NPCs[i].ID]
		if !ok && !opts.DryRun {
			return nil, fmt.Errorf("fixture character %q has no Agent mapping", draft.NPCs[i].ID)
		}
		draft.NPCs[i].AgentID = id
	}
	for i := range draft.Scenes {
		collect := func(ref string) {
			if strings.HasPrefix(ref, "asset://") {
				result.PendingAssets = append(result.PendingAssets, ref)
			}
		}
		collect(draft.Scenes[i].BackgroundRef)
		collect(draft.Scenes[i].MusicRef)
		for _, ref := range draft.Scenes[i].PortraitRefs {
			collect(ref)
		}
		for _, ref := range draft.Scenes[i].SoundEffectRefs {
			collect(ref)
		}
		// asset:// values are authored slots, not uploaded resources. Never save
		// them as if they were usable media references.
		draft.Scenes[i].BackgroundRef = ""
		draft.Scenes[i].MusicRef = ""
		draft.Scenes[i].PortraitRefs = nil
		draft.Scenes[i].SoundEffectRefs = nil
	}
	sort.Strings(result.PendingAssets)
	result.PendingAssets = compactSeedStrings(result.PendingAssets)
	if opts.DryRun {
		if existingFound {
			result.ProjectID, result.AlreadyExists = existing.ID, true
		}
		return result, nil
	}
	if existingFound {
		current, getErr := projects.Get(tenantID, existing.ID)
		if getErr != nil {
			return nil, fmt.Errorf("load existing Speeding P0 project: %w", getErr)
		}
		preserveSpeedingP0SceneMedia(&draft, current.Draft)
		updated, updateErr := projects.Update(tenantID, existing.ID, actor, UpdateGameProjectInput{
			Revision: current.Project.Revision, Name: current.Project.Name, Description: current.Project.Description,
			Genre: current.Project.Genre, PlayGoal: current.Project.PlayGoal, Draft: &draft,
		})
		if updateErr != nil {
			return nil, fmt.Errorf("update Speeding P0 draft: %w", updateErr)
		}
		result.ProjectID, result.ProjectUpdated, result.AlreadyExists = updated.Project.ID, true, true
		return result, nil
	}
	project, err := projects.Create(tenantID, actor, CreateGameProjectInput{
		Key: projectKey, Name: projectName, Description: "一场从雨夜持续到清晨的公司危机。你必须在证据消失、董事会夺权和监管追问之间作出选择。", Genre: "现代商战 / 对话策略", PlayGoal: "在五幕危机中调查异常付款，与三名关键人物周旋，并承担自己选择带来的结局。",
	})
	if err != nil {
		return nil, fmt.Errorf("create Speeding P0 project: %w", err)
	}
	updated, err := projects.Update(tenantID, project.Project.ID, actor, UpdateGameProjectInput{Revision: project.Project.Revision, Name: project.Project.Name, Description: project.Project.Description, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft})
	if err != nil {
		return nil, fmt.Errorf("import Speeding P0 draft: %w", err)
	}
	result.ProjectID, result.ProjectCreated = updated.Project.ID, true
	return result, nil
}

func preserveSpeedingP0SceneMedia(next *GameProjectDraft, current GameProjectDraft) {
	byID := make(map[string]GameProjectScene, len(current.Scenes))
	for _, scene := range current.Scenes {
		byID[scene.ID] = scene
	}
	for i := range next.Scenes {
		old, ok := byID[next.Scenes[i].ID]
		if !ok {
			continue
		}
		next.Scenes[i].BackgroundAssetID = old.BackgroundAssetID
		next.Scenes[i].MusicAssetID = old.MusicAssetID
		next.Scenes[i].SoundEffectAssetIDs = append([]string{}, old.SoundEffectAssetIDs...)
		oldCharacters := make(map[string]GameSceneCharacter, len(old.Characters))
		for _, character := range old.Characters {
			oldCharacters[character.CharacterID] = character
		}
		for j := range next.Scenes[i].Characters {
			oldCharacter, found := oldCharacters[next.Scenes[i].Characters[j].CharacterID]
			if !found {
				continue
			}
			next.Scenes[i].Characters[j].PortraitAssetID = oldCharacter.PortraitAssetID
			next.Scenes[i].Characters[j].EmotionPortraitAssetIDs = oldCharacter.EmotionPortraitAssetIDs
		}
	}
}

func compactSeedStrings(values []string) []string {
	out := values[:0]
	for _, value := range values {
		if len(out) == 0 || out[len(out)-1] != value {
			out = append(out, value)
		}
	}
	return out
}
