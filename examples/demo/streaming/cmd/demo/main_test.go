package main

import "testing"

func TestIsExitCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"exit", true},
		{"quit", true},
		{"bye", true},
		{"goodbye", true},
		{"q", true},
		{":q", true},
		{":quit", true},
		{":exit", true},
		{"done", true},
		{"finish", true},
		{"end", true},
		{"stop", true},
		{"close", true},
		{"leave", true},
		{"logout", true},
		{"ctrl+c", true},
		{"^c", true},
		{"cancel", true},
		{"abort", true},
		{"terminate", true},
		// Test case insensitive
		{"EXIT", true},
		{"QUIT", true},
		{"Bye", true},
		// Test with whitespace
		{"  exit  ", true},
		{"	quit	", true},
		// Test non-exit commands
		{"hello", false},
		{"continue", false},
		{"yes please", false},
		{"", false},
		{"implement the script", false},
	}

	for _, tt := range tests {
		result := isExitCommand(tt.input)
		if result != tt.expected {
			t.Errorf("isExitCommand(%q) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}
