package scanner

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// PortInfo represents information about a listening port
type PortInfo struct {
	Port        int
	PID         int32
	Process     string
	Status      string
	HTTPStatus  int           // HTTP status code (0 if not HTTP/failed)
	Latency     time.Duration // Response latency
	CPUPercent  float64       // CPU usage percentage
	MemoryMB    float64       // Memory usage in MB
	Selected    bool          // For multi-select kill
}

// PortType represents the category of a port
type PortType int

const (
	WellKnownPort   PortType = iota // 0-1023
	RegisteredPort                  // 1024-49151
	DynamicPort                     // 49152-65535
)

// GetPortType returns the type of port based on its number
func GetPortType(port int) PortType {
	if port >= 0 && port <= 1023 {
		return WellKnownPort
	} else if port >= 1024 && port <= 49151 {
		return RegisteredPort
	}
	return DynamicPort
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
			cpuPercent := 0.0
			memoryMB := 0.0

			if conn.Pid != 0 {
				p, err := process.NewProcess(conn.Pid)
				if err == nil {
					pName, _ = p.Name()
					
					// Get CPU percentage
					cpuPercent, _ = p.CPUPercent()
					
					// Get memory info
					memInfo, err := p.MemoryInfo()
					if err == nil {
						memoryMB = float64(memInfo.RSS) / 1024 / 1024 // Convert to MB
					}
				}
			}

			info := PortInfo{
				Port:       port,
				PID:        conn.Pid,
				Process:    pName,
				Status:     conn.Status,
				CPUPercent: cpuPercent,
				MemoryMB:   memoryMB,
				Selected:   false,
			}

			// Check HTTP health for common web ports
			if isWebPort(port) {
				status, latency := checkHTTPHealth(port)
				info.HTTPStatus = status
				info.Latency = latency
			}

			portMap[port] = info
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

// KillMultipleProcesses kills multiple processes by their PIDs
func KillMultipleProcesses(pids []int32) []error {
	var errors []error
	
	for _, pid := range pids {
		if err := KillProcess(pid); err != nil {
			errors = append(errors, err)
		}
	}
	
	return errors
}

// isWebPort checks if a port is commonly used for HTTP/HTTPS
func isWebPort(port int) bool {
	webPorts := map[int]bool{
		80:   true, // HTTP
		443:  true, // HTTPS
		3000: true, // Common dev port
		3001: true,
		4200: true, // Angular
		5000: true, // Flask, etc
		5173: true, // Vite
		8000: true, // Common dev port
		8080: true, // Common alt HTTP
		8443: true, // Common alt HTTPS
		8888: true,
		9000: true,
	}
	return webPorts[port]
}

// checkHTTPHealth sends an HTTP request to check if a port is responsive
func checkHTTPHealth(port int) (int, time.Duration) {
	client := &http.Client{
		Timeout: 1 * time.Second,
	}
	
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost:%d", port), nil)
	if err != nil {
		return 0, 0
	}
	
	resp, err := client.Do(req)
	latency := time.Since(start)
	
	if err != nil {
		return 0, 0
	}
	defer resp.Body.Close()
	
	return resp.StatusCode, latency
}
