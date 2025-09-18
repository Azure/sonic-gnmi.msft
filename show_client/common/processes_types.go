package common

// Struct to hold individual process details
type TopProcessData struct {
	PID     string `json:"pid"`
	User    string `json:"user"`
	PR      string `json:"pr"`
	NI      string `json:"ni"`
	VIRT    string `json:"virt"`
	RES     string `json:"res"`
	SHR     string `json:"shr"`
	S       string `json:"s"`
	CPU     string `json:"cpu"`
	MEM     string `json:"mem"`
	TIME    string `json:"time"`
	Command string `json:"command"`
}

// Struct to hold the full snapshot
type TopProcessCompleteResponse struct {
	Uptime      string           `json:"uptime"`
	Tasks       string           `json:"tasks"`
	CPUUsage    string           `json:"cpu_usage"`
	MemoryUsage string           `json:"memory_usage"`
	SwapUsage   string           `json:"swap_usage"`
	Processes   []TopProcessData `json:"processes"`
}
