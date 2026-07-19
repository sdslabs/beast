package database

import "testing"

func TestValidatedQueryColumnRejectsSQLFragments(t *testing.T) {
	if column, err := validatedQueryColumn("ID", "id", "name"); err != nil || column != "id" {
		t.Fatalf("expected normalized ID column, got %q, %v", column, err)
	}
	if _, err := validatedQueryColumn("id OR 1=1", "id", "name"); err == nil {
		t.Fatal("expected SQL fragment to be rejected")
	}
}
