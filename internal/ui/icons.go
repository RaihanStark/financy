package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Nav icons reuse Fyne's bundled theme icons so we don't ship custom assets yet.
func iconAccounts() fyne.Resource     { return theme.StorageIcon() }
func iconTransactions() fyne.Resource { return theme.HistoryIcon() }
func iconBudget() fyne.Resource       { return theme.AccountIcon() }
func iconAnalytics() fyne.Resource    { return theme.GridIcon() }
func iconRecurring() fyne.Resource    { return theme.MediaReplayIcon() }
func iconDebts() fyne.Resource        { return theme.ContentPasteIcon() }
