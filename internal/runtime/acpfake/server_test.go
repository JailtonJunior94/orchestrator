package acpfake_test

import (
	"context"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
)

func TestAcpfake_Scripted(t *testing.T) {
	t.Parallel()

	script := acpfake.NewScript().
		AppendAgentMessage("olá mundo").
		AppendToolCall("tc_1", "read_file").
		AppendToolCallUpdate("tc_1", "completed").
		AppendAgentThought("raciocínio interno").
		AppendSessionEnd()

	updates := script.Updates()
	if len(updates) != 5 {
		t.Fatalf("esperava 5 updates, got %d", len(updates))
	}
}

func TestAcpfake_ScriptUnknown(t *testing.T) {
	t.Parallel()

	script := acpfake.NewScript().
		AppendUnknown("weird_kind", nil).
		AppendSessionEnd()

	updates := script.Updates()
	if len(updates) != 2 {
		t.Fatalf("esperava 2 updates, got %d", len(updates))
	}
}

func TestAcpfake_ScriptRequestPermission(t *testing.T) {
	t.Parallel()

	script := acpfake.NewScript().
		AppendRequestPermission("edit_file").
		AppendSessionEnd()

	updates := script.Updates()
	if len(updates) != 2 {
		t.Fatalf("esperava 2 updates, got %d", len(updates))
	}
}

func TestAcpfake_ServerStart(t *testing.T) {
	t.Parallel()

	script := acpfake.NewScript().AppendAgentMessage("teste").AppendSessionEnd()
	srv := acpfake.NewServer(script)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pc, err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if pc == nil {
		t.Fatal("PipeConn retornado é nil")
	}
	if pc.ClientReader == nil {
		t.Error("ClientReader é nil")
	}
	if pc.ClientWriter == nil {
		t.Error("ClientWriter é nil")
	}
}

func TestAcpfake_ScriptAgentMessageDelay(t *testing.T) {
	t.Parallel()

	script := acpfake.NewScript().
		AppendAgentMessageWithDelay("atrasada", 10*time.Millisecond).
		AppendSessionEnd()

	updates := script.Updates()
	if len(updates) != 2 {
		t.Fatalf("esperava 2 updates, got %d", len(updates))
	}
	if updates[0].Delay != 10*time.Millisecond {
		t.Errorf("delay esperado 10ms, got %v", updates[0].Delay)
	}
}
