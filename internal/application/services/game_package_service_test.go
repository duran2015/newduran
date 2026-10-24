package services

import (
	"encoding/json"
	"strings"
	"testing"

	"control-panel/internal/domain/gamepackage"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func gamePackageTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&gamepackage.Package{}))
	return db
}

func counterManifest(version string) json.RawMessage {
	return json.RawMessage(`{
      "schemaVersion":"game.zerone.io/v1",
      "package":{"key":"io.example.counter","version":"` + version + `","displayName":"Counter"},
      "registry":{
        "metrics":[{"id":"count","name":"Count","type":"integer","initial":0,"minimum":0}],
        "actions":[{"id":"increment","name":"Increment","exampleInput":{"amount":2},"effects":[{"operation":"increment","target":"count","valueFrom":"amount","defaultValue":1}],"emits":"counter.changed"}],
        "events":[{"id":"counter.changed","name":"Counter changed"}],
        "endings":[]
      },
      "acceptanceScenarios":[{"id":"basic.increment","commands":[{"actionId":"increment","input":{"amount":2}}]}]
    }`)
}

func TestGamePackageRegisterVersionIsolationAndTestRun(t *testing.T) {
	svc := NewGamePackageService(gamePackageTestDB(t))
	v1, err := svc.Register("tenant-a", "tester", counterManifest("1.0.0"))
	require.NoError(t, err)
	v11, err := svc.Register("tenant-a", "tester", counterManifest("1.1.0"))
	require.NoError(t, err)
	require.NotEqual(t, v1.Package.ID, v11.Package.ID)

	_, err = svc.SetEnabled("tenant-a", v1.Package.ID, true)
	require.NoError(t, err)
	result, err := svc.TestRun("tenant-a", v1.Package.ID, GameTestRunInput{Commands: []GameTestCommand{{ActionID: "increment", Input: map[string]any{"amount": float64(3)}}}})
	require.NoError(t, err)
	require.EqualValues(t, 3, result.FinalState["count"])
	require.Equal(t, "counter.changed", result.Steps[0].Event.ID)
	require.Len(t, result.Steps[0].StateDiff, 1)

	list, err := svc.List("tenant-a", nil)
	require.NoError(t, err)
	require.Equal(t, 2, list.Total)
	other, err := svc.List("tenant-b", nil)
	require.NoError(t, err)
	require.Zero(t, other.Total)
}

func TestGamePackageRegistrationIsContentAddressed(t *testing.T) {
	svc := NewGamePackageService(gamePackageTestDB(t))
	first, err := svc.Register("tenant-a", "tester", counterManifest("1.0.0"))
	require.NoError(t, err)
	second, err := svc.Register("tenant-a", "tester", counterManifest("1.0.0"))
	require.NoError(t, err)
	require.Equal(t, first.Package.ID, second.Package.ID)

	changed := json.RawMessage(strings.Replace(string(counterManifest("1.0.0")), `"displayName":"Counter"`, `"displayName":"Changed"`, 1))
	_, err = svc.Register("tenant-a", "tester", changed)
	require.Error(t, err)
}

func TestGamePackageTestRunEnforcesPublishedActionConditions(t *testing.T) {
	svc := NewGamePackageService(gamePackageTestDB(t))
	raw := json.RawMessage(strings.Replace(string(counterManifest("1.0.0")),
		`"name":"Increment","exampleInput"`,
		`"name":"Increment","conditions":[{"target":"count","operator":"gte","value":2}],"exampleInput"`, 1))
	pkg, err := svc.Register("tenant-a", "tester", raw)
	require.NoError(t, err)
	_, err = svc.SetEnabled("tenant-a", pkg.Package.ID, true)
	require.NoError(t, err)
	_, err = svc.TestRun("tenant-a", pkg.Package.ID, GameTestRunInput{Commands: []GameTestCommand{{ActionID: "increment"}}})
	require.ErrorContains(t, err, "does not meet its conditions")
	result, err := svc.TestRun("tenant-a", pkg.Package.ID, GameTestRunInput{InitialState: map[string]any{"count": float64(2)}, Commands: []GameTestCommand{{ActionID: "increment"}}})
	require.NoError(t, err)
	require.EqualValues(t, 3, result.FinalState["count"])
}

func TestGamePackageKeepsCreatorDisplayUnitAndTimeCost(t *testing.T) {
	svc := NewGamePackageService(gamePackageTestDB(t))
	raw := json.RawMessage(strings.NewReplacer(
		`"name":"Count","type"`, `"name":"Count","displayUnit":"亿元","type"`,
		`"name":"Increment","exampleInput"`, `"name":"Increment","timeCost":2,"exampleInput"`,
	).Replace(string(counterManifest("1.0.0"))))
	pkg, err := svc.Register("tenant-a", "tester", raw)
	require.NoError(t, err)
	require.Equal(t, "亿元", pkg.Manifest.Registry.Metrics[0].DisplayUnit)
	require.Equal(t, 2, pkg.Manifest.Registry.Actions[0].TimeCost)

	negative := json.RawMessage(strings.Replace(string(raw), `"timeCost":2`, `"timeCost":-1`, 1))
	_, err = svc.Register("tenant-a", "tester", negative)
	require.ErrorContains(t, err, "timeCost cannot be negative")
}
