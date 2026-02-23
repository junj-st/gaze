package history

import (
	"fmt"
	"sort"
	"time"

	"github.com/junjiang/gaze/internal/scanner"
)

// PortEvent represents a port state change event
type PortEvent struct {
	Port      int
	PID       int32
	Process   string
	EventType EventType
	Timestamp time.Time
}

// EventType represents the type of port event
type EventType string

const (
	EventPortOpened EventType = "OPENED"
	EventPortClosed EventType = "CLOSED"
)

// PortHistory tracks a port's lifecycle
type PortHistory struct {
	Port      int
	PID       int32
	Process   string
	FirstSeen time.Time
	LastSeen  time.Time
	IsActive  bool
	OpenCount int
	Events    []PortEvent
}

// Tracker manages port history tracking
type Tracker struct {
	history      map[int]*PortHistory // Key: port number
	events       []PortEvent
	maxEvents    int
	maxHistories int
}

// NewTracker creates a new history tracker
func NewTracker(maxEvents, maxHistories int) *Tracker {
	return &Tracker{
		history:      make(map[int]*PortHistory),
		events:       make([]PortEvent, 0),
		maxEvents:    maxEvents,
		maxHistories: maxHistories,
	}
}

// Update processes a new scan and tracks changes
func (t *Tracker) Update(currentPorts []scanner.PortInfo) {
	now := time.Now()
	currentPortMap := make(map[int]scanner.PortInfo)

	// Build map of current ports
	for _, p := range currentPorts {
		currentPortMap[p.Port] = p
	}

	// Check for newly opened ports
	for port, info := range currentPortMap {
		if h, exists := t.history[port]; exists {
			// Port still active, update last seen
			h.LastSeen = now
			if !h.IsActive {
				// Port was closed but now reopened
				h.IsActive = true
				h.OpenCount++
				event := PortEvent{
					Port:      port,
					PID:       info.PID,
					Process:   info.Process,
					EventType: EventPortOpened,
					Timestamp: now,
				}
				h.Events = append(h.Events, event)
				t.addEvent(event)
			}
		} else {
			// New port detected
			h := &PortHistory{
				Port:      port,
				PID:       info.PID,
				Process:   info.Process,
				FirstSeen: now,
				LastSeen:  now,
				IsActive:  true,
				OpenCount: 1,
				Events:    []PortEvent{},
			}
			event := PortEvent{
				Port:      port,
				PID:       info.PID,
				Process:   info.Process,
				EventType: EventPortOpened,
				Timestamp: now,
			}
			h.Events = append(h.Events, event)
			t.history[port] = h
			t.addEvent(event)
		}
	}

	// Check for closed ports
	for port, h := range t.history {
		if h.IsActive {
			if _, stillActive := currentPortMap[port]; !stillActive {
				// Port has closed
				h.IsActive = false
				h.LastSeen = now
				event := PortEvent{
					Port:      port,
					PID:       h.PID,
					Process:   h.Process,
					EventType: EventPortClosed,
					Timestamp: now,
				}
				h.Events = append(h.Events, event)
				t.addEvent(event)
			}
		}
	}

	// Cleanup old histories if needed
	t.cleanup()
}

// GetUptime returns the uptime for a port
func (t *Tracker) GetUptime(port int) time.Duration {
	if h, exists := t.history[port]; exists && h.IsActive {
		return time.Since(h.FirstSeen)
	}
	return 0
}

// GetHistory returns the history for a specific port
func (t *Tracker) GetHistory(port int) *PortHistory {
	return t.history[port]
}

// GetAllHistory returns all port histories
func (t *Tracker) GetAllHistory() []*PortHistory {
	histories := make([]*PortHistory, 0, len(t.history))
	for _, h := range t.history {
		histories = append(histories, h)
	}

	// Sort by last seen (most recent first)
	sort.Slice(histories, func(i, j int) bool {
		return histories[i].LastSeen.After(histories[j].LastSeen)
	})

	return histories
}

// GetRecentEvents returns the most recent events
func (t *Tracker) GetRecentEvents(limit int) []PortEvent {
	if limit <= 0 || limit > len(t.events) {
		limit = len(t.events)
	}

	// Return most recent events (events are already in chronological order)
	start := len(t.events) - limit
	if start < 0 {
		start = 0
	}

	return t.events[start:]
}

// GetStats returns tracking statistics
func (t *Tracker) GetStats() HistoryStats {
	activeCount := 0
	totalEvents := len(t.events)

	for _, h := range t.history {
		if h.IsActive {
			activeCount++
		}
	}

	return HistoryStats{
		TotalPortsTracked: len(t.history),
		ActivePorts:       activeCount,
		TotalEvents:       totalEvents,
	}
}

// HistoryStats contains statistics about port tracking
type HistoryStats struct {
	TotalPortsTracked int
	ActivePorts       int
	TotalEvents       int
}

// addEvent adds an event to the tracker
func (t *Tracker) addEvent(event PortEvent) {
	t.events = append(t.events, event)

	// Trim events if we exceed max
	if len(t.events) > t.maxEvents {
		// Keep only the most recent events
		t.events = t.events[len(t.events)-t.maxEvents:]
	}
}

// cleanup removes old inactive port histories
func (t *Tracker) cleanup() {
	if len(t.history) <= t.maxHistories {
		return
	}

	// Get all inactive histories
	inactive := make([]*PortHistory, 0)
	for _, h := range t.history {
		if !h.IsActive {
			inactive = append(inactive, h)
		}
	}

	// Sort by last seen (oldest first)
	sort.Slice(inactive, func(i, j int) bool {
		return inactive[i].LastSeen.Before(inactive[j].LastSeen)
	})

	// Remove oldest inactive histories
	toRemove := len(t.history) - t.maxHistories
	for i := 0; i < toRemove && i < len(inactive); i++ {
		delete(t.history, inactive[i].Port)
	}
}

// FormatUptime formats a duration as a human-readable string
func FormatUptime(d time.Duration) string {
	if d == 0 {
		return "-"
	}

	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
