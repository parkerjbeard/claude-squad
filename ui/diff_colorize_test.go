package ui

import (
	"strings"
	"testing"
)

// TestColorizeDiffBasic validates that colorizeDiff preserves structure
// and includes expected content segments for different diff line types.
func TestColorizeDiffBasic(t *testing.T) {
	input := strings.Join([]string{
		"diff --git a/file.txt b/file.txt",
		"index 1111111..2222222 100644",
		"--- a/file.txt",
		"+++ b/file.txt",
		"@@ -1,3 +1,4 @@",
		" line unchanged",
		"+added line",
		"-removed line",
		"",
	}, "\n")

	out := colorizeDiff(input)

	// Basic shape: output may have trailing newline added
	inLines := strings.Split(input, "\n")
	outLines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(outLines) != len(inLines) {
		t.Fatalf("expected %d lines, got %d", len(inLines), len(outLines))
	}

	// Ensure core content survives styling
	if !strings.Contains(out, "@@ -1,3 +1,4 @@") {
		t.Fatalf("expected hunk header present in output")
	}
	if !strings.Contains(out, "+added line") {
		t.Fatalf("expected added line present in output")
	}
	if !strings.Contains(out, "-removed line") {
		t.Fatalf("expected removed line present in output")
	}
	if !strings.Contains(out, "line unchanged") {
		t.Fatalf("expected unchanged line present in output")
	}
}
