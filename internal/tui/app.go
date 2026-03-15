package tui

import (
	"errors"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/jwp23/kanteto/internal/nlp"
	"github.com/jwp23/kanteto/internal/task"
)

// Syncer provides sync operations for the TUI.
type Syncer interface {
	Push() error
	Pull() error
	HasRemote(name string) bool
}

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

	// Edit-time mode
	editing   bool
	editInput string

	// Tag mode
	tagging  bool
	tagInput string

	// Untag mode
	untagging  bool
	untagInput string

	// Reparse confirmation
	reparseConfirm bool
	reparseResult  string

	// Week view cursor (0=Sunday, 6=Saturday)
	weekCursorDay int

	// Month view cursor (1-based day of month)
	monthCursorDay int

	// Pre-fetched tasks for month view (keyed by day of month)
	monthTasks map[int][]task.Task

	// Pre-fetched tasks for week view (keyed by date string "2006-01-02")
	weekTasks map[string][]task.Task

	// Active profile name
	profile string

	// Midnight detection
	lastDate int // YearDay of the last known date

	// Alert
	alertedIDs  map[string]bool
	alertPlayer AlertPlayer

	// Sync
	syncer     Syncer
	syncStatus string

	// Help overlay
	showHelp bool

	// Dimensions
	width  int
	height int

	err error
}

type refreshMsg struct{}
type tickMsg time.Time
type syncResultMsg struct {
	op  string
	err error
}
type clearSyncMsg struct{}

// New creates and returns the Bubble Tea program.
func New(svc *task.Service, profile string, syncer Syncer, alertPlayer AlertPlayer) *tea.Program {
	now := time.Now()
	m := model{
		svc:         svc,
		viewMode:    dayView,
		viewDate:    now,
		lastDate:    now.YearDay(),
		profile:     profile,
		syncer:      syncer,
		alertedIDs:  make(map[string]bool),
		alertPlayer: alertPlayer,
	}
	return tea.NewProgram(m, tea.WithAltScreen())
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.loadTasks, tickEvery(time.Minute))
}

