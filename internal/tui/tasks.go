package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sanja/octocli_cg/internal/brain"
)

type refreshMsg struct{}

type taskModel struct {
	store       brain.Store
	tasks       []brain.Metadata
	selected    int
	currentLog  string
	statusLine  string
	quitting    bool
	titleStyle  lipgloss.Style
	mutedStyle  lipgloss.Style
	highlighted lipgloss.Style
}

func NewTaskModel(store brain.Store) tea.Model {
	return &taskModel{
		store:       store,
		titleStyle:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),
		mutedStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		highlighted: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")),
	}
}

func (m *taskModel) Init() tea.Cmd {
	return tea.Batch(m.refresh(), tickCmd())
}

func (m *taskModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("q", "ctrl+c"))):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if m.selected > 0 {
				m.selected--
			}
			return m, m.loadLog()
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if m.selected < len(m.tasks)-1 {
				m.selected++
			}
			return m, m.loadLog()
		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			return m, tea.Batch(m.refresh(), m.loadLog())
		}
	case refreshMsg:
		return m, tea.Batch(m.refresh(), m.loadLog(), tickCmd())
	case loadedTasksMsg:
		m.tasks = msg.tasks
		if m.selected >= len(m.tasks) && len(m.tasks) > 0 {
			m.selected = len(m.tasks) - 1
		}
		m.statusLine = msg.status
		return m, nil
	case loadedLogMsg:
		m.currentLog = msg.content
		return m, nil
	case errMsg:
		m.statusLine = msg.Error()
		return m, tickCmd()
	}
	return m, nil
}

func (m *taskModel) View() string {
	if m.quitting {
		return "bye\n"
	}

	var left strings.Builder
	left.WriteString(m.titleStyle.Render("octocli_cg task monitor"))
	left.WriteString("\n")
	left.WriteString(m.mutedStyle.Render("↑/↓ move • r refresh • q quit"))
	left.WriteString("\n\n")
	if len(m.tasks) == 0 {
		left.WriteString("No tracked tasks found.\n")
	} else {
		for i, task := range m.tasks {
			line := fmt.Sprintf("%s [%s] %s", task.ID, task.Status, task.Goal)
			if i == m.selected {
				left.WriteString(m.highlighted.Render("> " + line))
			} else {
				left.WriteString("  " + line)
			}
			left.WriteString("\n")
		}
	}

	rightTitle := m.titleStyle.Render("task log")
	logContent := strings.TrimSpace(m.currentLog)
	if logContent == "" {
		logContent = "No log output yet."
	}

	leftPane := lipgloss.NewStyle().Width(50).PaddingRight(2).Render(left.String())
	rightPane := lipgloss.NewStyle().Width(70).Render(rightTitle + "\n\n" + logContent)
	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	footer := "\n" + m.mutedStyle.Render(m.statusLine)
	return body + footer + "\n"
}

type loadedTasksMsg struct {
	tasks  []brain.Metadata
	status string
}

type loadedLogMsg struct{ content string }
type errMsg struct{ error }

func (m *taskModel) refresh() tea.Cmd {
	return func() tea.Msg {
		tasks, err := m.store.List()
		if err != nil {
			return errMsg{err}
		}
		return loadedTasksMsg{tasks: tasks, status: fmt.Sprintf("Last refresh: %s", time.Now().Format(time.Kitchen))}
	}
}

func (m *taskModel) loadLog() tea.Cmd {
	if len(m.tasks) == 0 || m.selected >= len(m.tasks) {
		return func() tea.Msg { return loadedLogMsg{content: ""} }
	}
	id := m.tasks[m.selected].ID
	return func() tea.Msg {
		content, err := m.store.LogContents(id)
		if err != nil {
			return errMsg{err}
		}
		return loadedLogMsg{content: content}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(1200*time.Millisecond, func(time.Time) tea.Msg { return refreshMsg{} })
}
