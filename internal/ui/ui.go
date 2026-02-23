package ui

import (
	"fmt"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/junjiang/gaze/internal/scanner"
)

var (
	// Color scheme
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#F25D94"))

	portStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00D9FF"))

	pidStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888"))

	processStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#88FF88"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			Padding(1, 0)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			Bold(true)
)

type tickMsg time.Time
type scanResultMsg []scanner.PortInfo
type errorMsg struct{ err error }

// SortColumn represents which column to sort by
type SortColumn int

const (
	SortByPort SortColumn = iota
	SortByPID
	SortByProcess
)

// Model represents the application state
type Model struct {
	ports        []scanner.PortInfo
	cursor       int
	table        table.Model
	err          error
	lastScan     time.Time
	isScanning   bool
	sortColumn   SortColumn
	sortAscending bool
}

// InitialModel creates the initial model
func InitialModel() Model {
	columns := []table.Column{
		{Title: "Port", Width: 10},
		{Title: "PID", Width: 10},
		{Title: "Process", Width: 25},
		{Title: "Status", Width: 15},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4"))

	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#F25D94")).
		Bold(true)

	t.SetStyles(s)

	return Model{
		ports:         []scanner.PortInfo{},
		table:         t,
		lastScan:      time.Now(),
		sortColumn:    SortByPort,
		sortAscending: true,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		scanPorts(),
	)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit

		case "k", "K":
			if len(m.ports) > 0 && m.table.Cursor() < len(m.ports) {
				selectedPort := m.ports[m.table.Cursor()]
				if selectedPort.PID != 0 {
					err := scanner.KillProcess(selectedPort.PID)
					if err != nil {
						m.err = fmt.Errorf("failed to kill process %d: %w", selectedPort.PID, err)
					} else {
						// Immediately rescan after killing
						return m, scanPorts()
					}
				}
			}

		case "r", "R":
			// Manual refresh
			return m, scanPorts()

		case "s", "S":
			// Cycle through sort columns
			m.sortColumn = (m.sortColumn + 1) % 3
			m.sortPorts()
			m.updateTableRows()

		case "a", "A":
			// Toggle sort order
			m.sortAscending = !m.sortAscending
			m.sortPorts()
			m.updateTableRows()
		}

	case tickMsg:
		// Auto-refresh every 3 seconds
		return m, tea.Batch(
			tickCmd(),
			scanPorts(),
		)

	case scanResultMsg:
		m.ports = []scanner.PortInfo(msg)
		m.lastScan = time.Now()
		m.isScanning = false
		m.err = nil

		// Sort and update table
		m.sortPorts()
		m.updateTableRows()

	case errorMsg:
		m.err = msg.err
		m.isScanning = false

	case tea.WindowSizeMsg:
		// Handle window resize
		m.table.SetHeight(msg.Height - 10)
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the UI
func (m Model) View() string {
	var s string

	// Title
	s += titleStyle.Render("🔍 GAZE - Local Port Monitor") + "\n\n"

	// Table
	s += m.table.View() + "\n\n"

	// Status line
	statusLine := fmt.Sprintf("Monitoring %d ports • Last scan: %s ago",
		len(m.ports),
		time.Since(m.lastScan).Round(time.Second))

	if m.isScanning {
		statusLine += " • Scanning..."
	}

	s += statusStyle.Render(statusLine) + "\n"

	// Error display
	if m.err != nil {
		s += errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n"
	}

	// Sort indicator
	sortInfo := m.getSortIndicator()
	s += lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(sortInfo) + "\n"

	// Help text
	help := "↑/↓: Navigate • s: Sort Column • a: Sort Order • k: Kill • r: Refresh • q: Quit"
	s += helpStyle.Render(help)

	return s
}

// tickCmd sends a tick message every 3 seconds
func tickCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// scanPorts runs the port scanner in the background
func scanPorts() tea.Cmd {
	return func() tea.Msg {
		ports, err := scanner.ScanPorts()
		if err != nil {
			return errorMsg{err}
		}
		return scanResultMsg(ports)
	}
}

// sortPorts sorts the ports based on current sort settings
func (m *Model) sortPorts() {
	sort.Slice(m.ports, func(i, j int) bool {
		var less bool
		switch m.sortColumn {
		case SortByPort:
			less = m.ports[i].Port < m.ports[j].Port
		case SortByPID:
			less = m.ports[i].PID < m.ports[j].PID
		case SortByProcess:
			less = m.ports[i].Process < m.ports[j].Process
		}
		if !m.sortAscending {
			return !less
		}
		return less
	})
}

// updateTableRows updates the table with current port data
func (m *Model) updateTableRows() {
	rows := []table.Row{}
	for _, p := range m.ports {
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", p.Port),
			fmt.Sprintf("%d", p.PID),
			p.Process,
			p.Status,
		})
	}
	m.table.SetRows(rows)
}

// getSortIndicator returns a string showing the current sort state
func (m Model) getSortIndicator() string {
	var column string
	switch m.sortColumn {
	case SortByPort:
		column = "Port"
	case SortByPID:
		column = "PID"
	case SortByProcess:
		column = "Process"
	}

	direction := "↑"
	if !m.sortAscending {
		direction = "↓"
	}

	return fmt.Sprintf("Sorted by: %s %s", column, direction)
}
