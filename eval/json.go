// Package eval contains Phase 1 migration skeleton types for the new architecture.
// It currently provides minimal runnable reports and in-memory aggregation only.
package eval

import "encoding/json"

// EncodeRunReport encodes a run report as pretty JSON.
func EncodeRunReport(report RunReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

// EncodePairReport encodes a pair report as pretty JSON.
func EncodePairReport(report PairReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}
