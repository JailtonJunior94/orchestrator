package aispecharness

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateVersion_ValidVersion(t *testing.T) {
	dir := t.TempDir()
	versionFile := filepath.Join(dir, "VERSION")

	cmd := newUpdateVersionCmd()
	if err := cmd.Flags().Set("version", "1.2.3"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("version-file", versionFile); err != nil {
		t.Fatal(err)
	}

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	content, err := os.ReadFile(versionFile)
	if err != nil {
		t.Fatalf("could not read version file: %v", err)
	}
	if string(content) != "1.2.3\n" {
		t.Fatalf("expected '1.2.3\\n', got %q", string(content))
	}
}

func TestUpdateVersion_InvalidVersions(t *testing.T) {
	cases := []string{"v1.0.0", "1.0", "abc", "1.2.3.4", ""}

	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			dir := t.TempDir()
			cmd := newUpdateVersionCmd()
			if err := cmd.Flags().Set("version", tc); err != nil {
				t.Fatal(err)
			}
			if err := cmd.Flags().Set("version-file", filepath.Join(dir, "VERSION")); err != nil {
				t.Fatal(err)
			}

			err := cmd.RunE(cmd, nil)
			if err == nil {
				t.Fatalf("expected error for version %q, got nil", tc)
			}
		})
	}
}

func TestUpdateVersion_CustomVersionFile(t *testing.T) {
	dir := t.TempDir()
	versionFile := filepath.Join(dir, "MY_VERSION")

	cmd := newUpdateVersionCmd()
	if err := cmd.Flags().Set("version", "2.0.0"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("version-file", versionFile); err != nil {
		t.Fatal(err)
	}

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	content, err := os.ReadFile(versionFile)
	if err != nil {
		t.Fatalf("could not read version file: %v", err)
	}
	if string(content) != "2.0.0\n" {
		t.Fatalf("expected '2.0.0\\n', got %q", string(content))
	}
}
