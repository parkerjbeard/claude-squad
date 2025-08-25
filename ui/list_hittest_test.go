package ui

import (
    "claude-squad/session"
    "testing"
)

func TestListHitTestBasic(t *testing.T) {
    l := &List{height: 40, width: 80}
    // Add 5 dummy instances
    for i := 0; i < 5; i++ {
        l.items = append(l.items, &session.Instance{Title: "x"})
    }

    // Before content start should be -1
    if idx := l.HitTest(0); idx != -1 {
        t.Fatalf("expected -1 before content start, got %d", idx)
    }
    // First item region starts around y=5; any y in 5..8 should map to 0 or 1 depending on block size
    if idx := l.HitTest(5); idx != 0 {
        t.Fatalf("expected index 0 at y=5, got %d", idx)
    }
    if idx := l.HitTest(9); idx != 1 {
        t.Fatalf("expected index 1 at y=9, got %d", idx)
    }
    // Click inside third item block should map to index 2
    if idx := l.HitTest(5 + 2*4); idx != 2 { // contentStart + 2 blocks
        t.Fatalf("expected index 2 in third block, got %d", idx)
    }
}
