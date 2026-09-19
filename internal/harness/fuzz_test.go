package harness

import "testing"

func FuzzParseHarnessContract(f *testing.F) {
	f.Add([]byte("version: 1\ngit:\n  auto_commit: false\n  auto_push: false\napproval:\n  require_for_destructive_operations: true\nquality:\n  require_tests: true\n  require_lint: true\nevidence:\n  require_execution_report: true\nskills:\n  discovery_mode: declared\n"))
	f.Add([]byte("version: 1\n"))
	f.Add([]byte("version: 2\ngit:\n  auto_commit: true\n"))
	f.Add([]byte("{}"))
	f.Add([]byte(""))
	f.Add([]byte("- just\n- a\n- list\n"))
	f.Add([]byte("version: 1\ngit: not-a-mapping\n"))
	f.Add([]byte("1: weird-numeric-key\nversion: 1\n"))
	f.Add([]byte("version: 1\nunknown_field: true\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		raw, err := NewYAMLBridge().Decode(data)
		if err != nil {
			return
		}
		_ = NewSchemaValidator().Validate(raw)
	})
}
