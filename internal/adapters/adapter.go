package adapters

// DiagnosticResult is one entry in a doctor-style validation report.
//
// Status is "pass" or "fail". Fix is an optional human-readable hint
// (e.g. a CLI command) that resolves a "fail" check; it is empty for
// "pass" results.
type DiagnosticResult struct {
	Check   string
	Status  string
	Message string
	Fix     string
}

// Adapter is the contract every host-platform integration implements.
//
// Implementations patch a platform's user settings file and any platform
// plugin manifests so that memex hooks fire on the right events. All
// methods MUST be idempotent and MUST preserve user-authored entries.
type Adapter interface {
	SettingsPath() string
	SkillsPath() string
	Install(binaryPath string) error
	Uninstall(binaryPath string) error
	Validate(binaryPath string) []DiagnosticResult
}
