package scanner

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// PortInfo represents information about a listening port
type PortInfo struct {
	Port       int
	PID        int32
	Process    string
	Status     string
	HTTPStatus int           // HTTP response status code (0 if not checked)
	Latency    time.Duration // Response latency
	CPUPercent float64       // CPU usage percentage
	MemoryMB   float64       // Memory usage in MB
	Selected   bool          // For multi-select mode
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
			var cpuPercent, memoryMB float64
			if conn.Pid != 0 {
				p, err := process.NewProcess(conn.Pid)
				if err == nil {
					pName, _ = p.Name()
					// Get CPU and memory usage
					cpuPercent, _ = p.CPUPercent()
					memInfo, err := p.MemoryInfo()
					if err == nil {
						memoryMB = float64(memInfo.RSS) / 1024 / 1024
					}
				}
			}

			portInfo := PortInfo{
				Port:       port,
				PID:        conn.Pid,
				Process:    pName,
				Status:     conn.Status,
				CPUPercent: cpuPercent,
				MemoryMB:   memoryMB,
			}

			// Check HTTP health for common web ports
			if isWebPort(port) {
				statusCode, latency := checkHTTPHealth(port)
				portInfo.HTTPStatus = statusCode
				portInfo.Latency = latency
			}

			portMap[port] = portInfo
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

// GetPortType categorizes a port into well-known, registered, or dynamic
func GetPortType(port int) string {
	if port < 1024 {
		return "well-known"
	} else if port < 49152 {
		return "registered"
	}
	return "dynamic"
}

// isWebPort checks if a port is commonly used for web services
func isWebPort(port int) bool {
	commonWebPorts := []int{80, 443, 8080, 8000, 8443, 3000, 5000, 3001, 4200, 5173, 8888, 9000}
	for _, p := range commonWebPorts {
		if port == p {
			return true
		}
	}
	return false
}

// checkHTTPHealth performs HTTP health check with latency measurement
func checkHTTPHealth(port int) (int, time.Duration) {
	client := &http.Client{
		Timeout: 1 * time.Second,
	}

	url := fmt.Sprintf("http://localhost:%d", port)
	start := time.Now()
	resp, err := client.Get(url)
	latency := time.Since(start)

	if err != nil {
		return 0, 0
	}
	defer resp.Body.Close()

	return resp.StatusCode, latency
}

// KillMultipleProcesses kills multiple processes by their PIDs
func KillMultipleProcesses(pids []int32) error {
	var errors []error
	for _, pid := range pids {
		if err := KillProcess(pid); err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("failed to kill %d processes", len(errors))
	}
	return nil
}
