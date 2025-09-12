package helpers

import (
    "bufio"
	"encoding/json"
	"fmt"
    "os"
    "regexp"
	"strconv"
    "strings"
	"time"
)

// TopOutput holds the complete parsed output of the `top` command.
type TopOutput struct {
	Summary   TopSummary
	Processes []Process
}

// TopSummary holds the system-wide information from the top of the `top` output.
type TopSummary struct {
	Timestamp     time.Time
	Uptime        string // e.g., "up 14:31"
	TotalUsers    int
	LoadAverage   []float64 // Last 1, 5, and 15 minutes
	TaskSummary   TaskSummary
	CPUSummary    CPUSummary
	MemorySummary MemorySummary
	SwapSummary   SwapSummary
}

// TaskSummary holds statistics on the system tasks (processes).
type TaskSummary struct {
	Total    int
	Running  int
	Sleeping int
	Stopped  int
	Zombie   int
}

// CPUSummary holds detailed CPU usage statistics.
type CPUSummary struct {
	User              float64
	System            float64
	Nice              float64
	Idle              float64
	IOWait            float64
	HardwareInterrupt float64
	SoftwareInterrupt float64
	Steal             float64
}

// MemorySummary holds system memory usage statistics.
type MemorySummary struct {
	Total     int64
	Free      int64
	Used      int64
	BuffCache int64
}

// SwapSummary holds system swap space usage statistics.
type SwapSummary struct {
	Total int64
	Free  int64
	Used  int64
	Avail int64
}

// Process holds the information for a single process entry from the `top` output.
type Process struct {
	PID      int
	User     string
	Priority int
	Nice     int
	Virt     uint64  // Virtual Memory
	Res      uint64  // Resident Memory
	Shr      uint64  // Shared Memory
	State    string // e.g., "S" for sleep
	CPU      float64
	Memory   float64
	Time     string // e.g., "00:00.12"
	Command  string
}

func LoadProcessesDataFromCmdOutput(data string) ([]byte, error) {
	//Store the data in process struct
    var output TopOutput
	scanner := bufio.NewScanner(strings.NewReader(data))

	for i := 0; i < 7 && scanner.Scan(); i++ {
		line := scanner.Text()
		if strings.Contains(line, "top -") {
			// Extract uptime and load average
			parts := strings.Split(line, ",")
			if len(parts) >= 3 {
				output.Summary.Uptime = strings.TrimSpace(parts[0])
				output.Summary.LoadAverage = parseLoadAverage(parts[2])
			}
		} else if strings.Contains(line, "Tasks:") {
			// Extract task summary
			output.Summary.TaskSummary = parseTaskSummary(line)
		} else if strings.Contains(line, "%Cpu(s):") {
			// Extract CPU summary
			output.Summary.CPUSummary = parseCPUSummary(line)
		} else if strings.Contains(line, "MiB Mem :") {
			// Extract memory summary
			output.Summary.MemorySummary = parseMemorySummary(line)
		} else if strings.Contains(line, "MiB Swap:") {
			// Extract swap summary
			output.Summary.SwapSummary = parseSwapSummary(line)
		}

		if strings.Contains(line, "PID") && strings.Contains(line, "COMMAND") {
			//Headers
			break
		}
	}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		process, err := parseProcessLine(line)
		if err == nil {
			output.Processes = append(output.Processes, process)
		}
	}
    return json.Marshal(output)
}

// Helper functions for parsing
func parseLoadAverage(s string) []float64 {
	var loads []float64
	loadStr := strings.TrimPrefix(strings.TrimSpace(s), "load average:")
	loadParts := strings.Split(loadStr, ",")
	for _, p := range loadParts {
		val, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err == nil {
			loads = append(loads, val)
		}
	}
	return loads
}

