package main

import (
	"TodoList/internal/config"
	"TodoList/internal/ui"
	"fyne.io/fyne/v2/app"
	"log"
)

func main() {
	// 初始化配置管理器
	configManager, err := config.NewManager()
	if err != nil {
		log.Fatal(err)
	}

	// 创建应用
	myApp := app.New()

	// 应用主题设置
	cfg := configManager.GetConfig()
	if cfg.Theme.DarkMode {
		// 设置深色主题
	}

	// 创建主窗口
	mainWindow := ui.NewMainWindow(myApp, configManager)

	// 设置窗口大小
	mainWindow.SetSize(float32(cfg.App.WindowWidth), float32(cfg.App.WindowHeight))

	mainWindow.Show()
}