func tickEvery(d time.Duration) tea.Cmd {
	return tea.Every(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
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

	case tickMsg:
		now := time.Time(msg)
		if now.YearDay() != m.lastDate || now.Year() != m.viewDate.Year() {
			m.lastDate = now.YearDay()
			oldToday := time.Date(m.viewDate.Year(), m.viewDate.Month(), m.viewDate.Day(), 0, 0, 0, 0, m.viewDate.Location())
			yesterday := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
			if oldToday.Equal(yesterday) {
				m.viewDate = now
			}
		}
		m.refreshData()

		// Check for newly-due tasks
		newlyDue := newlyDueTasks(m.allTasks, time.Now(), m.alertedIDs)
		for _, id := range newlyDue {
			m.alertedIDs[id] = true
		}
		if len(newlyDue) > 0 && m.alertPlayer != nil {
			return m, tea.Batch(playAlert(m.alertPlayer), tickEvery(time.Minute))
		}
		return m, tickEvery(time.Minute)

	case alertPlayedMsg:
		return m, nil

	case syncResultMsg:
		if msg.err != nil {
			m.err = msg.err
			m.syncStatus = ""
		} else {
			switch msg.op {
			case "push":
				m.syncStatus = "Push complete"
			case "pull":
				m.syncStatus = "Pull complete"
				m.refreshData()
			}
		}
		return m, clearSyncStatusAfter(3 * time.Second)

	case clearSyncMsg:
		m.syncStatus = ""
		return m, nil

	case tea.KeyMsg:
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		if m.reparseConfirm {
			return m.handleReparseConfirm(msg)
		}

		if m.adding {
			return m.handleAddInput(msg)
		}

		if m.snoozing {
			return m.handleSnoozeInput(msg)
		}

		if m.editing {
			return m.handleEditInput(msg)
		}

		if m.tagging {
			return m.handleTagInput(msg)
		}

		if m.untagging {
			return m.handleUntagInput(msg)
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

	all, err := m.svc.ListAll()
	if err != nil {
		m.err = err
		return
	}

	m.overdue = nil
	m.today = nil
	m.upcoming = nil
	m.undated = nil

	realNow := time.Now()
	for _, t := range all {
		switch {
		case t.DueAt == nil:
			m.undated = append(m.undated, t)
		case t.DueAt.Before(realNow) && t.DueAt.Before(startOfDay):
			m.overdue = append(m.overdue, t)
		case !t.DueAt.Before(startOfDay) && t.DueAt.Before(endOfDay):
			m.today = append(m.today, t)
		case !t.DueAt.Before(endOfDay) && t.DueAt.Before(endOfWeek):
			m.upcoming = append(m.upcoming, t)
		default:
			if t.DueAt.Before(realNow) {
				m.overdue = append(m.overdue, t)
			}
		}
	}

	// Flatten for cursor
	m.allTasks = nil
	m.allTasks = append(m.allTasks, m.overdue...)
	m.allTasks = append(m.allTasks, m.today...)
	m.allTasks = append(m.allTasks, m.upcoming...)
	m.allTasks = append(m.allTasks, m.undated...)

	if m.cursor >= len(m.allTasks) {
		m.cursor = max(0, len(m.allTasks)-1)
	}

	// Pre-compute month tasks from fetched data
	if m.viewMode == monthView {
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		firstOfNextMonth := firstOfMonth.AddDate(0, 1, 0)
		m.monthTasks = make(map[int][]task.Task)
		for _, t := range all {
			if t.DueAt != nil && !t.DueAt.Before(firstOfMonth) && t.DueAt.Before(firstOfNextMonth) {
				day := t.DueAt.Day()
				m.monthTasks[day] = append(m.monthTasks[day], t)
			}
		}
	}

	// Pre-compute week tasks from fetched data
	if m.viewMode == weekView {
		weekday := now.Weekday()
		startOfWeek := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		startOfWeek = startOfWeek.AddDate(0, 0, -int(weekday))
		endOfWeek := startOfWeek.AddDate(0, 0, 7)
		m.weekTasks = make(map[string][]task.Task)
		for _, t := range all {
			if t.DueAt != nil && !t.DueAt.Before(startOfWeek) && t.DueAt.Before(endOfWeek) {
				key := t.DueAt.Format("2006-01-02")
				m.weekTasks[key] = append(m.weekTasks[key], t)
			}
		}
	}
}

func (m model) handleKeypress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.err = nil
	m.reparseResult = ""
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		if m.viewMode == weekView {
			return m.weekCursorMove(1), nil
		}
		if m.viewMode == monthView {
			return m.monthCursorMove(7), nil
		}
		if m.cursor < len(m.allTasks)-1 {
			m.cursor++
		}
		return m, nil

	case "k", "up":
		if m.viewMode == weekView {
			return m.weekCursorMove(-1), nil
		}
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

	case "e":
		if len(m.allTasks) > 0 && m.cursor < len(m.allTasks) {
			m.editing = true
			m.editInput = ""
		}
		return m, nil

	case "d":
		m.viewMode = dayView
		m.refreshData()
		return m, nil

	case "w":
		m.viewMode = weekView
		m.weekCursorDay = int(m.viewDate.Weekday())
		m.refreshData()
		return m, nil

	case "m":
		m.viewMode = monthView
		m.monthCursorDay = m.viewDate.Day()
		m.refreshData()
		return m, nil

	case "left":
		if m.viewMode == weekView {
			return m.weekCursorMove(-1), nil
		}
		if m.viewMode == monthView {
			return m.monthCursorMove(-1), nil
		}
		return m.timeNav(-1), nil

	case "right":
		if m.viewMode == weekView {
			return m.weekCursorMove(1), nil
		}
		if m.viewMode == monthView {
			return m.monthCursorMove(1), nil
		}
		return m.timeNav(1), nil

	case "h":
		return m.timeNav(-1), nil

	case "l":
		return m.timeNav(1), nil

	case "enter":
		if m.viewMode == weekView {
			return m.weekDrillDown(), nil
		}
		if m.viewMode == monthView {
			return m.monthDrillDown(), nil
		}
		return m, nil

	case "t":
		if len(m.allTasks) > 0 && m.cursor < len(m.allTasks) {
			m.tagging = true
			m.tagInput = ""
		}
		return m, nil

	case "T":
		if len(m.allTasks) > 0 && m.cursor < len(m.allTasks) {
			m.untagging = true
			m.untagInput = ""
		}
		return m, nil

	case ".":
		m.viewDate = time.Now()
		m.refreshData()
		return m, nil

	case "R":
		undated, err := m.svc.ListUndated()
		if err != nil {
			m.err = err
			return m, nil
		}
		if len(undated) == 0 {
			m.reparseResult = "No undated tasks to reparse"
			return m, nil
		}
		// Count how many have extractable deadlines
		count := 0
		for _, tk := range undated {
			_, dueAt := nlp.ExtractDeadline(tk.Title)
			if dueAt != nil {
				count++
			}
		}
		if count == 0 {
			m.reparseResult = fmt.Sprintf("Scanned %d undated tasks — no deadlines found", len(undated))
			return m, nil
		}
		m.reparseConfirm = true
		m.reparseResult = fmt.Sprintf("Found %d/%d tasks with deadlines. Press y to reparse, esc to cancel", count, len(undated))
		return m, nil

	case "P":
		if m.syncer == nil {
			m.err = errors.New("sync not available")
			return m, nil
		}
		if !m.syncer.HasRemote("origin") {
			m.err = errors.New("no remote configured")
			return m, nil
		}
		m.syncStatus = "Pushing..."
		return m, m.doPush()

	case "p":
		if m.syncer == nil {
			m.err = errors.New("sync not available")
			return m, nil
		}
		if !m.syncer.HasRemote("origin") {
			m.err = errors.New("no remote configured")
			return m, nil
		}
		m.syncStatus = "Pulling..."
		return m, m.doPull()

	case "?":
		m.showHelp = true
		return m, nil
	}

	return m, nil
}

