package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Nav icons reuse Fyne's bundled theme icons so we don't ship custom assets yet.
func iconDashboard() fyne.Resource    { return theme.HomeIcon() }
func iconAccounts() fyne.Resource     { return theme.StorageIcon() }
func iconTransactions() fyne.Resource { return theme.HistoryIcon() }
func iconBudget() fyne.Resource       { return theme.GridIcon() }
func iconAnalytics() fyne.Resource    { return theme.GridIcon() }
func iconDebts() fyne.Resource        { return theme.WarningIcon() }
func iconIncome() fyne.Resource       { return theme.DownloadIcon() }
func iconGoals() fyne.Resource        { return theme.ConfirmIcon() }
func iconCategories() fyne.Resource   { return theme.ListIcon() }
func iconSettings() fyne.Resource     { return theme.SettingsIcon() }
