package services

import (
	"embed"
	"encoding/json"
	"testing"

	"control-panel/internal/domain/gamepackage"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

//go:embed testdata/gamepackages/*.json
var m0GamePackageFixtures embed.FS

func newM0GamePackageService(t *testing.T) *GamePackageService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&gamepackage.Package{}))
	return NewGamePackageService(db)
}

func registerM0Fixture(t *testing.T, service *GamePackageService, tenantID, filename string) *GamePackageDetail {
	t.Helper()
	raw, err := m0GamePackageFixtures.ReadFile("testdata/gamepackages/" + filename)
	require.NoError(t, err)
	detail, err := service.Register(tenantID, "m0-acceptance", json.RawMessage(raw))
	require.NoError(t, err)
	return detail
}

func enableM0Package(t *testing.T, service *GamePackageService, tenantID, id string) {
	t.Helper()
	result, err := service.SetEnabled(tenantID, id, true)
	require.NoError(t, err)
	require.True(t, result.Enabled)
}

func manifestMetricIDs(manifest GamePackageManifest) []string {
	ids := make([]string, 0, len(manifest.Registry.Metrics))
	for _, metric := range manifest.Registry.Metrics {
		ids = append(ids, metric.ID)
	}
	return ids
}

// M0-01 proves that the same Kernel accepts two unrelated packages and keeps
// their registries and state isolated. There is deliberately no Speeding data.
func TestM0Acceptance01TwoDifferentPackages(t *testing.T) {
	service := newM0GamePackageService(t)
	tenantID := "m0-product"
	counter := registerM0Fixture(t, service, tenantID, "counter-v1.json")
	restaurant := registerM0Fixture(t, service, tenantID, "restaurant-v1.json")
	enableM0Package(t, service, tenantID, counter.Package.ID)
	enableM0Package(t, service, tenantID, restaurant.Package.ID)

	require.Equal(t, []string{"count"}, manifestMetricIDs(counter.Manifest))
	require.ElementsMatch(t, []string{"inventory", "cash", "operating"}, manifestMetricIDs(restaurant.Manifest))
	require.NotContains(t, string(marshalTestJSON(t, counter.Manifest)), "speeding")
	require.NotContains(t, string(marshalTestJSON(t, restaurant.Manifest)), "speeding")

	counterRun, err := service.TestRun(tenantID, counter.Package.ID, GameTestRunInput{Commands: []GameTestCommand{
		{ActionID: "increase", Input: map[string]any{"amount": 3}},
		{ActionID: "decrease"},
	}})
	require.NoError(t, err)
	require.Equal(t, float64(2), numericState(t, counterRun.FinalState, "count"))
	require.Len(t, counterRun.Steps, 2)
	require.Equal(t, "counter.increased", counterRun.Steps[0].Event.ID)

	restaurantRun, err := service.TestRun(tenantID, restaurant.Package.ID, GameTestRunInput{Commands: []GameTestCommand{
		{ActionID: "purchase"},
		{ActionID: "wait"},
		{ActionID: "supply_interruption"},
		{ActionID: "close"},
	}})
	require.NoError(t, err)
	require.Equal(t, float64(80), numericState(t, restaurantRun.FinalState, "cash"))
	require.Equal(t, float64(5), numericState(t, restaurantRun.FinalState, "inventory"))
	require.Equal(t, false, restaurantRun.FinalState["operating"])
	require.Equal(t, "restaurant.closed", restaurantRun.Steps[3].Event.ID)
	require.NotContains(t, restaurantRun.FinalState, "count")
	require.NotContains(t, counterRun.FinalState, "inventory")
}

// M0-02 proves that activation is tenant/package-version local. Disabling the
// restaurant blocks only new restaurant test runs; the package and its exact
// immutable version remain inspectable, and the counter continues to run.
func TestM0Acceptance02DisableAndReenableIsolation(t *testing.T) {
	service := newM0GamePackageService(t)
	tenantID := "m0-product"
	counter := registerM0Fixture(t, service, tenantID, "counter-v1.json")
	restaurant := registerM0Fixture(t, service, tenantID, "restaurant-v1.json")
	enableM0Package(t, service, tenantID, counter.Package.ID)
	enableM0Package(t, service, tenantID, restaurant.Package.ID)

	disabled, err := service.SetEnabled(tenantID, restaurant.Package.ID, false)
	require.NoError(t, err)
	require.False(t, disabled.Enabled)
	_, err = service.TestRun(tenantID, restaurant.Package.ID, GameTestRunInput{Commands: []GameTestCommand{{ActionID: "wait"}}})
	require.ErrorContains(t, err, "disabled")

	counterRun, err := service.TestRun(tenantID, counter.Package.ID, GameTestRunInput{Commands: []GameTestCommand{{ActionID: "increase"}}})
	require.NoError(t, err)
	require.Equal(t, float64(1), numericState(t, counterRun.FinalState, "count"))

	inspectable, err := service.Get(tenantID, restaurant.Package.ID)
	require.NoError(t, err)
	require.Equal(t, "1.0.0", inspectable.Package.Version)
	require.Equal(t, restaurant.Package.ContentHash, inspectable.Package.ContentHash)
	enableM0Package(t, service, tenantID, restaurant.Package.ID)
	restored, err := service.TestRun(tenantID, restaurant.Package.ID, GameTestRunInput{Commands: []GameTestCommand{{ActionID: "wait"}}})
	require.NoError(t, err)
	require.Equal(t, float64(4), numericState(t, restored.FinalState, "inventory"))
}

