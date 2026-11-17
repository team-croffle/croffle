package store

import (
	"regexp"
	"strings"
	"testing"
)

func TestTeamCreateNameRules(t *testing.T) {
	if got, err := NormaliseTeamName("  Design Team  "); err != nil || got != "Design Team" {
		t.Fatalf("NormaliseTeamName = %q, %v", got, err)
	}
	if _, err := NormaliseTeamName(strings.Repeat("가", 64)); err != nil {
		t.Fatalf("64 characters: %v", err)
	}
	for _, bad := range []string{"", "   ", strings.Repeat("a", 65)} {
		if _, err := NormaliseTeamName(bad); err == nil {
			t.Errorf("NormaliseTeamName(%q) accepted", bad)
		}
	}
	if id := NewTeamID(); !regexp.MustCompile(`^team-[0-9a-f]{10}$`).MatchString(id) {
		t.Fatalf("NewTeamID = %q", id)
	}
	for _, bad := range []string{"", "Team", "team eng", "team/eng", strings.Repeat("a", 65)} {
		if ValidateTeamID(bad) == nil {
			t.Errorf("ValidateTeamID(%q) accepted", bad)
		}
	}
}
