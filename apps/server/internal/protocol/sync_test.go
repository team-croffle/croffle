package protocol

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func sampleRow() Row {
	return Row{
		// Promoted: the server owns these as columns.
		"version":      float64(3),
		"updatedAtMs":  float64(1_700_000_000_000),
		"deletedAt":    nil,
		"lastWriterId": "device:client-1",
		"ownerId":      "acct-1",
		"teamId":       nil,
		// Opaque: the server never interprets these.
		"id":             "row-1",
		"title":          "standup",
		"isAllDay":       float64(0),
		"startDate":      float64(1_700_000_000),
		"recurringRule":  nil,
		"useDefaultRemi": float64(1),
	}
}

func TestSplitRowKeepsOnlyTheOpaqueFields(t *testing.T) {
	payload, err := SplitRow(sampleRow())
	if err != nil {
		t.Fatalf("SplitRow: %v", err)
	}

	var rest map[string]any
	if err := json.Unmarshal(payload, &rest); err != nil {
		t.Fatalf("decode payload: %v", err)
	}

	// Two sources of truth for ownership or for the LWW key would be a bug
	// waiting to happen, so the promoted fields are not in the payload.
	for _, field := range promotedFields {
		if _, ok := rest[field]; ok {
			t.Errorf("payload still contains the promoted field %q", field)
		}
	}
	for _, field := range []string{"id", "title", "startDate", "recurringRule"} {
		if _, ok := rest[field]; !ok {
			t.Errorf("payload lost the calendar field %q", field)
		}
	}
}

func TestSplitThenMergeRoundTrips(t *testing.T) {
	original := sampleRow()

	payload, err := SplitRow(original)
	if err != nil {
		t.Fatalf("SplitRow: %v", err)
	}

	writer := "device:client-1"
	merged, err := MergeRow(payload, 3, 1_700_000_000_000, &writer, "acct-1", nil, nil)
	if err != nil {
		t.Fatalf("MergeRow: %v", err)
	}

	// A client has to receive the same shape it sent, or decodeRow on the
	// desktop side sees a different row than it wrote.
	if len(merged) != len(original) {
		t.Fatalf("merged has %d fields, want %d\n got: %v\nwant: %v", len(merged), len(original), merged, original)
	}
	for key, want := range original {
		got, ok := merged[key]
		if !ok {
			t.Errorf("merged row is missing %q", key)
			continue
		}
		if fmtValue(got) != fmtValue(want) {
			t.Errorf("%q = %v, want %v", key, got, want)
		}
	}
}

func TestMergeRowRestoresTombstoneAndTeam(t *testing.T) {
	deletedAt := int64(1_700_000_001_000)
	team := "team-default"

	merged, err := MergeRow(json.RawMessage(`{"id":"row-1"}`), 4, 1_700_000_001_000, nil, "acct-1", &team, &deletedAt)
	if err != nil {
		t.Fatalf("MergeRow: %v", err)
	}

	if merged["deletedAt"] != deletedAt {
		t.Errorf("deletedAt = %v, want %d", merged["deletedAt"], deletedAt)
	}
	if merged["teamId"] != team {
		t.Errorf("teamId = %v, want %s", merged["teamId"], team)
	}
	// Absent values are explicit nulls rather than missing keys, so the client
	// can tell "cleared" from "unknown".
	if value, ok := merged["lastWriterId"]; !ok || value != nil {
		t.Errorf("lastWriterId = %v (present: %v), want an explicit null", value, ok)
	}
}

func TestMergeRowRejectsBrokenPayload(t *testing.T) {
	if _, err := MergeRow(json.RawMessage(`not json`), 1, 1, nil, "acct-1", nil, nil); err == nil {
		t.Error("MergeRow accepted a payload that is not JSON")
	}
}

func TestTeamIDOf(t *testing.T) {
	cases := []struct {
		name string
		row  Row
		want string
		ok   bool
	}{
		{name: "missing", row: Row{}},
		{name: "null", row: Row{"teamId": nil}},
		{name: "blank", row: Row{"teamId": "   "}},
		{name: "set", row: Row{"teamId": "team-default"}, want: "team-default", ok: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := TeamIDOf(tc.row)
			if err != nil {
				t.Fatalf("TeamIDOf: %v", err)
			}
			if !tc.ok {
				if got != nil {
					t.Errorf("teamId = %q, want nil (private row)", *got)
				}
				return
			}
			if got == nil || *got != tc.want {
				t.Errorf("teamId = %v, want %q", got, tc.want)
			}
		})
	}

	if _, err := TeamIDOf(Row{"teamId": float64(3)}); err == nil {
		t.Error("TeamIDOf accepted a number")
	}
}

