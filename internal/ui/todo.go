package ui

import (
	"TodoList/internal/models"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"image/color"
	"sort"
	"time"

	"TodoList/internal/storage"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type TaskStatus string

const (
	StatusTodo  TaskStatus = "TODO"
	StatusDoing TaskStatus = "DOING"
	StatusDone  TaskStatus = "DONE"
	StatusUndo  TaskStatus = "UNDO"
)

// TodoItem 表示单个待办事项
type TodoItem struct {
	task      *models.Task
	check     *widget.Check
	label     *canvas.Text
	editBtn   *widget.Button
	parent    *TodoList
	container *fyne.Container
}

// 创建新的待办事项
func NewTodoItem(task *models.Task, parent *TodoList) *TodoItem {
	item := &TodoItem{
		task:   task,
		parent: parent,
	}

	// 创建勾选按钮
	checkBtn := widget.NewButtonWithIcon("", theme.ConfirmIcon(), func() {
		currentStatus := TaskStatus(item.task.Status)
		switch currentStatus {
		case StatusTodo:
			item.parent.moveTask(item.task, StatusDoing)
		case StatusDoing:
			item.parent.moveTask(item.task, StatusDone)
		case StatusUndo:
			item.parent.moveTask(item.task, StatusTodo)
		}
	})
	checkBtn.Resize(fyne.NewSize(10, 10))

	title := canvas.NewText(task.Title, color.Black)
	title.TextStyle = fyne.TextStyle{Italic: true}
	title.Alignment = fyne.TextAlignCenter
	title.Resize(fyne.NewSize(200, 0))

	// 创建删除按钮，根据状态设置不同的行为
	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		if TaskStatus(item.task.Status) == StatusUndo {
			// 如果是 Undo 状态，直接删除任务
			if err := parent.db.DeleteTask(item.task.ID); err != nil {
				fmt.Println("Error deleting task:", err)
				return
			}
			// 从内存中移除任务
			parent.removeTask(item.task)
		} else {
			// 其他状态移动到 Undo
			parent.moveTask(item.task, StatusUndo)
		}
	})

	// 创建编辑按钮
	editBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), item.onEditClicked)

	// 创建按钮容器
	buttons := container.NewHBox(checkBtn, deleteBtn)

	item.container = container.NewHBox(
		container.NewHBox(buttons),
		title,
		layout.NewSpacer(),
		editBtn,
	)
	item.label = title
	return item
}

// 处理编辑按钮点击
func (i *TodoItem) onEditClicked() {
	dialog := widget.NewEntry()
	dialog.MultiLine = true
	dialog.SetText(i.task.Title)

	w := fyne.CurrentApp().NewWindow("编辑任务")
	w.SetContent(container.NewVBox(
		dialog,
		container.NewHBox(
			widget.NewButton("取消", func() {
				w.Close()
			}),
			widget.NewButton("保存", func() {
				i.updateText(dialog.Text)
				w.Close()
			}),
		),
	))
	w.Resize(fyne.NewSize(300, 200))
	w.CenterOnScreen()
	w.Show()
}

// 更新文本
func (i *TodoItem) updateText(newText string) {
	i.task.Title = newText
	i.label.Text = newText
	i.parent.refreshAllLists()
}

// TodoList 表示一个状态的任务列表
type StatusList struct {
	status     TaskStatus
	items      []*TodoItem
	list       *widget.List
	parent     *TodoList
	countLabel *widget.Label
}

func NewStatusList(status TaskStatus, parent *TodoList) *StatusList {
	sl := &StatusList{
		status: status,
		parent: parent,
		items:  make([]*TodoItem, 0),
	}

	sl.list = widget.NewList(
		func() int {
			return len(sl.items)
		},
		func() fyne.CanvasObject {
			return NewTodoItem(&models.Task{}, parent).container
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(sl.items) {
				container := obj.(*fyne.Container)
				container.Objects = sl.items[id].container.Objects
			}
		},
	)

	// 设置列表的最小尺寸，确保它能够占满整个高度
	sl.list.Resize(fyne.NewSize(200, 600))

	return sl
}

// 更新列表项
func (sl *StatusList) updateItems(tasks []*models.Task) {
	sl.items = make([]*TodoItem, len(tasks))
	for i, task := range tasks {
		sl.items[i] = NewTodoItem(task, sl.parent)
	}
	sl.list.Refresh()
}