// M0-03 proves the extension point with data, not a Kernel branch. The new
// package version declares morale, its action and event using existing metric
// and effect types, while the original version and counter stay unchanged.
func TestM0Acceptance03RestaurantMoraleExtensionWithoutKernelChange(t *testing.T) {
	service := newM0GamePackageService(t)
	tenantID := "m0-product"
	counter := registerM0Fixture(t, service, tenantID, "counter-v1.json")
	restaurantV1 := registerM0Fixture(t, service, tenantID, "restaurant-v1.json")
	restaurantV11 := registerM0Fixture(t, service, tenantID, "restaurant-morale-v1.1.json")

	require.Equal(t, restaurantV1.Package.Key, restaurantV11.Package.Key)
	require.Equal(t, "1.0.0", restaurantV1.Package.Version)
	require.Equal(t, "1.1.0", restaurantV11.Package.Version)
	require.NotContains(t, manifestMetricIDs(restaurantV1.Manifest), "employee_morale")
	require.Contains(t, manifestMetricIDs(restaurantV11.Manifest), "employee_morale")
	require.Equal(t, []string{"count"}, manifestMetricIDs(counter.Manifest))

	enableM0Package(t, service, tenantID, restaurantV11.Package.ID)
	run, err := service.TestRun(tenantID, restaurantV11.Package.ID, GameTestRunInput{Commands: []GameTestCommand{{ActionID: "team_building"}}})
	require.NoError(t, err)
	require.Equal(t, float64(90), numericState(t, run.FinalState, "cash"))
	require.Equal(t, float64(75), numericState(t, run.FinalState, "employee_morale"))
	require.Equal(t, "restaurant.morale_improved", run.Steps[0].Event.ID)
	require.ElementsMatch(t, []string{"cash", "employee_morale"}, []string{
		run.Steps[0].StateDiff[0].MetricID,
		run.Steps[0].StateDiff[1].MetricID,
	})

	list, err := service.List(tenantID, nil)
	require.NoError(t, err)
	require.Len(t, list.Items, 3)
}

func TestEnsureM0AcceptancePackagesIsIdempotentAndPreservesManualState(t *testing.T) {
	service := newM0GamePackageService(t)
	tenantID := "m0-seed"
	require.NoError(t, service.EnsureM0AcceptancePackages(tenantID))

	list, err := service.List(tenantID, nil)
	require.NoError(t, err)
	require.Len(t, list.Items, 3)
	byVersion := map[string]GamePackageListItem{}
	for _, item := range list.Items {
		byVersion[item.Key+"@"+item.Version] = item
	}
	require.True(t, byVersion["io.example.counter@1.0.0"].Enabled)
	require.True(t, byVersion["io.example.restaurant@1.0.0"].Enabled)
	require.False(t, byVersion["io.example.restaurant@1.1.0"].Enabled)

	// Product changes are authoritative after the initial installation.
	_, err = service.SetEnabled(tenantID, byVersion["io.example.counter@1.0.0"].ID, false)
	require.NoError(t, err)
	_, err = service.SetEnabled(tenantID, byVersion["io.example.restaurant@1.1.0"].ID, true)
	require.NoError(t, err)
	require.NoError(t, service.EnsureM0AcceptancePackages(tenantID))

	list, err = service.List(tenantID, nil)
	require.NoError(t, err)
	require.Len(t, list.Items, 3)
	for _, item := range list.Items {
		switch item.Key + "@" + item.Version {
		case "io.example.counter@1.0.0":
			require.False(t, item.Enabled)
		case "io.example.restaurant@1.1.0":
			require.True(t, item.Enabled)
		}
	}

	otherTenant, err := service.List("unseeded-tenant", nil)
	require.NoError(t, err)
	require.Empty(t, otherTenant.Items)
}

func marshalTestJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	return raw
}

func numericState(t *testing.T, state map[string]any, key string) float64 {
	t.Helper()
	number, ok := gameNumber(state[key])
	require.True(t, ok, "state %s should be numeric, got %#v", key, state[key])
	return number
}
