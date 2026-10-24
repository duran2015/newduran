package services

import (
	"encoding/json"
	"testing"
)

func TestPublishedEventRetainsSceneAndEndingForPlayer(t *testing.T) {
	manifest := GamePackageManifest{Content: &GamePackageContent{StoryEvents: []GameStoryEvent{{ID: "hearing", Name: "听证会", SceneID: "court", EndsGame: true, EndReason: "调查完成"}}}}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var restored GamePackageManifest
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	draft := draftFromPackage(restored)
	if len(draft.Events) != 1 || draft.Events[0].SceneID != "court" || draft.Events[0].EndReason != "调查完成" || !draft.Events[0].EndsGame {
		t.Fatalf("published event lost presentation or ending: %#v", draft.Events)
	}
}
