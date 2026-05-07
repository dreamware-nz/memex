package session

// Snapshot is the typed aggregation of `events` rows for a single session.
// It is produced by BuildSnapshot and consumed by RenderSnapshot.
type Snapshot struct {
	ToolCounts    map[string]int
	FilesRead     []string
	FilesWritten  []string
	ErrorCount    int
	FirstEventTS  int64
	LastEventTS   int64
}
