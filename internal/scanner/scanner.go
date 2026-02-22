package scanner

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// PortInfo represents information about a listening port
type PortInfo struct {
	Port    int
	PID     int32
	Process string
	Status  string
}

// ScanPorts scans for all active network connections
func ScanPorts() ([]PortInfo, error) {
	conns, err := net.Connections("inet")
	if err != nil {
		return nil, fmt.Errorf("failed to get connections: %w", err)
	}

	// Use a map to deduplicate ports with the same PID
	portMap := make(map[int]PortInfo)

	for _, conn := range conns {
		if conn.Laddr.Port != 0 && conn.Status == "LISTEN" {
			port := int(conn.Laddr.Port)

			// Skip if already have this port
			if _, exists := portMap[port]; exists {
				continue
			}

			pName := "Unknown"
			if conn.Pid != 0 {
				p, err := process.NewProcess(conn.Pid)
				if err == nil {
					pName, _ = p.Name()
				}
			}

			portMap[port] = PortInfo{
				Port:    port,
				PID:     conn.Pid,
				Process: pName,
				Status:  conn.Status,
			}
		}
	}

	// Convert map to slice
	var results []PortInfo
	for _, info := range portMap {
		results = append(results, info)
	}

	return results, nil
}

// KillProcess kills a process by its PID
func KillProcess(pid int32) error {
	if pid == 0 {
		return fmt.Errorf("invalid PID: 0")
	}

	p, err := os.FindProcess(int(pid))
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	err = p.Kill()
	if err != nil {
		return fmt.Errorf("failed to kill process: %w", err)
	}

	return nil
}

// GetProcessName returns the name of a process by PID
func GetProcessName(pid int32) string {
	if pid == 0 {
		return "System"
	}

	p, err := process.NewProcess(pid)
	if err != nil {
		return "Unknown"
	}

	name, err := p.Name()
	if err != nil {
		return "Unknown"
	}

	return name
}

// CheckPortHealth sends a basic HTTP request to check if a port is responsive
func CheckPortHealth(port int) (int, error) {
	// Simple implementation using curl
	cmd := exec.Command("curl", "-s", "-o", "/dev/null", "-w", "%{http_code}",
		fmt.Sprintf("http://localhost:%d", port), "--connect-timeout", "1")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	statusCode, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0, err
	}

	return statusCode, nil
}
