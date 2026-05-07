package session

// ToolStat is the per-tool aggregation row produced by ComputeReport.
//
// Token and byte fields are summed across every `events` row whose payload
// carries the matching tool name. Counts can legitimately be zero when the
// hook handler did not record token/byte fields for a given call.
type ToolStat struct {
	ToolName  string
	CallCount int
	TokensIn  int64
	TokensOut int64
	BytesIn   int64
	BytesOut  int64
}

// FullReport is the analytics aggregation over a session's `events` table.
//
// SavingsRatio is clamped to [0, 1]; values outside that range indicate a
// data anomaly and are flattened to 0 rather than propagated.
type FullReport struct {
	Tools               []ToolStat
	TotalBytesProcessed int64
	TotalBytesReturned  int64
	SavingsRatio        float64
	UptimeMs            int64
}