// 修改 TodoList 结构
type TodoList struct {
	tasks       map[string][]*models.Task
	currentDate string
	dateSelect  *widget.Select
	todoList    *StatusList
	doingList   *StatusList
	doneList    *StatusList
	undoList    *StatusList
	input       *widget.Entry
	addBtn      *widget.Button
	container   *fyne.Container
	db          *storage.Database // 添加数据库引用
}

func NewTodoList(db *storage.Database) *TodoList {
	todo := &TodoList{
		tasks: make(map[string][]*models.Task),
		input: widget.NewEntry(),
		db:    db,
	}

	// 从数据库加载日期列表
	dates, err := db.GetDistinctDates()
	if err != nil {
		dates = []string{}
	}

	// 确保今天的日期在列表中
	today := time.Now().Format("2006-01-02")
	hasToday := false
	for _, date := range dates {
		if date == today {
			hasToday = true
			break
		}
	}
	if !hasToday {
		dates = append(dates, today)
	}
	sort.Strings(dates)

	// 初始化日期选择器
	todo.currentDate = today
	todo.dateSelect = widget.NewSelect(dates, todo.onDateSelected)

	// 先初始化所有列表
	todo.todoList = &StatusList{
		status: StatusTodo,
		parent: todo,
		items:  make([]*TodoItem, 0),
	}
	todo.doingList = &StatusList{
		status: StatusDoing,
		parent: todo,
		items:  make([]*TodoItem, 0),
	}
	todo.doneList = &StatusList{
		status: StatusDone,
		parent: todo,
		items:  make([]*TodoItem, 0),
	}
	todo.undoList = &StatusList{
		status: StatusUndo,
		parent: todo,
		items:  make([]*TodoItem, 0),
	}

	// 设置界面
	todo.setup()

	// 最后加载数据并设���选中日期
	if err := todo.loadTasksForDate(today); err != nil {
		fmt.Println("Error loading tasks:", err)
	}
	todo.dateSelect.SetSelected(today) // 设置默认选中今天

	return todo
}

// 加载指定日期的任务
func (t *TodoList) loadTasksForDate(date string) error {
	tasks, err := t.db.GetTasksByDate(date)
	if err != nil {
		return err
	}

	t.tasks[date] = tasks
	t.refreshAllLists()
	return nil
}

// 日期选择回调
func (t *TodoList) onDateSelected(date string) {
	if date == "" {
		return
	}
	t.currentDate = date
	if err := t.loadTasksForDate(date); err != nil {
		fmt.Println("Error loading tasks:", err)
	}
}

// 修改添加任务的方法
func (t *TodoList) addTask() {
	if text := t.input.Text; text != "" {
		task := &models.Task{
			Title:     text,
			Status:    models.TaskStatus(string(StatusTodo)),
			CreatedAt: time.Now(),
			Date:      t.currentDate,
			Priority:  1, // 设置默认优先级
		}

		// 保存到数据库
		if err := t.db.SaveTask(task); err != nil {
			// 处理错误，可以显示一个对话框
			fmt.Println("Error saving task:", err)
			return
		}

		// 更新内存中的任务列表
		if _, ok := t.tasks[t.currentDate]; !ok {
			t.tasks[t.currentDate] = make([]*models.Task, 0)
		}
		t.tasks[t.currentDate] = append(t.tasks[t.currentDate], task)

		// 清空输入框并刷新列表
		t.input.SetText("")
		t.refreshAllLists()
	}
}

// 修改移动任务的方法
func (t *TodoList) moveTask(task *models.Task, newStatus TaskStatus) {
	task.Status = models.TaskStatus(string(newStatus))
	if newStatus == StatusDone {
		now := time.Now()
		task.CompletedAt = &now
	}

	// 更新数据库
	if err := t.db.SaveTask(task); err != nil {
		// 处理错误，可以显示一个对话框
		fmt.Println("Error updating task:", err)
		return
	}

	t.refreshAllLists()
}

