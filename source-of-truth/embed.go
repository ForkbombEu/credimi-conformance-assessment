package sourceoftruth

import "embed"

// FS contains the bundled source-of-truth files used by the conformance library
// when callers do not provide an external source directory.
//
//go:embed credimi-flat-conformance-test-list-v1.1.md credimi-conformance-aggregation-taxonomy-v1.1.yaml
var FS embed.FS
