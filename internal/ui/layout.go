package ui

// Layout constants — centralized to reduce magic numbers in view code.
// Changing a value here propagates to all consumers automatically.
const (
	// tabBoxHeaderLines is the number of non-content lines consumed by the
	// tab-box header area: tabs + sep + header + sep.
	tabBoxHeaderLines = 4

	// compactWidthThreshold is the terminal width below which footers
	// switch to a compact single-line layout.
	compactWidthThreshold = 60

	// twoLineFooterMinHeight is the minimum terminal height for showing
	// a second footer line with additional key hints.
	twoLineFooterMinHeight = 20

	// cmdPrompt modal sizing.
	cmdPromptMaxW    = 88
	cmdPromptMaxH    = 9
	cmdPromptMarginW = 6
	cmdPromptMarginH = 10

	// helpMaxBoxWidth is the maximum width for the help overlay box.
	helpMaxBoxWidth = 88

	// helpBoxOverhead is the lines consumed by the help box border + padding
	// (top border + top padding + bottom padding + bottom border).
	helpBoxOverhead = 4

	// modalFormLabelWidth is the label column width for modal forms
	// (group form, host form).
	modalFormLabelWidth = 14

	// defaultsFormLabelWidth is the label column width for the full-screen
	// settings (defaults) form.
	defaultsFormLabelWidth = 16

	// quitConfirmMaxW is the maximum content width for the quit confirm dialog.
	quitConfirmMaxW = 52

	// confirmDialogMaxW is the maximum content width for generic confirm
	// dialogs (delete, connect, remove).
	confirmDialogMaxW = 60

	// confirmDialogMargin is the margin subtracted from the terminal width
	// when sizing confirm dialogs.
	confirmDialogMargin = 4

	// confirmDialogPadding is the extra width added to confirm dialog
	// content to form the total box width.
	confirmDialogPadding = 6
)

func tabBoxListContentHeight(width, height int) int {
	innerH := max(0, height-2)
	footerLines := 1
	if width >= compactWidthThreshold && height >= twoLineFooterMinHeight {
		footerLines = 2
	}
	// tab/header lines + footer separator + footer lines.
	return max(1, innerH-tabBoxHeaderLines-1-footerLines)
}
