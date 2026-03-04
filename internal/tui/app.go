package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/jwp23/kanteto/internal/task"
)

type viewMode int

const (
	dayView viewMode = iota
	weekView
	monthView
)

type model struct {
	svc      *task.Service
	viewMode viewMode
	viewDate time.Time
	cursor   int

	// Task lists for day view
	overdue  []task.Task
	today    []task.Task
	upcoming []task.Task
	undated  []task.Task

	// All tasks flattened for cursor navigation
	allTasks []task.Task

	// Input mode
	adding   bool
	addInput string

	// Help overlay
	showHelp bool

	// Dimensions
	width  int
	height int

	err error
}

type refreshMsg struct{}

// New creates and returns the Bubble Tea program.
func New(svc *task.Service) *tea.Program {
	m := model{
		svc:      svc,
		viewMode: dayView,
		viewDate: time.Now(),
	}
	return tea.NewProgram(m, tea.WithAltScreen())
}

func (m model) Init() tea.Cmd {
	return m.loadTasks
}

func (m model) loadTasks() tea.Msg {
	return refreshMsg{}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case refreshMsg:
		m.refreshData()
		return m, nil

	case tea.KeyMsg:
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		if m.adding {
			return m.handleAddInput(msg)
		}

		return m.handleKeypress(msg)
	}

	return m, nil
}

func (m *model) refreshData() {
	now := m.viewDate
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	endOfWeek := endOfDay.AddDate(0, 0, 7)

	overdue, _ := m.svc.ListOverdue()
	today, _ := m.svc.ListByDateRange(startOfDay, endOfDay)
	upcoming, _ := m.svc.ListByDateRange(endOfDay, endOfWeek)
	undated, _ := m.svc.ListUndated()

	m.overdue = overdue
	m.today = today
	m.upcoming = upcoming
	m.undated = undated

	// Flatten for cursor
	m.allTasks = nil
	m.allTasks = append(m.allTasks, overdue...)
	m.allTasks = append(m.allTasks, today...)
	m.allTasks = append(m.allTasks, upcoming...)
	m.allTasks = append(m.allTasks, undated...)

	if m.cursor >= len(m.allTasks) {
		m.cursor = max(0, len(m.allTasks)-1)
	}
}

func (m model) handleKeypress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		if m.cursor < len(m.allTasks)-1 {
			m.cursor++
		}
		return m, nil

	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case " ":
		if len(m.allTasks) > 0 && m.cursor < len(m.allTasks) {
			t := m.allTasks[m.cursor]
			m.svc.Complete(t.ID)
			m.refreshData()
		}
		return m, nil

	case "x", "delete":
		if len(m.allTasks) > 0 && m.cursor < len(m.allTasks) {
			t := m.allTasks[m.cursor]
			m.svc.Delete(t.ID)
			m.refreshData()
		}
		return m, nil

	case "a":
		m.adding = true
		m.addInput = ""
		return m, nil

	case "d":
		m.viewMode = dayView
		m.refreshData()
		return m, nil

	case "w":
		m.viewMode = weekView
		m.refreshData()
		return m, nil

	case "m":
		m.viewMode = monthView
		m.refreshData()
		return m, nil

	case "h", "left":
		m.viewDate = m.viewDate.AddDate(0, 0, -1)
		if m.viewMode == weekView {
			m.viewDate = m.viewDate.AddDate(0, 0, -6)
		} else if m.viewMode == monthView {
			m.viewDate = m.viewDate.AddDate(0, -1, 0)
		}
		m.refreshData()
		return m, nil

	case "l", "right":
		m.viewDate = m.viewDate.AddDate(0, 0, 1)
		if m.viewMode == weekView {
			m.viewDate = m.viewDate.AddDate(0, 0, 6)
		} else if m.viewMode == monthView {
			m.viewDate = m.viewDate.AddDate(0, 1, 0)
		}
		m.refreshData()
		return m, nil

	case "t":
		m.viewDate = time.Now()
		m.refreshData()
		return m, nil

	case "?":
		m.showHelp = true
		return m, nil
	}

	return m, nil
}

func (m model) handleAddInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.addInput != "" {
			m.svc.Add(m.addInput, nil)
			m.refreshData()
		}
		m.adding = false
		m.addInput = ""
		return m, nil

	case "esc":
		m.adding = false
		m.addInput = ""
		return m, nil

	case "backspace":
		if len(m.addInput) > 0 {
			m.addInput = m.addInput[:len(m.addInput)-1]
		}
		return m, nil

	default:
		if len(msg.String()) == 1 {
			m.addInput += msg.String()
		}
		return m, nil
	}
}

func (m model) View() string {
	if m.showHelp {
		return renderHelp(m)
	}

	var content string
	switch m.viewMode {
	case dayView:
		content = renderDayView(m)
	case weekView:
		content = renderWeekView(m)
	case monthView:
		content = renderMonthView(m)
	}

	footer := renderFooter(m)
	return content + "\n" + footer
}

func renderFooter(m model) string {
	if m.adding {
		return fmt.Sprintf("  > %s█", m.addInput)
	}

	viewIndicator := "[d]ay [w]eek [m]onth"
	switch m.viewMode {
	case dayView:
		viewIndicator = "[D]ay [w]eek [m]onth"
	case weekView:
		viewIndicator = "[d]ay [W]eek [m]onth"
	case monthView:
		viewIndicator = "[d]ay [w]eek [M]onth"
	}

	return helpStyle.Render(fmt.Sprintf("  %s  |  j/k:move  space:done  a:add  x:delete  h/l:nav  t:today  ?:help  q:quit", viewIndicator))
}

func renderHelp(m model) string {
	help := `
  Kanteto — Keyboard Shortcuts

  Navigation
    j / ↓       Move down
    k / ↑       Move up
    h / ←       Previous day/week/month
    l / →       Next day/week/month
    t           Jump to today

  Views
    d           Day view
    w           Week view
    m           Month view

  Actions
    space       Complete task
    a           Add new task
    x / delete  Delete task
    ?           Toggle help
    q           Quit

  Press any key to close this help.
`
	return help
}
