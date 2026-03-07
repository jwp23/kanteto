package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/jwp23/kanteto/internal/nlp"
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

	// Snooze mode
	snoozing    bool
	snoozeInput string

	// Month view cursor (1-based day of month)
	monthCursorDay int

	// Pre-fetched tasks for month view (keyed by day of month)
	monthTasks map[int][]task.Task

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

		if m.snoozing {
			return m.handleSnoozeInput(msg)
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

	// Pre-fetch month tasks in a single query
	if m.viewMode == monthView {
		now := m.viewDate
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		firstOfNextMonth := firstOfMonth.AddDate(0, 1, 0)
		monthTasks, _ := m.svc.ListByDateRange(firstOfMonth, firstOfNextMonth)
		m.monthTasks = make(map[int][]task.Task)
		for _, t := range monthTasks {
			if t.DueAt != nil {
				day := t.DueAt.Day()
				m.monthTasks[day] = append(m.monthTasks[day], t)
			}
		}
	}
}

func (m model) handleKeypress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.err = nil
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		if m.viewMode == monthView {
			return m.monthCursorMove(7), nil
		}
		if m.cursor < len(m.allTasks)-1 {
			m.cursor++
		}
		return m, nil

	case "k", "up":
		if m.viewMode == monthView {
			return m.monthCursorMove(-7), nil
		}
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case " ":
		if len(m.allTasks) > 0 && m.cursor < len(m.allTasks) {
			t := m.allTasks[m.cursor]
			if err := m.svc.Complete(t.ID); err != nil {
				m.err = err
			}
			m.refreshData()
		}
		return m, nil

	case "x", "delete":
		if len(m.allTasks) > 0 && m.cursor < len(m.allTasks) {
			t := m.allTasks[m.cursor]
			if err := m.svc.Delete(t.ID); err != nil {
				m.err = err
			}
			m.refreshData()
		}
		return m, nil

	case "a":
		m.adding = true
		m.addInput = ""
		return m, nil

	case "s":
		if len(m.allTasks) > 0 && m.cursor < len(m.allTasks) {
			m.snoozing = true
			m.snoozeInput = ""
		}
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
		m.monthCursorDay = m.viewDate.Day()
		m.refreshData()
		return m, nil

	case "left":
		if m.viewMode == monthView {
			return m.monthCursorMove(-1), nil
		}
		return m.timeNav(-1), nil

	case "right":
		if m.viewMode == monthView {
			return m.monthCursorMove(1), nil
		}
		return m.timeNav(1), nil

	case "h":
		return m.timeNav(-1), nil

	case "l":
		return m.timeNav(1), nil

	case "enter":
		if m.viewMode == monthView {
			return m.monthDrillDown(), nil
		}
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

func (m model) timeNav(dir int) model {
	switch m.viewMode {
	case dayView:
		m.viewDate = m.viewDate.AddDate(0, 0, dir)
	case weekView:
		m.viewDate = m.viewDate.AddDate(0, 0, 7*dir)
	case monthView:
		m.viewDate = m.viewDate.AddDate(0, dir, 0)
		m.monthCursorDay = min(m.monthCursorDay, daysInMonth(m.viewDate))
	}
	m.refreshData()
	return m
}

func (m model) monthCursorMove(delta int) model {
	dim := daysInMonth(m.viewDate)
	newDay := m.monthCursorDay + delta
	if newDay < 1 {
		newDay = 1
	} else if newDay > dim {
		newDay = dim
	}
	m.monthCursorDay = newDay
	return m
}

func (m model) monthDrillDown() model {
	m.viewDate = time.Date(m.viewDate.Year(), m.viewDate.Month(), m.monthCursorDay, 0, 0, 0, 0, m.viewDate.Location())
	m.viewMode = dayView
	m.refreshData()
	return m
}

func daysInMonth(t time.Time) int {
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, t.Location()).Day()
}

func (m model) handleAddInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.addInput != "" {
			title, dueAt := nlp.ExtractDeadline(m.addInput)
			if _, err := m.svc.Add(title, dueAt); err != nil {
				m.err = err
			}
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

func (m model) handleSnoozeInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.snoozeInput != "" {
			d, err := nlp.ParseDuration(m.snoozeInput)
			if err != nil {
				m.err = err
			} else {
				t := m.allTasks[m.cursor]
				if err := m.svc.Snooze(t.ID, d); err != nil {
					m.err = err
				}
			}
			m.refreshData()
		}
		m.snoozing = false
		m.snoozeInput = ""
		return m, nil

	case "esc":
		m.snoozing = false
		m.snoozeInput = ""
		return m, nil

	case "backspace":
		if len(m.snoozeInput) > 0 {
			m.snoozeInput = m.snoozeInput[:len(m.snoozeInput)-1]
		}
		return m, nil

	default:
		if len(msg.String()) == 1 {
			m.snoozeInput += msg.String()
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
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("  Error: %s", m.err))
	}

	if m.adding {
		return fmt.Sprintf("  > %s█", m.addInput)
	}

	if m.snoozing {
		return fmt.Sprintf("  snooze for: %s█  (e.g. 1h, 30m, 2 hours)", m.snoozeInput)
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

	keys := "j/k:move  space:done  a:add  s:snooze  x:delete  h/l:nav  t:today  ?:help  q:quit"
	if m.viewMode == monthView {
		keys = "j/k:week  ←/→:day  enter:open  h/l:month  a:add  t:today  ?:help  q:quit"
	}
	return helpStyle.Render(fmt.Sprintf("  %s  |  %s", viewIndicator, keys))
}

func renderHelp(m model) string {
	help := `
  Kanteto — Keyboard Shortcuts

  Navigation
    j / ↓       Move down (week in month view)
    k / ↑       Move up (week in month view)
    ← / →       Move by day in month view
    h / l       Previous/next day, week, or month
    t           Jump to today
    enter       Open selected day (month view)

  Views
    d           Day view
    w           Week view
    m           Month view

  Actions
    space       Complete task
    a           Add new task
    s           Snooze task
    x / delete  Delete task
    ?           Toggle help
    q           Quit

  Press any key to close this help.
`
	return help
}
