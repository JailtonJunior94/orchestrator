package bugschema

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzValidateBugReport verifica que nenhum JSON de entrada causa panic no validador.
func FuzzValidateBugReport(f *testing.F) {
	// Seed corpus baseado no bug-schema.json
	f.Add([]byte(`[{"id":"BUG-001","severity":"high","file":"main.go","line":42,"reproduction":"call foo()","expected":"returns nil","actual":"panics"}]`))
	f.Add([]byte(`[{"id":"B","severity":"critical","file":"f.go","line":1,"reproduction":"r","expected":"e","actual":"a"}]`))
	f.Add([]byte(`[{"id":"BUG-002","severity":"low","file":"handler.go","line":10,"reproduction":"send request","expected":"200 OK","actual":"500 error"}]`))
	// Casos adversariais
	f.Add([]byte(`[]`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(``))
	f.Add([]byte(`not valid json`))
	f.Add([]byte(`[{}]`))
	f.Add([]byte(`[{"severity":"extreme"}]`))

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		schemaPath := filepath.Join(dir, "bug-schema.json")
		bugsPath := filepath.Join(dir, "bugs.json")

		if err := os.WriteFile(schemaPath, []byte(schemaFixture), 0600); err != nil {
			t.Skip("falha ao criar schema temporario")
		}
		if err := os.WriteFile(bugsPath, data, 0600); err != nil {
			t.Skip("falha ao criar arquivo de bugs temporario")
		}

		// JSONs invalidos devem retornar erro sem panic
		_ = Validate(bugsPath, schemaPath)
	})
}
