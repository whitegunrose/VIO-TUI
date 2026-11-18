package canvas

import (
	"VIO/internal/model"
	"testing"
)

func TestKeepManual(t *testing.T) {
	synced := []model.Assignment{{ID: "c1", CanvasID: 1}, {ID: "c2", CanvasID: 2}}
	local := []model.Assignment{{ID: "old", CanvasID: 9}, {ID: "m1", Manual: true}}
	got := KeepManual(synced, local)
	var ids []string
	for _, a := range got {
		ids = append(ids, a.ID)
	}
	want := []string{"c1", "c2", "m1"}
	if len(ids) != len(want) {
		t.Fatalf("got %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("got %v, want %v", ids, want)
		}
	}
}