func (m model) doPush() tea.Cmd {
	syncer := m.syncer
	return func() tea.Msg {
		err := syncer.Push()
		return syncResultMsg{op: "push", err: err}
	}
}

func (m model) doPull() tea.Cmd {
	syncer := m.syncer
	return func() tea.Msg {
		err := syncer.Pull()
		return syncResultMsg{op: "pull", err: err}
	}
}

func clearSyncStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearSyncMsg{}
	})
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

func (m model) weekCursorMove(delta int) model {
	newDay := m.weekCursorDay + delta
	if newDay < 0 {
		newDay = 0
	} else if newDay > 6 {
		newDay = 6
	}
	m.weekCursorDay = newDay
	return m
}

func (m model) weekDrillDown() model {
	weekday := m.viewDate.Weekday()
	startOfWeek := m.viewDate.AddDate(0, 0, -int(weekday))
	m.viewDate = startOfWeek.AddDate(0, 0, m.weekCursorDay)
	m.viewMode = dayView
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

func (m model) handleReparseConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		result, err := m.svc.Reparse()
		if err != nil {
			m.err = err
		} else {
			m.reparseResult = fmt.Sprintf("Reparsed: %d/%d tasks updated", result.Updated, result.Total)
		}
		m.reparseConfirm = false
		m.refreshData()
		return m, nil

	case "esc", "n":
		m.reparseConfirm = false
		m.reparseResult = ""
		return m, nil
	}

	return m, nil
}

