package main

import "testing"

func TestDestructiveMaintenanceRequiresConfirmation(t *testing.T) {
	previous := ConfirmDestructive
	ConfirmDestructive = false
	defer func() { ConfirmDestructive = previous }()

	if err := requireDestructiveConfirmation(); err == nil {
		t.Fatal("expected destructive operation without --yes to fail")
	}
	ConfirmDestructive = true
	if err := requireDestructiveConfirmation(); err != nil {
		t.Fatalf("expected confirmed destructive operation to pass: %v", err)
	}
}
