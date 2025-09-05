import "time"

// TopOutput holds the complete parsed output of the `top` command.
type TopOutput struct {
    Summary   TopSummary
    Processes []Process
}

// TopSummary holds the system-wide information from the top of the `top` output.
type TopSummary struct {
    Timestamp      time.Time
    Uptime         string // e.g., "up 14:31"
    TotalUsers     int
    LoadAverage    []float64 // Last 1, 5, and 15 minutes
    TaskSummary    TaskSummary
    CPUSummary     CPUSummary
    MemorySummary  MemorySummary
    SwapSummary    SwapSummary
}

// TaskSummary holds statistics on the system tasks (processes).
type TaskSummary struct {
    Total   int
    Running int
    Sleeping int
    Stopped int
    Zombie  int
}

// CPUSummary holds detailed CPU usage statistics.
type CPUSummary struct {
    User float64
    System float64
    Nice float64
    Idle float64
    IOWait float64
    HardwareInterrupt float64
    SoftwareInterrupt float64
    Steal float64
}

// MemorySummary holds system memory usage statistics.
type MemorySummary struct {
    Total int64
    Free int64
    Used int64
    BuffCache int64
}

// SwapSummary holds system swap space usage statistics.
type SwapSummary struct {
    Total int64
    Free int64
    Used int64
    Avail int64
}

// Process holds the information for a single process entry from the `top` output.
type Process struct {
    PID         int
    User        string
    Priority    int
    Nice        int
    Virt        int64 // Virtual Memory
    Res         int64 // Resident Memory
    Shr         int64 // Shared Memory
    State       string // e.g., "S" for sleep
    CPU         float64
    Memory      float64
    Time        string // e.g., "00:00.12"
    Command     string
}
