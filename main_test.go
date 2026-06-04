package main

import "testing"

func TestRunHelp(t *testing.T) {
	if err := run([]string{"help"}); err != nil {
		t.Fatalf("run help failed: %v", err)
	}
}

func TestRunVersion(t *testing.T) {
	if err := run([]string{"version"}); err != nil {
		t.Fatalf("run version failed: %v", err)
	}
}

func TestRunRequiresCommand(t *testing.T) {
	if err := run(nil); err == nil {
		t.Fatal("expected missing command error")
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	if err := run([]string{"unknown"}); err == nil {
		t.Fatal("expected unknown command error")
	}
}
