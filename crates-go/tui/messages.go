package tui

// StatusMsg is a transient toast-style status surfaced by Update.
type StatusMsg struct {
	Text string
	OK   bool
}

// ClearStatusMsg removes the active status message.
type ClearStatusMsg struct{}

// EditorFinishedMsg is delivered after an external $EDITOR session ends.
// Tag is an opaque identifier supplied by the caller (commonly a record's
// short id) so the receiver can correlate the result.
type EditorFinishedMsg struct {
	Tag     string
	Content string
	Err     error
}
