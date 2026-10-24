package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"control-panel/internal/application/services"
	"control-panel/internal/config"
	"control-panel/internal/domain/gameasset"
	"control-panel/pkg/database"
)

type assetSpec struct {
	key, name, kind, path string
}

func main() {
	projectID := flag.String("project", "", "game project ID")
	tenantID := flag.String("tenant", "default", "tenant ID")
	root := flag.String("assets", "assets/speeding-p0", "asset directory")
	flag.Parse()
	if *projectID == "" {
		fatal(fmt.Errorf("project is required"))
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		fatal(err)
	}
	if err := database.InitDatabase(&cfg.Database); err != nil {
		fatal(err)
	}
	defer database.Close()
	assetService := services.NewGameAssetService(database.GetDB(), nil, cfg.Server.GameAssetDir)
	projectService := services.NewGameProjectService(database.GetDB(), services.NewGamePackageService(database.GetDB()))
	detail, err := projectService.Get(*tenantID, *projectID)
	if err != nil {
		fatal(err)
	}
	specs := []assetSpec{
		{"bg-office", "雨夜办公室", gameasset.KindBackground, "backgrounds/crisis-office-rain.png"},
		{"bg-evidence", "证据封存室", gameasset.KindBackground, "backgrounds/evidence-room.png"},
		{"bg-finance", "财务作战室", gameasset.KindBackground, "backgrounds/finance-war-room.png"},
		{"bg-board", "董事会议室", gameasset.KindBackground, "backgrounds/boardroom.png"},
		{"bg-regulatory", "监管会谈室", gameasset.KindBackground, "backgrounds/regulatory-room.png"},
		{"portrait-finance", "莫云岑", gameasset.KindPortrait, "portraits/mo-yuncen.png"},
		{"portrait-legal", "谢衡", gameasset.KindPortrait, "portraits/xie-heng.png"},
		{"portrait-board", "罗重霄", gameasset.KindPortrait, "portraits/luo-chongxiao.png"},
		{"music-office", "雨夜警报", gameasset.KindMusic, "audio/rain-opening.mp3"},
		{"music-evidence", "证据倒计时", gameasset.KindMusic, "audio/investigation.mp3"},
		{"music-finance", "资金路径", gameasset.KindMusic, "audio/investigation-reveal.mp3"},
		{"music-board", "董事会压力", gameasset.KindMusic, "audio/crisis.mp3"},
		{"music-regulatory", "监管会谈", gameasset.KindMusic, "audio/resolution.mp3"},
		{"sfx-alert", "系统警报", gameasset.KindSound, "audio/system-alert.mp3"},
		{"sfx-lock", "封存落锁", gameasset.KindSound, "audio/archive-lock.mp3"},
		{"sfx-document", "放下文件", gameasset.KindSound, "audio/document-place.mp3"},
		{"sfx-door", "会议室关门", gameasset.KindSound, "audio/door-close.mp3"},
		{"sfx-ending", "结局音", gameasset.KindSound, "audio/ending-sting.mp3"},
		{"sfx-phone", "手机震动", gameasset.KindSound, "audio/phone-vibrate.mp3"},
		{"sfx-rain", "窗外暴雨", gameasset.KindSound, "audio/rain-window.mp3"},
		{"sfx-vote", "表决确认", gameasset.KindSound, "audio/vote-confirm.mp3"},
	}
	ids := map[string]string{}
	ctx := context.Background()
	for _, spec := range specs {
		path := filepath.Join(*root, spec.path)
		file, err := os.Open(path)
		if err != nil {
			fatal(err)
		}
		info, err := file.Stat()
		if err != nil {
			file.Close()
			fatal(err)
		}
		asset, err := assetService.Upload(ctx, *tenantID, *projectID, "install-speeding-p0-assets", spec.name, spec.kind, filepath.Base(path), file, info.Size())
		file.Close()
		if err != nil {
			fatal(fmt.Errorf("upload %s: %w", spec.name, err))
		}
		ids[spec.key] = asset.ID
	}
	draft := detail.Draft
	for i := range draft.Scenes {
		scene := &draft.Scenes[i]
		switch scene.ID {
		case "crisis_office":
			scene.BackgroundAssetID, scene.MusicAssetID = ids["bg-office"], ids["music-office"]
			scene.SoundEffectAssetIDs = []string{ids["sfx-rain"], ids["sfx-alert"]}
		case "evidence_room":
			scene.BackgroundAssetID, scene.MusicAssetID = ids["bg-evidence"], ids["music-evidence"]
			scene.SoundEffectAssetIDs = []string{ids["sfx-lock"]}
		case "finance_war_room":
			scene.BackgroundAssetID, scene.MusicAssetID = ids["bg-finance"], ids["music-finance"]
			scene.SoundEffectAssetIDs = []string{ids["sfx-document"], ids["sfx-phone"]}
		case "boardroom":
			scene.BackgroundAssetID, scene.MusicAssetID = ids["bg-board"], ids["music-board"]
			scene.SoundEffectAssetIDs = []string{ids["sfx-door"], ids["sfx-vote"]}
		case "regulatory_room":
			scene.BackgroundAssetID, scene.MusicAssetID = ids["bg-regulatory"], ids["music-regulatory"]
			scene.SoundEffectAssetIDs = []string{ids["sfx-ending"]}
		}
		for j := range scene.Characters {
			layer := &scene.Characters[j]
			switch layer.CharacterID {
			case "finance":
				layer.PortraitAssetID = ids["portrait-finance"]
			case "legal":
				layer.PortraitAssetID = ids["portrait-legal"]
			case "board_rival":
				layer.PortraitAssetID = ids["portrait-board"]
			}
		}
	}
	_, err = projectService.Update(*tenantID, *projectID, "install-speeding-p0-assets", services.UpdateGameProjectInput{Revision: detail.Project.Revision, Name: detail.Project.Name, Description: detail.Project.Description, Genre: detail.Project.Genre, PlayGoal: detail.Project.PlayGoal, Draft: &draft})
	if err != nil {
		fatal(err)
	}
	fmt.Printf("已为 %s 安装 %d 个资源并绑定到五幕场景\n", *projectID, len(specs))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "install Speeding P0 assets:", err)
	os.Exit(1)
}
