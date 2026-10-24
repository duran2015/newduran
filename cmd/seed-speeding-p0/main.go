package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"control-panel/internal/application/services"
	"control-panel/internal/config"
	"control-panel/pkg/database"
)

func main() {
	tenant := flag.String("tenant", "default", "tenant to seed")
	dryRun := flag.Bool("dry-run", false, "validate and print planned changes without writing")
	updateExisting := flag.Bool("update-existing", false, "refresh an existing demo while preserving its uploaded scene media")
	projectKey := flag.String("project-key", services.SpeedingP0LocalProjectKey, "project key")
	projectName := flag.String("project-name", "Speeding：雨夜决议", "display name for the new project")
	director := flag.String("director", "director", "existing director Agent name")
	finance := flag.String("finance", "cfo", "existing finance Agent name")
	legal := flag.String("legal", "counsel", "existing legal Agent name")
	boardRival := flag.String("board-rival", "board-rival", "board rival Agent name (created if missing)")
	flag.Parse()
	cfg, err := config.LoadConfig()
	if err != nil {
		fatal(err)
	}
	if err := database.InitDatabase(&cfg.Database); err != nil {
		fatal(err)
	}
	defer database.Close()
	packages := services.NewGamePackageService(database.GetDB())
	result, err := services.SeedSpeedingP0Local(database.GetDB(), services.NewGameProjectService(database.GetDB(), packages), services.NewAgentService(cfg.Provider.EncryptionKey, ""), services.SpeedingP0SeedOptions{
		TenantID: *tenant, Actor: "seed-speeding-p0", DryRun: *dryRun, UpdateExisting: *updateExisting, ProjectKey: *projectKey, ProjectName: *projectName,
		AgentNames: map[string]string{"director": *director, "finance": *finance, "legal": *legal, "board_rival": *boardRival},
	})
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
}

func fatal(err error) { fmt.Fprintln(os.Stderr, "seed Speeding P0:", err); os.Exit(1) }