// setup 方法中的列表布局
func (t *TodoList) setup() {
	// 创建标题
	title := widget.NewLabelWithStyle("任务管理器", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// 创建日期选择器
	dateContainer := container.NewHBox(
		widget.NewLabel("Date:"),
		t.dateSelect,
	)

	// 创建输入框和添加按钮
	t.input = widget.NewEntry()
	t.input.SetPlaceHolder("Add a new task...")
	t.addBtn = widget.NewButtonWithIcon("Add Task", theme.ContentAddIcon(), t.addTask)

	inputContainer := container.NewBorder(
		nil, nil, nil, t.addBtn,
		t.input,
	)

	// 创建四列布局
	todoList, todoColumn, todoCount := createColumnList("Todo", color.NRGBA{R: 240, G: 248, B: 255, A: 255}, t, StatusTodo)
	doingList, doingColumn, doingCount := createColumnList("Doing", color.NRGBA{R: 255, G: 250, B: 240, A: 255}, t, StatusDoing)
	doneList, doneColumn, doneCount := createColumnList("Done", color.NRGBA{R: 240, G: 255, B: 240, A: 255}, t, StatusDone)
	undoList, undoColumn, undoCount := createColumnList("Undo", color.NRGBA{R: 255, G: 240, B: 240, A: 255}, t, StatusUndo)

	listsContainer := container.NewGridWithColumns(4,
		todoColumn,
		doingColumn,
		doneColumn,
		undoColumn,
	)

	// 使用 Border 布局组织整体界面
	t.container = container.NewBorder(
		container.NewVBox(
			title,
			dateContainer, // 添加日期选择器
			inputContainer,
		),
		nil, nil, nil,
		listsContainer,
	)

	// 只更新必要的字段
	t.todoList.list = todoList
	t.todoList.countLabel = todoCount

	t.doingList.list = doingList
	t.doingList.countLabel = doingCount

	t.doneList.list = doneList
	t.doneList.countLabel = doneCount

	t.undoList.list = undoList
	t.undoList.countLabel = undoCount
}

// 创建列表列
func createColumnList(title string, bgColor color.Color, t *TodoList, status TaskStatus) (*widget.List, *fyne.Container, *widget.Label) {
	countLabel := widget.NewLabel("0")

	list := widget.NewList(
		func() int {
			tasks := t.getTasksByStatus(status)
			countLabel.SetText(fmt.Sprintf("%d", len(tasks))) // 更新数量
			return len(tasks)
		},
		func() fyne.CanvasObject {
			return NewTodoItem(&models.Task{}, t).container
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			tasks := t.getTasksByStatus(status)
			if id < len(tasks) {
				item := NewTodoItem(tasks[id], t)
				container := obj.(*fyne.Container)
				container.Objects = item.container.Objects
			}
		},
	)

	// 创建标题和数量显示
	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	header := container.NewHBox(
		titleLabel,
		widget.NewLabel(" "), // 添加一个空格作为分隔
		countLabel,
	)

	// 创建带背景色的容器
	background := canvas.NewRectangle(bgColor)

	// 返回���个值
	return list, container.NewBorder(
		header,
		nil, nil, nil,
		background,
		list,
	), countLabel
}

// 获取指定状态的任务
func (t *TodoList) getTasksByStatus(status TaskStatus) []*models.Task {
	var result []*models.Task
	if tasks, ok := t.tasks[t.currentDate]; ok {
		for _, task := range tasks {
			if models.TaskStatus(string(status)) == task.Status {
				result = append(result, task)
			}
		}
	}
	return result
}

// 修改刷新方法
func (t *TodoList) refreshAllLists() {
	t.todoList.list.Refresh()
	t.doingList.list.Refresh()
	t.doneList.list.Refresh()
	t.undoList.list.Refresh()

	// 更新所有数量显示
	t.todoList.countLabel.SetText(fmt.Sprintf("%d", len(t.getTasksByStatus(StatusTodo))))
	t.doingList.countLabel.SetText(fmt.Sprintf("%d", len(t.getTasksByStatus(StatusDoing))))
	t.doneList.countLabel.SetText(fmt.Sprintf("%d", len(t.getTasksByStatus(StatusDone))))
	t.undoList.countLabel.SetText(fmt.Sprintf("%d", len(t.getTasksByStatus(StatusUndo))))
}

// 添加移除任务的方法
func (t *TodoList) removeTask(task *models.Task) {
	// 从内存中移除任务
	if tasks, ok := t.tasks[t.currentDate]; ok {
		for i, currentTask := range tasks {
			if currentTask.ID == task.ID {
				t.tasks[t.currentDate] = append(tasks[:i], tasks[i+1:]...)
				break
			}
		}
	}
	// 刷新显示
	t.refreshAllLists()
}
