package jenkins

// Job is one entry from `jk job ls --json`.
type Job struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Color string `json:"color"`
}

// Run is one entry from `jk run ls --json`.
type Run struct {
	ID         string `json:"id"`
	Number     int    `json:"number"`
	Status     string `json:"status"`
	Result     string `json:"result"`
	DurationMs int64  `json:"durationMs"`
	StartTime  string `json:"startTime"`
	Branch     string `json:"branch"`
	Commit     string `json:"commit"`
	URL        string `json:"url"`
	QueueID    int64  `json:"queueId"`
}

// RunList is the top-level shape of `jk run ls --json`.
type RunList struct {
	SchemaVersion string `json:"schemaVersion"`
	Items         []Run  `json:"items"`
	NextCursor    string `json:"nextCursor"`
}

// Param describes one job parameter definition from `jk run params --json`.
type Param struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Default      string   `json:"default"`
	IsSecret     bool     `json:"isSecret"`
	SampleValues []string `json:"sampleValues"`
	Frequency    int      `json:"frequency"`
}

// ParamList is the top-level shape of `jk run params --json`.
type ParamList struct {
	JobPath    string  `json:"jobPath"`
	Source     string  `json:"source"`
	Parameters []Param `json:"parameters"`
}

// AuthStatus mirrors `jk auth status` (parsed from plain text, no --json support).
type AuthStatus struct {
	Context  string
	URL      string
	Username string
}
