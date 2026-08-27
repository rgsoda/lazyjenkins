package jenkins

// Job is one entry from `jk job ls --json`. This can be a real buildable
// job OR a folder / multibranch-pipeline container: Jenkins only assigns a
// ball color to actual buildable job types (even "notbuilt" for one that's
// never run), so an empty Color means it's a container with no runs of its
// own — its real (per-branch, typically) jobs are one level deeper.
type Job struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Color string `json:"color"`
}

// IsFolder reports whether this entry is a folder/container rather than a
// runnable job — see the Job doc comment.
func (j Job) IsFolder() bool { return j.Color == "" }

// Run is one entry from `jk run ls --json`. Branch/Commit reflect the SCM
// checkout of the job's own Jenkinsfile, which for parameterized "deploy
// anything" jobs is unrelated to what was actually deployed — that lives in
// Fields.Parameters (e.g. a "gitBranch" build parameter) instead.
type Run struct {
	ID         string     `json:"id"`
	Number     int        `json:"number"`
	Status     string     `json:"status"`
	Result     string     `json:"result"`
	DurationMs int64      `json:"durationMs"`
	StartTime  string     `json:"startTime"`
	Branch     string     `json:"branch"`
	Commit     string     `json:"commit"`
	URL        string     `json:"url"`
	QueueID    int64      `json:"queueId"`
	Fields     *RunFields `json:"fields,omitempty"`
}

// RunFields holds the optional extra data requested via `--select`.
type RunFields struct {
	Parameters map[string]string `json:"parameters,omitempty"`
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

// Context is one configured Jenkins connection, from `jk context ls`
// (parsed from plain text — like auth status, this command has no --json
// support either).
type Context struct {
	Name   string
	URL    string
	Active bool
}
