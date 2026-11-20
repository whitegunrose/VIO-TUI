package canvas

import (
	"VIO/internal/model"
	"fmt"
	"strings"
	"testing"
)

// Fuzz tests check laws: rules that must hold for every input. See
// aidlc-docs/design-artifacts/testing_strategy.md. To search for new failing
// inputs: go test -run '^$' -fuzz FuzzNextLink -fuzztime 30s ./internal/canvas

// Law C1: reading a Link header never panics, and the URL it returns is
// copied from inside that header (never made up).
func FuzzNextLink(f *testing.F) {
	for _, s := range []string{
		"",
		`<https://x/api/v1/courses?page=2>; rel="next", <https://x/api/v1/courses?page=1>; rel="first"`,
		`<https://x/a?page=1>; rel="current",<https://x/a?page=2>; rel="next"`,
		`rel="next"`, `>; rel="next" <`, `<<>>; rel="next"`, `<a,b>; rel="next"`,
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, header string) {
		next := nextLink(header)
		if next != "" && !strings.Contains(header, "<"+next+">") {
			t.Fatalf("nextLink(%q) = %q, which is not a <url> in the header", header, next)
		}
	})
}

// Law C2: merging keeps Canvas in charge of which courses exist and the
// user in charge of their own fields. The result has exactly Canvas's
// courses, in Canvas's order, and each one keeps the location, meetings and
// notes the user gave the course with the same Canvas ID.
func FuzzMergeCourses(f *testing.F) {
	f.Add([]byte{1, 2, 3}, []byte{3, 1, 4})
	f.Add([]byte{}, []byte{1})
	f.Add([]byte{1, 1}, []byte{1})
	f.Fuzz(func(t *testing.T, existingIDs, freshIDs []byte) {
		var existing, fresh []model.Course
		userLoc := map[int]string{}
		for i, b := range existingIDs {
			id := int(b % 8)
			loc := fmt.Sprintf("room-%d", i)
			existing = append(existing, model.Course{CanvasID: id, Location: loc, Notes: loc})
			userLoc[id] = loc // the last course with an ID is the one kept
		}
		for _, b := range freshIDs {
			id := int(b % 8)
			fresh = append(fresh, model.Course{CanvasID: id, Name: fmt.Sprintf("course %d", id)})
		}

		merged := mergeCourses(existing, fresh)
		if len(merged) != len(fresh) {
			t.Fatalf("merged %d courses, Canvas sent %d", len(merged), len(fresh))
		}
		for i, c := range merged {
			if c.CanvasID != fresh[i].CanvasID || c.Name != fresh[i].Name {
				t.Fatalf("course %d is %+v, want Canvas's %+v", i, c, fresh[i])
			}
			if c.Location != userLoc[c.CanvasID] || c.Notes != userLoc[c.CanvasID] {
				t.Fatalf("course %d (Canvas ID %d) location %q, want the user's %q", i, c.CanvasID, c.Location, userLoc[c.CanvasID])
			}
		}
	})
}
