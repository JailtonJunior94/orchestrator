package output_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

func newTestPrinter(verbose bool) (*output.Printer, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	p := &output.Printer{Out: out, Err: errBuf, Verbose: verbose}
	return p, out, errBuf
}

func TestInfo(t *testing.T) {
	p, out, _ := newTestPrinter(false)
	p.Info("hello %s", "world")
	if got := out.String(); got != "hello world\n" {
		t.Errorf("Info() = %q, want %q", got, "hello world\n")
	}
}

func TestStep(t *testing.T) {
	p, out, _ := newTestPrinter(false)
	p.Step("doing thing")
	got := out.String()
	if !strings.HasPrefix(got, "-> ") {
		t.Errorf("Step() output %q should start with '-> '", got)
	}
	if !strings.Contains(got, "doing thing") {
		t.Errorf("Step() output %q should contain 'doing thing'", got)
	}
}

func TestDebug_verbose(t *testing.T) {
	p, out, _ := newTestPrinter(true)
	p.Debug("internal state %d", 42)
	got := out.String()
	if !strings.Contains(got, "[debug]") {
		t.Errorf("Debug(verbose=true) = %q, want [debug] prefix", got)
	}
	if !strings.Contains(got, "42") {
		t.Errorf("Debug(verbose=true) = %q, want '42'", got)
	}
}

func TestDebug_silent(t *testing.T) {
	p, out, _ := newTestPrinter(false)
	p.Debug("should not appear")
	if got := out.String(); got != "" {
		t.Errorf("Debug(verbose=false) produced output %q, want empty", got)
	}
}

func TestWarn(t *testing.T) {
	p, _, errBuf := newTestPrinter(false)
	p.Warn("watch out")
	if got := errBuf.String(); !strings.HasPrefix(got, "AVISO:") {
		t.Errorf("Warn() = %q, want 'AVISO:' prefix", got)
	}
}

func TestError(t *testing.T) {
	p, _, errBuf := newTestPrinter(false)
	p.Error("something broke")
	if got := errBuf.String(); !strings.HasPrefix(got, "ERRO:") {
		t.Errorf("Error() = %q, want 'ERRO:' prefix", got)
	}
}

func TestDryRun(t *testing.T) {
	p, out, _ := newTestPrinter(false)
	p.DryRun("would copy file")
	if got := out.String(); !strings.Contains(got, "[dry-run]") {
		t.Errorf("DryRun() = %q, want '[dry-run]' tag", got)
	}
}

func TestStatus(t *testing.T) {
	p, out, _ := newTestPrinter(false)
	p.Status("OK", "bugfix", "v1.2")
	got := out.String()
	if !strings.Contains(got, "OK") || !strings.Contains(got, "bugfix") || !strings.Contains(got, "v1.2") {
		t.Errorf("Status() = %q, want OK, bugfix, v1.2", got)
	}
}

func TestNew_defaults(t *testing.T) {
	p := output.New(true)
	if p == nil {
		t.Fatal("New() returned nil")
	}
	if !p.Verbose {
		t.Error("New(true) should set Verbose=true")
	}
	if p.Out == nil || p.Err == nil {
		t.Error("New() should set Out and Err writers")
	}
}
