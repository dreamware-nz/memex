package routing

import _ "embed"

//go:embed compression.md
var compressionMD string

// OutputCompressionDoc returns the bundled caveman-mode output compression directive.
func OutputCompressionDoc() string { return compressionMD }
