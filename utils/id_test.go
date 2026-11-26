package utils

import "testing"

func TestDockerProjectNameHelpers(t *testing.T) {
	challengeName := "Web Challenge 01"
	encoded := EncodeID(challengeName)

	if got, want := ProjectNameNotInstanced(challengeName), "beast-"+encoded; got != want {
		t.Fatalf("ProjectNameNotInstanced() = %q, want %q", got, want)
	}

	if got, want := ComposeDockerProjectNameInstanced(challengeName, "abc123"), "beast-instance-"+encoded+"-abc123"; got != want {
		t.Fatalf("ComposeDockerProjectNameInstanced() = %q, want %q", got, want)
	}
}

func TestExtractInstanceIDFromComposeProjectName(t *testing.T) {
	for _, tc := range []struct {
		name     string
		project  string
		wantID   string
		wantPass bool
	}{
		{
			name:     "hyphenated uuid prefix",
			project:  ComposeDockerProjectNameInstanced("instanced-compose", "22b3361d-263"),
			wantID:   "22b3361d-263",
			wantPass: true,
		},
		{
			name:     "simple id",
			project:  ComposeDockerProjectNameInstanced("Web Challenge 01", "abc123"),
			wantID:   "abc123",
			wantPass: true,
		},
		{
			name:     "non instanced compose project",
			project:  ProjectNameNotInstanced("compose-type"),
			wantID:   "",
			wantPass: false,
		},
		{
			name:     "missing instance segment",
			project:  "beast-instance-abcdef",
			wantID:   "",
			wantPass: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotID, gotOK := ExtractInstanceIDFromComposeProjectName(tc.project)
			if gotOK != tc.wantPass || gotID != tc.wantID {
				t.Fatalf("ExtractInstanceIDFromComposeProjectName(%q) = (%q, %v), want (%q, %v)",
					tc.project, gotID, gotOK, tc.wantID, tc.wantPass)
			}
		})
	}
}
