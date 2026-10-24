package services

import (
	"encoding/json"
	"testing"

	"control-panel/internal/domain/gameproject"
)

func TestPlayerSceneChoosesEmotionPortraitAndFallsBackToDefault(t *testing.T) {
	draft := GameProjectDraft{
		NPCs: []GameProjectNPC{{ID: "cfo", DisplayName: "莫云岑"}},
		Scenes: []GameProjectScene{{
			ID: "office", Name: "办公室",
			Characters: []GameSceneCharacter{{
				ID: "cfo-layer", CharacterID: "cfo", PortraitAssetID: "portrait-default",
				EmotionPortraitAssetIDs: map[string]string{"angry": "portrait-angry"},
				X:                       50, Y: 0, Scale: 1, Visible: true,
			}},
		}},
	}
	session := gameproject.Session{Source: "draft", ProjectID: "game-1"}

	angry := playerScenes(session, draft, map[string]string{"cfo": "angry"})
	if got := angry[0].Characters[0]; got.Mood != "angry" || got.PortraitURL != "/api/v1/admin/game-projects/game-1/assets/portrait-angry/content" {
		t.Fatalf("愤怒状态应选择对应立绘，got=%#v", got)
	}
	tense := playerScenes(session, draft, map[string]string{"cfo": "tense"})
	if got := tense[0].Characters[0]; got.Mood != "tense" || got.PortraitURL != "/api/v1/admin/game-projects/game-1/assets/portrait-default/content" {
		t.Fatalf("未配置的情绪应继续使用默认立绘，got=%#v", got)
	}
}

func TestPublishedSceneRetainsEmotionPortraitAssetIDs(t *testing.T) {
	manifest := GamePackageManifest{Content: &GamePackageContent{Scenes: []GameSceneDefinition{{
		ID: "office", Name: "办公室", Characters: []GameSceneCharacter{{
			ID: "cfo-layer", CharacterID: "cfo", PortraitAssetID: "portrait-default",
			EmotionPortraitAssetIDs: map[string]string{"calm": "portrait-calm", "angry": "portrait-angry"},
		}},
	}}}}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var restored GamePackageManifest
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	draft := draftFromPackage(restored)
	variants := draft.Scenes[0].Characters[0].EmotionPortraitAssetIDs
	if variants["calm"] != "portrait-calm" || variants["angry"] != "portrait-angry" {
		t.Fatalf("发布快照丢失情绪立绘引用: %#v", variants)
	}
}
