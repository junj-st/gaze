package ui

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/junjiang/gaze/internal/export"
	"github.com/junjiang/gaze/internal/history"
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

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true)

	uptimeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00D9FF"))

	eventOpenStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true)

	eventCloseStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			Bold(true)
)

type tickMsg time.Time
type scanResultMsg []scanner.PortInfo
type errorMsg struct{ err error }
type exportSuccessMsg struct{ path string }

// ViewMode represents the current view
type ViewMode int

const (
	ViewPorts ViewMode = iota
	ViewHistory
)

// SortColumn represents which column to sort by
type SortColumn int

const (
	SortByPort SortColumn = iota
	SortByPID
	SortByProcess
)

// Model represents the application state
type Model struct {
	ports         []scanner.PortInfo
	cursor        int
	table         table.Model
	err           error
	lastScan      time.Time
	isScanning    bool
	sortColumn    SortColumn
	sortAscending bool
	historyTracker *history.Tracker
	viewMode      ViewMode
	exportMsg     string
	exportMsgTime time.Time
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
		ports:          []scanner.PortInfo{},
		table:          t,
		lastScan:       time.Now(),
		sortColumn:     SortByPort,
		sortAscending:  true,
		historyTracker: history.NewTracker(1000, 500), // Track last 1000 events, 500 ports
		viewMode:       ViewPorts,
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

		case "h", "H":
			// Toggle history view
			if m.viewMode == ViewPorts {
				m.viewMode = ViewHistory
				m.updateHistoryTable()
			} else {
				m.viewMode = ViewPorts
				m.updateTableRows()
			}

		case "e", "E":
			// Export current data
			if len(m.ports) > 0 {
				return m, exportData(m.ports)
			}
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

		// Update history tracker
		m.historyTracker.Update(m.ports)

		// Sort and update table
		m.sortPorts()
		if m.viewMode == ViewPorts {
			m.updateTableRows()
		} else {
			m.updateHistoryTable()
		}

	case exportSuccessMsg:
		m.exportMsg = fmt.Sprintf("Exported to: %s", msg.path)
		m.exportMsgTime = time.Now()

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
	if m.viewMode == ViewPorts {
		s += titleStyle.Render("🔍 GAZE - Local Port Monitor") + "\n\n"
	} else {
		s += titleStyle.Render("📜 GAZE - Port History") + "\n\n"
	}

	// Table
	s += m.table.View() + "\n\n"

	// Status line
	if m.viewMode == ViewPorts {
		statusLine := fmt.Sprintf("Monitoring %d ports • Last scan: %s ago",
			len(m.ports),
			time.Since(m.lastScan).Round(time.Second))

		if m.isScanning {
			statusLine += " • Scanning..."
		}

		s += statusStyle.Render(statusLine) + "\n"
	} else {
		// History view status
		stats := m.historyTracker.GetStats()
		statusLine := fmt.Sprintf("Tracked: %d ports • Active: %d • Events: %d",
			stats.TotalPortsTracked,
			stats.ActivePorts,
			stats.TotalEvents)
		s += statusStyle.Render(statusLine) + "\n"
	}

	// Export success message (fade after 3 seconds)
	if m.exportMsg != "" && time.Since(m.exportMsgTime) < 3*time.Second {
		s += successStyle.Render(m.exportMsg) + "\n"
	}

	// Error display
	if m.err != nil {
		s += errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n"
	}

	// Sort indicator (only in ports view)
	if m.viewMode == ViewPorts {
		sortInfo := m.getSortIndicator()
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(sortInfo) + "\n"
	}

	// Help text
	if m.viewMode == ViewPorts {
		help := "↑/↓: Navigate • s: Sort • a: Order • e: Export • h: History • k: Kill • r: Refresh • q: Quit"
		s += helpStyle.Render(help)
	} else {
		help := "↑/↓: Navigate • h: Back to Ports • e: Export • q: Quit"
		s += helpStyle.Render(help)
	}

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
	// Update columns to include Uptime
	columns := []table.Column{
		{Title: "Port", Width: 10},
		{Title: "PID", Width: 10},
		{Title: "Process", Width: 25},
		{Title: "Uptime", Width: 15},
		{Title: "Status", Width: 15},
	}
	m.table.SetColumns(columns)

	rows := []table.Row{}
	for _, p := range m.ports {
		uptime := history.FormatUptime(m.historyTracker.GetUptime(p.Port))
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", p.Port),
			fmt.Sprintf("%d", p.PID),
			p.Process,
			uptime,
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

// updateHistoryTable updates the table with port history data
func (m *Model) updateHistoryTable() {
	// Update columns for history view
	columns := []table.Column{
		{Title: "Port", Width: 10},
		{Title: "Process", Width: 25},
		{Title: "Status", Width: 10},
		{Title: "First Seen", Width: 20},
		{Title: "Last Seen", Width: 20},
		{Title: "Uptime", Width: 15},
	}
	m.table.SetColumns(columns)

	histories := m.historyTracker.GetAllHistory()
	rows := []table.Row{}

	for _, h := range histories {
		status := "CLOSED"
		statusTime := h.LastSeen.Format("15:04:05")
		if h.IsActive {
			status = "ACTIVE"
		}

		uptime := "-"
		if h.IsActive {
			uptime = history.FormatUptime(time.Since(h.FirstSeen))
		}

		rows = append(rows, table.Row{
			fmt.Sprintf("%d", h.Port),
			h.Process,
			status,
			h.FirstSeen.Format("15:04:05"),
			statusTime,
			uptime,
		})
	}

	m.table.SetRows(rows)
}

// exportData exports the current port data to files
func exportData(ports []scanner.PortInfo) tea.Cmd {
	return func() tea.Msg {
		// Get home directory for exports
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return errorMsg{fmt.Errorf("failed to get home directory: %w", err)}
		}

		exportDir := homeDir

		// Export to both JSON and CSV
		jsonPath, err := export.ToJSON(ports, exportDir)
		if err != nil {
			return errorMsg{fmt.Errorf("failed to export JSON: %w", err)}
		}

		csvPath, err := export.ToCSV(ports, exportDir)
		if err != nil {
			return errorMsg{fmt.Errorf("failed to export CSV: %w", err)}
		}

		// Return success with both paths
		paths := fmt.Sprintf("%s, %s", jsonPath, csvPath)
		return exportSuccessMsg{path: paths}
	}
}