func (m model) handleTagInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.tagInput != "" {
			t := m.allTasks[m.cursor]
			if err := m.svc.AddTag(t.ID, m.tagInput); err != nil {
				m.err = err
			}
			m.refreshData()
		}
		m.tagging = false
		m.tagInput = ""
		return m, nil

	case "esc":
		m.tagging = false
		m.tagInput = ""
		return m, nil

	case "backspace":
		if len(m.tagInput) > 0 {
			m.tagInput = m.tagInput[:len(m.tagInput)-1]
		}
		return m, nil

	default:
		if len(msg.String()) == 1 {
			m.tagInput += msg.String()
		}
		return m, nil
	}
}

func (m model) handleUntagInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.untagInput != "" {
			t := m.allTasks[m.cursor]
			if err := m.svc.RemoveTag(t.ID, m.untagInput); err != nil {
				m.err = err
			}
			m.refreshData()
		}
		m.untagging = false
		m.untagInput = ""
		return m, nil

	case "esc":
		m.untagging = false
		m.untagInput = ""
		return m, nil

	case "backspace":
		if len(m.untagInput) > 0 {
			m.untagInput = m.untagInput[:len(m.untagInput)-1]
		}
		return m, nil

	default:
		if len(msg.String()) == 1 {
			m.untagInput += msg.String()
		}
		return m, nil
	}
}

func (m model) handleEditInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.editInput != "" {
			t, err := nlp.ParseDate(m.editInput)
			if err != nil {
				m.err = err
			} else {
				tk := m.allTasks[m.cursor]
				if err := m.svc.SetDueAt(tk.ID, t); err != nil {
					m.err = err
				}
			}
			m.refreshData()
		}
		m.editing = false
		m.editInput = ""
		return m, nil

	case "esc":
		m.editing = false
		m.editInput = ""
		return m, nil

	case "backspace":
		if len(m.editInput) > 0 {
			m.editInput = m.editInput[:len(m.editInput)-1]
		}
		return m, nil

	default:
		if len(msg.String()) == 1 {
			m.editInput += msg.String()
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

	if m.editing {
		return fmt.Sprintf("  new deadline: %s█  (e.g. friday 3pm, tomorrow, march 15)", m.editInput)
	}

	if m.tagging {
		return fmt.Sprintf("  tag: %s█  (enter to add, esc to cancel)", m.tagInput)
	}

	if m.untagging {
		return fmt.Sprintf("  remove tag: %s█  (enter to remove, esc to cancel)", m.untagInput)
	}

	if m.reparseResult != "" {
		return fmt.Sprintf("  %s", m.reparseResult)
	}

	if m.syncStatus != "" {
		return fmt.Sprintf("  %s", m.syncStatus)
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

	keys := "j/k:move  space:done  a:add  e:edit  s:snooze  x:delete  t:tag  T:untag  p:pull  P:push  h/l:nav  .:today  ?:help  q:quit"
	if m.viewMode == weekView {
		keys = "j/k:day  ←/→:day  enter:open  h/l:week  a:add  .:today  ?:help  q:quit"
	}
	if m.viewMode == monthView {
		keys = "j/k:week  ←/→:day  enter:open  h/l:month  a:add  .:today  ?:help  q:quit"
	}
	return helpStyle.Render(fmt.Sprintf("  %s  |  %s", viewIndicator, keys))
}

func renderHelp(m model) string {
	help := `
  Kanteto — Keyboard Shortcuts

  Navigation
    j / ↓       Move down (day in week view, week in month view)
    k / ↑       Move up (day in week view, week in month view)
    ← / →       Move by day in week/month view
    h / l       Previous/next day, week, or month
    .           Jump to today
    enter       Open selected day (week/month view)

  Views
    d           Day view
    w           Week view
    m           Month view

  Actions
    space       Complete task
    a           Add new task
    e           Edit deadline
    s           Snooze task
    t           Add tag
    T           Remove tag
    R           Reparse undated tasks
    p           Pull from remote
    P           Push to remote
    x / delete  Delete task
    ?           Toggle help
    q           Quit

  Press any key to close this help.
`
	return help
}
