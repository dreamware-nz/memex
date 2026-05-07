package tools

// Thresholds matched verbatim with context-mode/src/server.ts.
const (
	// IntentSearchThreshold gates the intent-driven search path: when an
	// intent argument is set and combined output exceeds this size, the
	// tool indexes and returns matched section previews instead of raw
	// output.
	IntentSearchThreshold = 5_000
	// LargeOutputThreshold gates the auto-index path: when no intent is
	// supplied and output exceeds this size, the tool indexes and returns
	// a pointer-style summary referencing the indexed source label.
	LargeOutputThreshold = 102_400
)
