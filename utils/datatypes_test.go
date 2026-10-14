package utils

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCleanStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "nil slice returns empty slice",
			input:    nil,
			expected: []string{},
		},
		{
			name:     "empty slice returns empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "slice with only empty strings returns empty slice",
			input:    []string{"", ""},
			expected: []string{},
		},
		{
			name:     "slice with whitespace strings returns empty slice",
			input:    []string{"   ", "\t"},
			expected: []string{},
		},
		{
			name:     "slice with mixed strings removes empty elements",
			input:    []string{"tag1", "", "  tag2  ", "   "},
			expected: []string{"tag1", "tag2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanStringSlice(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("CleanStringSlice(%v) = %v, want %v", tt.input, got, tt.expected)
			}

			jsonData, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}
			if len(tt.expected) == 0 && string(jsonData) != "[]" {
				t.Errorf("json.Marshal(got) = %s, want []", string(jsonData))
			}
		})
	}
}

func TestSplitDelimited(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		sep      string
		expected []string
	}{
		{
			name:     "empty string returns empty slice",
			input:    "",
			sep:      "::::",
			expected: []string{},
		},
		{
			name:     "whitespace string returns empty slice",
			input:    "   ",
			sep:      "::::",
			expected: []string{},
		},
		{
			name:     "delimiters only returns empty slice",
			input:    "::::::::",
			sep:      "::::",
			expected: []string{},
		},
		{
			name:     "single item",
			input:    "asset1.png",
			sep:      "::::",
			expected: []string{"asset1.png"},
		},
		{
			name:     "multiple items",
			input:    "asset1.png::::asset2.zip",
			sep:      "::::",
			expected: []string{"asset1.png", "asset2.zip"},
		},
		{
			name:     "multiple items with trailing delimiter",
			input:    "asset1.png::::asset2.zip::::",
			sep:      "::::",
			expected: []string{"asset1.png", "asset2.zip"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitDelimited(tt.input, tt.sep)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("SplitDelimited(%q, %q) = %v (len %d), want %v (len %d)",
					tt.input, tt.sep, got, len(got), tt.expected, len(tt.expected))
			}

			// Ensure JSON serialization gives [] and never [""] or null
			jsonData, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}
			if len(tt.expected) == 0 && string(jsonData) != "[]" {
				t.Errorf("json.Marshal(got) = %s, want []", string(jsonData))
			}
		})
	}
}
