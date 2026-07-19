package main

import (
	"slices"
	"testing"
)

func TestBeastRedisACLRulesUseExplicitCommands(t *testing.T) {
	rules := beastRedisACLRules("secret")
	if !slices.Contains(rules, "reset") || !slices.Contains(rules, "~beast:*") {
		t.Fatalf("ACL rules do not reset and constrain key access: %v", rules)
	}
	if slices.Contains(rules, "+@all") {
		t.Fatalf("ACL rules grant every Redis command: %v", rules)
	}
	if !slices.Contains(rules, "+eval") || !slices.Contains(rules, "+config|get") {
		t.Fatalf("ACL rules omit required Beast commands: %v", rules)
	}
}
