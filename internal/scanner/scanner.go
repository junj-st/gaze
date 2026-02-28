package scanner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// PortInfo represents information about a listening port
type PortInfo struct {
	Port          int
	PID           int32
	Process       string
	Status        string
	HTTPStatus    int           // HTTP response status code (0 if not checked)
	Latency       time.Duration // Response latency
	CPUPercent    float64       // CPU usage percentage
	MemoryMB      float64       // Memory usage in MB
	Selected      bool          // For multi-select mode
	ContainerID   string        // Docker container ID (short form)
	ContainerName string        // Docker container name
	IsContainer   bool          // Whether this process is in a container
}

// DockerContainer represents a Docker container
type DockerContainer struct {
	ID     string `json:"ID"`
	Name   string `json:"Names"`
	Image  string `json:"Image"`
	Ports  string `json:"Ports"`
	Status string `json:"Status"`
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
			var containerID, containerName string
			var isContainer bool

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

					// Check if process is in a Docker container
					containerID, containerName, isContainer = getContainerInfo(conn.Pid)
				}
			}

			portInfo := PortInfo{
				Port:          port,
				PID:           conn.Pid,
				Process:       pName,
				Status:        conn.Status,
				CPUPercent:    cpuPercent,
				MemoryMB:      memoryMB,
				ContainerID:   containerID,
				ContainerName: containerName,
				IsContainer:   isContainer,
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

// getContainerInfo checks if a PID is running in a Docker container
// Returns containerID (short), containerName, and isContainer bool
func getContainerInfo(pid int32) (string, string, bool) {
	// Check if Docker is available
	if !isDockerAvailable() {
		return "", "", false
	}

	// Read cgroup file to detect container ID
	containerID := getContainerIDFromCgroup(pid)
	if containerID == "" {
		return "", "", false
	}

	// Get container name from Docker
	containerName := getContainerNameByID(containerID)

	// Return short container ID (first 12 chars)
	shortID := containerID
	if len(containerID) > 12 {
		shortID = containerID[:12]
	}

	return shortID, containerName, true
}

// isDockerAvailable checks if Docker CLI is available
func isDockerAvailable() bool {
	cmd := exec.Command("docker", "version")
	err := cmd.Run()
	return err == nil
}

// getContainerIDFromCgroup reads the cgroup file to extract container ID
func getContainerIDFromCgroup(pid int32) string {
	cgroupPath := fmt.Sprintf("/proc/%d/cgroup", pid)
	data, err := os.ReadFile(cgroupPath)
	if err != nil {
		return ""
	}

	// Look for docker container ID in cgroup
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.Contains(line, "docker") {
			// Extract container ID from paths like:
			// 0::/docker/1234567890abcdef...
			// 0::/system.slice/docker-1234567890abcdef.scope
			parts := strings.Split(line, "/")
			for _, part := range parts {
				if strings.HasPrefix(part, "docker-") {
					// Remove "docker-" prefix and ".scope" suffix
					id := strings.TrimPrefix(part, "docker-")
					id = strings.TrimSuffix(id, ".scope")
					return id
				}
				// Check if part is a long hex string (container ID)
				if len(part) == 64 {
					return part
				}
			}
		}
	}

	return ""
}

// getContainerNameByID gets the container name using Docker CLI
func getContainerNameByID(containerID string) string {
	cmd := exec.Command("docker", "inspect", "--format={{.Name}}", containerID)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	name := strings.TrimSpace(string(output))
	// Remove leading slash from container name
	name = strings.TrimPrefix(name, "/")
	return name
}

// ListDockerContainers returns all running Docker containers
func ListDockerContainers() ([]DockerContainer, error) {
	if !isDockerAvailable() {
		return nil, fmt.Errorf("docker is not available")
	}

	cmd := exec.Command("docker", "ps", "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var containers []DockerContainer
	lines := bytes.Split(output, []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var container DockerContainer
		if err := json.Unmarshal(line, &container); err == nil {
			containers = append(containers, container)
		}
	}

	return containers, nil
}

// StopContainer stops a Docker container by ID or name
func StopContainer(containerID string) error {
	if !isDockerAvailable() {
		return fmt.Errorf("docker is not available")
	}

	cmd := exec.Command("docker", "stop", containerID)
	return cmd.Run()
}

// RestartContainer restarts a Docker container by ID or name
func RestartContainer(containerID string) error {
	if !isDockerAvailable() {
		return fmt.Errorf("docker is not available")
	}

	cmd := exec.Command("docker", "restart", containerID)
	return cmd.Run()
}
