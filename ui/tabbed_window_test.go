package ui

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetActiveTabAndToggle(t *testing.T) {
	tw := NewTabbedWindow(NewPreviewPane(), NewDiffPane())
	// Default should be PreviewTab
	require.Equal(t, PreviewTab, tw.GetActiveTab())
	require.False(t, tw.IsInDiffTab())

	// Toggle to Diff tab
	tw.Toggle()
	require.Equal(t, DiffTab, tw.GetActiveTab())
	require.True(t, tw.IsInDiffTab())

	// Toggle back to Preview tab
	tw.Toggle()
	require.Equal(t, PreviewTab, tw.GetActiveTab())
	require.False(t, tw.IsInDiffTab())
}
