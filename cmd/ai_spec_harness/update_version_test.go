package aispecharness

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateVersion_ValidVersion(t *testing.T) {
	dir := t.TempDir()
	versionFile := filepath.Join(dir, "VERSION")

	updateVersionVersion = "1.2.3"
	updateVersionVersionFile = versionFile
	t.Cleanup(func() {
		updateVersionVersion = ""
		updateVersionVersionFile = "VERSION"
	})

	if err := updateVersionCmd.RunE(updateVersionCmd, nil); err != nil {
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
			updateVersionVersion = tc
			updateVersionVersionFile = filepath.Join(dir, "VERSION")
			t.Cleanup(func() {
				updateVersionVersion = ""
				updateVersionVersionFile = "VERSION"
			})

			err := updateVersionCmd.RunE(updateVersionCmd, nil)
			if err == nil {
				t.Fatalf("expected error for version %q, got nil", tc)
			}
		})
	}
}

func TestUpdateVersion_CustomVersionFile(t *testing.T) {
	dir := t.TempDir()
	versionFile := filepath.Join(dir, "MY_VERSION")

	updateVersionVersion = "2.0.0"
	updateVersionVersionFile = versionFile
	t.Cleanup(func() {
		updateVersionVersion = ""
		updateVersionVersionFile = "VERSION"
	})

	if err := updateVersionCmd.RunE(updateVersionCmd, nil); err != nil {
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