func TestDeletedAtOf(t *testing.T) {
	got, err := DeletedAtOf(Row{"deletedAt": float64(1_700_000_001_000)})
	if err != nil {
		t.Fatalf("DeletedAtOf: %v", err)
	}
	if got == nil || *got != 1_700_000_001_000 {
		t.Errorf("deletedAt = %v, want the millisecond value", got)
	}

	for _, row := range []Row{{}, {"deletedAt": nil}} {
		got, err := DeletedAtOf(row)
		if err != nil || got != nil {
			t.Errorf("DeletedAtOf(%v) = %v, %v; want nil, nil", row, got, err)
		}
	}

	if _, err := DeletedAtOf(Row{"deletedAt": "yesterday"}); err == nil {
		t.Error("DeletedAtOf accepted a string")
	}
}

func validCommit() Commit {
	return Commit{
		OpID: "op-1", Table: "schedule", RowID: "row-1", Op: OpUpsert,
		BaseVersion: 0, Version: 1, UpdatedAtMs: 1_700_000_000_000,
		LastWriterID: "device:client-1", Row: sampleRow(),
	}
}

func TestValidateCommitAcceptsAGoodOne(t *testing.T) {
	if err := ValidateCommit(validCommit()); err != nil {
		t.Fatalf("ValidateCommit: %v", err)
	}

	// A delete may arrive without a row body.
	remove := validCommit()
	remove.Op = OpDelete
	remove.Row = nil
	remove.BaseVersion = 1
	remove.Version = 2
	if err := ValidateCommit(remove); err != nil {
		t.Errorf("ValidateCommit(delete without a row) = %v", err)
	}
}

func TestValidateCommitRejections(t *testing.T) {
	cases := map[string]func(*Commit){
		"no opId":            func(c *Commit) { c.OpID = " " },
		"long opId":          func(c *Commit) { c.OpID = strings.Repeat("x", MaxIDLength+1) },
		"no rowId":           func(c *Commit) { c.RowID = "" },
		"long rowId":         func(c *Commit) { c.RowID = strings.Repeat("x", MaxIDLength+1) },
		"unknown table":      func(c *Commit) { c.Table = "secrets" },
		"unknown op":         func(c *Commit) { c.Op = "patch" },
		"negative base":      func(c *Commit) { c.BaseVersion = -1 },
		"zero version":       func(c *Commit) { c.Version = 0 },
		"no clock":           func(c *Commit) { c.UpdatedAtMs = 0 },
		"upsert without row": func(c *Commit) { c.Row = nil },
		// Row versions have to increase or baseVersion comparison stops
		// detecting anything; a client that wins a conflict must lift its
		// version above the server's before retrying.
		"version not above base": func(c *Commit) { c.BaseVersion = 5; c.Version = 5 },
		"version below base":     func(c *Commit) { c.BaseVersion = 5; c.Version = 3 },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			commit := validCommit()
			mutate(&commit)

			err := ValidateCommit(commit)
			if err == nil {
				t.Fatalf("ValidateCommit accepted %+v", commit)
			}
			// The handler turns this into a 400, so it has to be recognisable.
			if !errors.Is(err, ErrInvalidCommit) {
				t.Errorf("error = %v, want ErrInvalidCommit", err)
			}
		})
	}
}

func fmtValue(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "<unmarshalable>"
	}
	return string(data)
}

func TestBootstrapCursorRoundTrips(t *testing.T) {
	original := BootstrapCursor{BaseSeq: 42, Table: "schedule", RowID: "row-1"}

	encoded, err := EncodeBootstrapCursor(original)
	if err != nil {
		t.Fatalf("EncodeBootstrapCursor: %v", err)
	}
	// Opaque to the client: it only ever sends back what it was given.
	if strings.Contains(encoded, "schedule") {
		t.Errorf("cursor %q exposes its contents", encoded)
	}

	decoded, err := DecodeBootstrapCursor(encoded)
	if err != nil {
		t.Fatalf("DecodeBootstrapCursor: %v", err)
	}
	if decoded != original {
		t.Errorf("decoded = %+v, want %+v", decoded, original)
	}
}

func TestDecodeBootstrapCursorEmptyMeansFirstPage(t *testing.T) {
	for _, input := range []string{"", "   "} {
		cursor, err := DecodeBootstrapCursor(input)
		if err != nil {
			t.Errorf("DecodeBootstrapCursor(%q): %v", input, err)
		}
		if cursor != (BootstrapCursor{}) {
			t.Errorf("cursor = %+v, want the zero value", cursor)
		}
	}
}

func TestDecodeBootstrapCursorRejectsGarbage(t *testing.T) {
	for _, input := range []string{"!!!!", "YWJj", base64Of(`{"s":-1}`)} {
		if _, err := DecodeBootstrapCursor(input); !errors.Is(err, ErrInvalidCursor) {
			t.Errorf("DecodeBootstrapCursor(%q) error = %v, want ErrInvalidCursor", input, err)
		}
	}
}

func base64Of(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}
