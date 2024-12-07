package ui

import (
	"TodoList/internal/config"
	"TodoList/internal/storage"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

type MainWindow struct {
	window        fyne.Window
	timerManager  *TimerManager
	todo          *TodoList
	db            *storage.Database
	configManager *config.Manager
}

func NewMainWindow(app fyne.App, configManager *config.Manager) *MainWindow {
	db, err := storage.NewDatabase()
	if err != nil {
		return nil
	}

	w := &MainWindow{
		window:        app.NewWindow("番茄钟 + 待办事项"),
		configManager: configManager,
		db:            db,
		timerManager:  NewTimerManager(),
		todo:          NewTodoList(db),
	}
	w.setup()
	return w
}

func (w *MainWindow) SetSize(width, height float32) {
	w.window.Resize(fyne.NewSize(width, height))
}

func (w *MainWindow) setup() {
	stats := NewStatsView(w.db)

	tabs := container.NewAppTabs(
		container.NewTabItem("番茄钟", w.timerManager.container),
		container.NewTabItem("待办事项", w.todo.container),
		container.NewTabItem("统计", stats.container),
	)

	w.window.SetContent(tabs)
	w.window.Resize(fyne.NewSize(400, 500))
}

func (w *MainWindow) Show() {
	w.window.ShowAndRun()
}
