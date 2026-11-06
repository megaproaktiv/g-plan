package main

import (
	"strings"
	"testing"
)

func TestChatBatchRunsInOrder(t *testing.T) {
	a := NewApp()
	r := a.ChatBatch("# build a plan\n\nadd action \"One\"\nadd action \"Two\"\n// comment\nadd action \"Three\"\n")
	if r.Total != 3 || r.Done != 3 || r.Failed != 0 || r.Error != "" {
		t.Fatalf("%+v", r)
	}
	if len(a.project.Packages) != 3 || a.project.Packages[0].Name != "One" || a.project.Packages[2].Name != "Three" {
		t.Errorf("packages: %+v", a.project.Packages)
	}
}

func TestChatBatchStopsAtFirstError(t *testing.T) {
	a := NewApp()
	r := a.ChatBatch("add action \"One\"\nest 99.9.9 oops\nadd action \"Never\"")
	if r.Failed != 2 || r.Done != 1 || r.Error == "" {
		t.Fatalf("%+v", r)
	}
	if len(a.project.Packages) != 1 {
		t.Errorf("the command after the error must not run: %d packages", len(a.project.Packages))
	}
	if !strings.Contains(r.Log[len(r.Log)-1], "ERROR") {
		t.Errorf("log: %v", r.Log)
	}
}

func TestChatBatchStopsAtQuit(t *testing.T) {
	a := NewApp()
	r := a.ChatBatch("add action \"One\"\nquit\nadd action \"Never\"")
	if r.Stopped != 2 || r.Done != 1 || r.Failed != 0 || len(a.project.Packages) != 1 {
		t.Fatalf("%+v", r)
	}
}

func TestChatBatchIsOneUndoStep(t *testing.T) {
	a := NewApp()
	before := len(a.undoStack)
	a.ChatBatch("add action \"One\"\nadd action \"Two\"\nadd action \"Three\"")
	if len(a.undoStack) != before+1 {
		t.Fatalf("undo steps: %d, want 1", len(a.undoStack)-before)
	}
	if _, err := a.Undo(); err != nil {
		t.Fatal(err)
	}
	if len(a.project.Packages) != 0 {
		t.Errorf("undo should take back the whole batch, %d packages left", len(a.project.Packages))
	}
}

func TestChatBatchEmptyOrFailingFirstLeavesNoUndoStep(t *testing.T) {
	a := NewApp()
	before := len(a.undoStack)
	a.ChatBatch("\n# nothing\n")
	a.ChatBatch("est zzz 5")
	if len(a.undoStack) != before {
		t.Errorf("undo stack grew by %d", len(a.undoStack)-before)
	}
}
