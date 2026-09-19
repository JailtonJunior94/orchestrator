package tracking

import (
	"fmt"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func BenchmarkFakeFileSystem_WriteFile_Baseline(b *testing.B) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	data := []byte("---\nname: skill\nversion: 1.0.0\ndescription: benchmark fixture.\n---\n")

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		path := fmt.Sprintf("/project/file-%d.md", i)
		if err := ffs.WriteFile(path, data); err != nil {
			b.Fatalf("WriteFile: %v", err)
		}
	}
}

func BenchmarkTracker_WriteFile_WithChecksum(b *testing.B) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	tr := New(ffs, "/project", ".ai_spec_harness.json")
	data := []byte("---\nname: skill\nversion: 1.0.0\ndescription: benchmark fixture.\n---\n")

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		path := fmt.Sprintf("/project/file-%d.md", i)
		if err := tr.WriteFile(path, data); err != nil {
			b.Fatalf("WriteFile: %v", err)
		}
	}
}
