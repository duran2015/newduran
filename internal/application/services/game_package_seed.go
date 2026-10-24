package services

import (
	"embed"
	"encoding/json"
	"fmt"

	"control-panel/internal/domain/gamepackage"
)

// m0AcceptancePackageFiles are product-readable acceptance fixtures, not
// Kernel defaults. Keeping them as ordinary manifests makes the acceptance
// environment exercise the exact same registration path as future games.
//
//go:embed testdata/gamepackages/*.json
var m0AcceptancePackageFiles embed.FS

type m0AcceptancePackageSeed struct {
	filename       string
	defaultEnabled bool
}

var m0AcceptancePackageSeeds = []m0AcceptancePackageSeed{
	{filename: "counter-v1.json", defaultEnabled: true},
	{filename: "restaurant-v1.json", defaultEnabled: true},
	{filename: "restaurant-morale-v1.1.json", defaultEnabled: false},
}

// EnsureM0AcceptancePackages installs the three M0 acceptance manifests for a
// single tenant. Defaults are applied only on first installation: subsequent
// startups never undo a product manager's enable/disable decision.
func (s *GamePackageService) EnsureM0AcceptancePackages(tenantID string) error {
	for _, seed := range m0AcceptancePackageSeeds {
		path := "testdata/gamepackages/" + seed.filename
		raw, err := m0AcceptancePackageFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read M0 acceptance package %s: %w", seed.filename, err)
		}
		var manifest GamePackageManifest
		if err := json.Unmarshal(raw, &manifest); err != nil {
			return fmt.Errorf("decode M0 acceptance package %s: %w", seed.filename, err)
		}

		var existing int64
		if err := s.db.Model(&gamepackage.Package{}).
			Where("tenant_id=? AND `key`=? AND version=?", tenantID, manifest.Package.Key, manifest.Package.Version).
			Count(&existing).Error; err != nil {
			return err
		}
		detail, err := s.Register(tenantID, "m0-acceptance-seed", json.RawMessage(raw))
		if err != nil {
			return fmt.Errorf("register M0 acceptance package %s: %w", seed.filename, err)
		}
		if existing == 0 && seed.defaultEnabled {
			if _, err := s.SetEnabled(tenantID, detail.Package.ID, true); err != nil {
				return fmt.Errorf("enable M0 acceptance package %s: %w", seed.filename, err)
			}
		}
	}
	return nil
}
