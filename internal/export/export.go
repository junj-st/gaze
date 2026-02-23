package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/junjiang/gaze/internal/scanner"
)

// ExportFormat represents the export file format
type ExportFormat string

const (
	FormatJSON ExportFormat = "json"
	FormatCSV  ExportFormat = "csv"
)

// ExportSnapshot represents a snapshot of ports at a specific time
type ExportSnapshot struct {
	Timestamp time.Time          `json:"timestamp"`
	Ports     []scanner.PortInfo `json:"ports"`
	Summary   ExportSummary      `json:"summary"`
}

// ExportSummary provides aggregate information
type ExportSummary struct {
	TotalPorts      int            `json:"total_ports"`
	UniqueProcesses int            `json:"unique_processes"`
	ProcessCounts   map[string]int `json:"process_counts"`
}

// ToJSON exports the port data to a JSON file
func ToJSON(ports []scanner.PortInfo, outputDir string) (string, error) {
	timestamp := time.Now()
	filename := fmt.Sprintf("gaze-export-%s.json", timestamp.Format("2006-01-02-15-04-05"))
	filepath := filepath.Join(outputDir, filename)

	snapshot := ExportSnapshot{
		Timestamp: timestamp,
		Ports:     ports,
		Summary:   generateSummary(ports),
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	err = os.WriteFile(filepath, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write JSON file: %w", err)
	}

	return filepath, nil
}

// ToCSV exports the port data to a CSV file
func ToCSV(ports []scanner.PortInfo, outputDir string) (string, error) {
	timestamp := time.Now()
	filename := fmt.Sprintf("gaze-export-%s.csv", timestamp.Format("2006-01-02-15-04-05"))
	filepath := filepath.Join(outputDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Port", "PID", "Process", "Status", "Timestamp"}
	if err := writer.Write(header); err != nil {
		return "", fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data
	timestampStr := timestamp.Format(time.RFC3339)
	for _, p := range ports {
		record := []string{
			fmt.Sprintf("%d", p.Port),
			fmt.Sprintf("%d", p.PID),
			p.Process,
			p.Status,
			timestampStr,
		}
		if err := writer.Write(record); err != nil {
			return "", fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	return filepath, nil
}

// generateSummary creates a summary of the port data
func generateSummary(ports []scanner.PortInfo) ExportSummary {
	processCounts := make(map[string]int)

	for _, p := range ports {
		processCounts[p.Process]++
	}

	return ExportSummary{
		TotalPorts:      len(ports),
		UniqueProcesses: len(processCounts),
		ProcessCounts:   processCounts,
	}
}
