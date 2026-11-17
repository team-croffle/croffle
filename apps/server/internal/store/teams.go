package store

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// MaxTeamName is the longest team name, in characters.
const MaxTeamName = 64

// ErrInvalidTeamName is a name that is empty after trimming or too long.
var ErrInvalidTeamName = errors.New("store: a team name has 1 to 64 characters")

// NormaliseTeamName trims a team name and checks its length. Names need not be
// unique: the id identifies a team.
func NormaliseTeamName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > MaxTeamName {
		return "", ErrInvalidTeamName
	}
	return name, nil
}

// NewTeamID returns "team-" and 10 random lowercase hex characters.
func NewTeamID() string {
	var buf [5]byte
	// rand.Read panics rather than failing on supported platforms.
	_, _ = rand.Read(buf[:])
	return "team-" + hex.EncodeToString(buf[:])
}

// ValidateTeamID accepts the ids an operator may choose: 1 to 64 of
// [a-z0-9-], so an id is safe in a URL path and in a log line.
func ValidateTeamID(id string) error {
	if id == "" || len(id) > 64 {
		return fmt.Errorf("a team id has 1 to 64 characters, got %q", id)
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return fmt.Errorf("a team id uses a-z, 0-9 and -, got %q", id)
		}
	}
	return nil
}

// TeamMember is one account in a team.
type TeamMember struct {
	AccountID string    `json:"accountId"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

// Membership is an account's membership of one team, identified by when it
// began: leaving and joining again is a new membership.
type Membership struct {
	TeamID     string
	JoinedAtMs int64
}
