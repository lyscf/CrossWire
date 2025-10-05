package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"crosswire/internal/app"
	"crosswire/internal/storage"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// 获取用户数据目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	dataDir := filepath.Join(homeDir, ".crosswire")

	// 初始化数据库
	db, err := storage.NewDatabase(&storage.Config{
		DataDir:   dataDir,
		DebugMode: false,
	})
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// 创建应用实例
	application := app.NewApp(db)

	// 创建 Wails 应用
	err = wails.Run(&options.App{
		Title:  "CrossWire - CTF Team Communication",
		Width:  1280,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        application.Startup,
		OnDomReady:       application.DomReady,
		OnShutdown:       application.Shutdown,
		Bind: []interface{}{
			application,
		},
	})

	if err != nil {
		fmt.Println("Error:", err.Error())
	}
}