func parseTaskSummary(s string) TaskSummary {
	var summary TaskSummary
	parts := strings.Split(s, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.Contains(p, "total") {
			summary.Total, _ = strconv.Atoi(extractNumber(p))
		} else if strings.Contains(p, "running") {
			summary.Running, _ = strconv.Atoi(extractNumber(p))
		} else if strings.Contains(p, "sleeping") {
			summary.Sleeping, _ = strconv.Atoi(extractNumber(p))
		} else if strings.Contains(p, "stopped") {
			summary.Stopped, _ = strconv.Atoi(extractNumber(p))
		} else if strings.Contains(p, "zombie") {
			summary.Zombie, _ = strconv.Atoi(extractNumber(p))
		}
	}
	return summary
}

func parseCPUSummary(s string) CPUSummary {
	var summary CPUSummary
	// A more robust solution might use regex, but this is sufficient for a fixed format.
	s = strings.TrimPrefix(s, "%Cpu(s):")
	parts := strings.Fields(s)
	for i := 0; i < len(parts)-1; i += 2 {
		val, err := strconv.ParseFloat(parts[i], 64)
		if err != nil {
			continue
		}
		switch parts[i+1] {
		case "us,":
			summary.User = val
		case "sy,":
			summary.System = val
		case "ni,":
			summary.Nice = val
		case "id,":
			summary.Idle = val
		case "wa,":
			summary.IOWait = val
		case "hi,":
			summary.HardwareInterrupt = val
		case "si,":
			summary.SoftwareInterrupt = val
		case "st":
			summary.Steal = val
		}
	}
	return summary
}

func extractNumber(data string) string {
    re := regexp.MustCompile(`\d+`)
    foundNumbers := re.FindAllString(data, -1)
    if len(foundNumbers) > 0 {
        return foundNumbers[0]
    }
    return ""
}

func parseMemorySummary(s string) MemorySummary {
	var summary MemorySummary
	// A simple approach based on splitting, but may be fragile
	parts := strings.Split(s, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.Contains(p, "total") {
			summary.Total, _ = strconv.ParseInt(extractNumber(p), 10, 64)
		} else if strings.Contains(p, "free") {
			summary.Free, _ = strconv.ParseInt(extractNumber(p), 10, 64)
		} else if strings.Contains(p, "used") {
			summary.Used, _ = strconv.ParseInt(extractNumber(p), 10, 64)
		} else if strings.Contains(p, "buff/cache") {
			summary.BuffCache, _ = strconv.ParseInt(extractNumber(p), 10, 64)
		}
	}
	return summary
}

func parseSwapSummary(s string) SwapSummary {
	var summary SwapSummary
	parts := strings.Split(s, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.Contains(p, "total") {
			summary.Total, _ = strconv.ParseInt(extractNumber(p), 10, 64)
		} else if strings.Contains(p, "free") {
			summary.Free, _ = strconv.ParseInt(extractNumber(p), 10, 64)
		} else if strings.Contains(p, "used") {
			summary.Used, _ = strconv.ParseInt(extractNumber(p), 10, 64)
		} else if strings.Contains(p, "avail") {
			summary.Avail, _ = strconv.ParseInt(extractNumber(p), 10, 64)
		}
	}
	return summary
}

func parseMemoryValue(data string) uint64 {
    pageSize := uint64(os.Getpagesize()) / 1024
    parsedValue, _ := strconv.ParseUint(data, 10, 64)
    return parsedValue * pageSize
}

func parseProcessLine(s string) (Process, error) {
	var proc Process
	fields := strings.Fields(s)
	if len(fields) < 12 {
		return proc, fmt.Errorf("not enough fields in process line: %s", s)
	}

	proc.PID, _ = strconv.Atoi(fields[0])
	proc.User = fields[1]
	proc.Priority, _ = strconv.Atoi(fields[2])
	proc.Nice, _ = strconv.Atoi(fields[3])
	proc.Virt = parseMemoryValue(fields[4])
	proc.Res = parseMemoryValue(fields[5])
	proc.Shr = parseMemoryValue(fields[6])
	proc.State = fields[7]
	proc.CPU, _ = strconv.ParseFloat(fields[8], 64)
	proc.Memory, _ = strconv.ParseFloat(fields[9], 64)
	proc.Time = fields[10]
	proc.Command = strings.Join(fields[11:], " ")

	return proc, nil
}
