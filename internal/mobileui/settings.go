package mobileui

import (
	"errors"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
)

// openSettings is the top-level Settings page: a menu of sections, each drilling
// into its own sub-page (mirrors the desktop Preferences, split for mobile).
func (m *mobileApp) openSettings() {
	m.pushPage(func(close func()) fyne.CanvasObject {
		menu := listCard(
			settingsRow(theme.DocumentIcon(), "Document", "Open, export, demo or reset", m.settingsDocument),
			rowDivider(),
			settingsRow(theme.SettingsIcon(), "Configuration", "Currency & number format", m.settingsConfiguration),
			rowDivider(),
			settingsRow(theme.ListIcon(), "Categories", "Income & expense labels", m.settingsCategories),
			rowDivider(),
			settingsRow(theme.StorageIcon(), "Data summary", "Counts & totals", m.settingsDataSummary),
			rowDivider(),
			settingsRow(theme.InfoIcon(), "About", "Version & info", m.settingsAbout),
		)
		body := container.NewVBox(
			menu,
			gap(16),
			container.NewCenter(newText("Financy v"+core.Version, colInkFaint, 11, false)),
		)
		return m.formPage("Settings", "", body, nil, close)
	})
}

// settingsRow is a tappable menu entry: a tinted icon badge, a title/subtitle,
// and a trailing chevron.
func settingsRow(icon fyne.Resource, title, subtitle string, tap func()) fyne.CanvasObject {
	badge := iconCircle(icon, colPrimary)
	texts := flexBox(container.NewVBox(
		newText(title, colInk, 14, true),
		newText(subtitle, colInkFaint, 11, false),
	))
	chev := widget.NewIcon(theme.NewThemedResource(theme.NavigateNextIcon()))
	row := container.NewBorder(nil, nil, badge, container.NewCenter(chev), insets(texts, 0, 0, 10, 8))
	return newTappableCard(insets(row, 8, 8, 8, 8), tap)
}

// ---- Document ----

// settingsDocument manages the single on-device file: open/export a .financy
// file, load demo data, or start fresh, plus the live-sync status.
func (m *mobileApp) settingsDocument() {
	m.pushPage(func(close func()) fyne.CanvasObject {
		openBtn := widget.NewButton("Open .financy file…", func() { close(); m.importFile() })
		openBtn.Importance = widget.HighImportance
		exportBtn := widget.NewButton("Export a copy…", func() { close(); m.exportFile() })
		demoBtn := widget.NewButton("Load demo data", func() {
			close()
			m.confirmReplace("Replace everything with demo data?", true)
		})
		freshBtn := widget.NewButton("Erase & start fresh", func() {
			close()
			m.confirmReplace("Erase all data and start with an empty document?", false)
		})

		box := container.NewVBox(
			wrapText("This device keeps one document, saved automatically. Open a "+
				".financy file to bring it here, or export a copy to share it."),
			gap(10),
			openBtn,
			gap(6),
			exportBtn,
			gap(6),
			demoBtn,
			gap(6),
			freshBtn,
		)

		if m.sourceURI != nil {
			unlink := widget.NewButton("Stop syncing", func() {
				m.sourceURI = nil
				close()
			})
			box.Add(gap(16))
			box.Add(newText("Syncing to", colInk, 14, true))
			box.Add(newText(m.sourceURI.Name(), colPos, 12, true))
			box.Add(wrapText("Edits are written back to this file while the app is open. " +
				"Re-open it after restarting to resume syncing."))
			box.Add(gap(6))
			box.Add(unlink)
		}

		return m.formPage("Document", "", box, nil, close)
	})
}

// ---- Configuration ----

// settingsConfiguration sets the currency and number format (like the desktop
// Configuration tab).
func (m *mobileApp) settingsConfiguration() {
	m.pushPage(func(close func()) fyne.CanvasObject {
		currency := newSelect([]string{"Rp", "$", "€", "£"}, nil)
		currency.SetSelected(m.store.Currency())

		const fmtDefault = "Currency default"
		numFmt := newSelect(append([]string{fmtDefault}, core.NumberFormatStyles()...), nil)
		if m.store.NumberFormat() == "" {
			numFmt.SetSelected(fmtDefault)
		} else {
			numFmt.SetSelected(m.store.NumberFormat())
		}

		save := func() {
			// Owner and year aren't user-facing on mobile; preserve them.
			m.store.SetSettings(m.store.Owner(), currency.Selected, m.store.Year())
			style := numFmt.Selected
			if style == fmtDefault {
				style = ""
			}
			m.store.SetNumberFormat(style)
			close()
		}

		body := container.NewVBox(
			field("Currency", currency),
			field("Number format", numFmt),
			gap(4),
			wrapText("The currency sets the symbol and decimals; the number format controls "+
				"the thousands and decimal separators (e.g. 1,234.56 vs 1.234,56)."),
		)
		return m.formPage("Configuration", "Save", body, save, close)
	})
}

// ---- Categories ----

// settingsCategories lists income and expense categories with add/edit/delete
// (like the desktop Categories tab).
func (m *mobileApp) settingsCategories() { m.pushView(m.categoriesContent()) }

func (m *mobileApp) categoriesContent() fyne.CanvasObject {
	rebuild := func() { m.replaceView(m.categoriesContent()) }

	section := func(title string, accts []core.Account) fyne.CanvasObject {
		header := newText(title, colInkFaint, 11, true)
		if len(accts) == 0 {
			return container.NewVBox(header, gap(4), mutedCard("None yet"))
		}
		rows := make([]fyne.CanvasObject, 0, len(accts)*2)
		for i, a := range accts {
			if i > 0 {
				rows = append(rows, rowDivider())
			}
			rows = append(rows, m.categoryRow(a, rebuild))
		}
		return container.NewVBox(header, gap(4), listCard(rows...))
	}

	body := container.NewVBox(
		wrapText("Labels used to classify transactions."),
		gap(12),
		section("INCOME", m.store.IncomeAccounts()),
		gap(16),
		section("EXPENSES", m.store.ExpenseAccounts()),
		gap(20),
	)
	return m.formPage("Categories", "Add", body,
		func() { m.categoryForm(nil, rebuild) }, m.back)
}

func (m *mobileApp) categoryRow(a core.Account, done func()) fyne.CanvasObject {
	name := newText(a.Name, colInk, 14, true)
	texts := container.NewVBox(name)
	if n := strings.TrimSpace(a.Notes); n != "" {
		texts.Add(newText(ellipsize(n, 30), colInkFaint, 11, false))
	}
	edit := iconButton(theme.DocumentCreateIcon(), func() { m.categoryForm(&a, done) })
	del := iconButton(theme.DeleteIcon(), func() { m.deleteCategory(a, done) })
	row := container.NewBorder(nil, nil, flexBox(insets(texts, 0, 0, 2, 2)),
		container.NewHBox(edit, del))
	return insets(row, 4, 4, 8, 4)
}

// categoryForm adds or edits an income/expense category. existing == nil adds.
func (m *mobileApp) categoryForm(existing *core.Account, onDone func()) {
	name := newEntry()
	name.SetPlaceHolder("e.g. Groceries, Salary")
	typ := newSelect([]string{"Expense", "Income"}, nil)
	notes := newEntry()
	notes.SetPlaceHolder("Optional")

	title := "Add category"
	if existing != nil {
		title = "Edit category"
		name.SetText(existing.Name)
		typ.SetSelected(string(existing.Type))
		notes.SetText(existing.Notes)
	} else {
		typ.SetSelected("Expense")
	}

	nameF := field("Name", name)
	notesF := field("Notes", notes)
	commit := func() {
		if strings.TrimSpace(name.Text) == "" || typ.Selected == "" {
			dialog.ShowError(errors.New("enter a name and type"), m.win)
			return
		}
		acct := core.Account{Name: name.Text, Type: core.AcctType(typ.Selected), Notes: notes.Text}
		if existing != nil {
			m.store.UpdateAccount(existing.ID, acct)
		} else {
			m.store.AddAccount(acct)
		}
		m.back()
		if onDone != nil {
			onDone()
		}
	}
	body := container.NewVBox(nameF, field("Type", typ), notesF)
	m.pushView(m.formPage(title, "Save", body, commit, m.back))
}

func (m *mobileApp) deleteCategory(a core.Account, done func()) {
	if m.store.TxnCountForAccount(a.ID) > 0 {
		dialog.ShowInformation("Cannot delete",
			"“"+a.Name+"” is used by transactions. Remove them first.", m.win)
		return
	}
	m.pushPage(func(close func()) fyne.CanvasObject {
		del := widget.NewButton("Delete category", func() {
			close()
			m.store.DeleteAccount(a.ID)
			if done != nil {
				done()
			}
		})
		del.Importance = widget.HighImportance
		cancel := widget.NewButton("Cancel", close)
		body := container.NewVBox(
			wrapText("Delete “"+a.Name+"”? This can't be undone."),
			gap(18), del, gap(6), cancel,
		)
		return m.formPage("Delete category?", "", body, nil, close)
	})
}

// ---- Data summary ----

// settingsDataSummary is a read-only snapshot of counts and totals (like the
// desktop Data Summary tab).
func (m *mobileApp) settingsDataSummary() {
	m.pushPage(func(close func()) fyne.CanvasObject {
		s := m.store
		counts := listCard(
			detailRow("Asset accounts", strconv.Itoa(len(s.AssetAccounts()))),
			thinLine(),
			detailRow("Liability accounts", strconv.Itoa(len(s.LiabilityAccounts()))),
			thinLine(),
			detailRow("Income categories", strconv.Itoa(len(s.IncomeAccounts()))),
			thinLine(),
			detailRow("Expense categories", strconv.Itoa(len(s.ExpenseAccounts()))),
			thinLine(),
			detailRow("Transactions", strconv.Itoa(len(s.Transactions()))),
		)
		totals := listCard(
			detailRow("Total assets", core.FmtMoney(s.TotalAssets())),
			thinLine(),
			detailRow("Total liabilities", core.FmtMoney(s.TotalLiabilities())),
			thinLine(),
			detailRow("Net worth", core.FmtMoney(s.NetWorth())),
			thinLine(),
			detailRow("Income (YTD)", core.FmtMoney(s.IncomeTotal())),
			thinLine(),
			detailRow("Expenses (YTD)", core.FmtMoney(s.ExpenseTotal())),
		)
		body := container.NewVBox(
			newText("Counts", colInkFaint, 11, true), gap(4), counts,
			gap(16),
			newText("Totals", colInkFaint, 11, true), gap(4), totals,
		)
		return m.formPage("Data summary", "", body, nil, close)
	})
}

// ---- About ----

func (m *mobileApp) settingsAbout() {
	m.pushPage(func(close func()) fyne.CanvasObject {
		body := container.NewVBox(
			gap(10),
			container.NewCenter(newText("Financy", colInk, 22, true)),
			container.NewCenter(newText("v"+core.Version, colInkDim, 13, false)),
			gap(16),
			wrapText("Financy is a private, offline, double-entry budgeting app. Your data "+
				"lives in a single .financy file on your device and is never uploaded."),
		)
		return m.formPage("About", "", body, nil, close)
	})
}

// confirmReplace is a full-screen confirmation before discarding the current
// document (load demo / start fresh).
func (m *mobileApp) confirmReplace(msg string, demo bool) {
	m.pushPage(func(close func()) fyne.CanvasObject {
		replace := widget.NewButton("Replace", func() { close(); m.replaceDocument(demo) })
		replace.Importance = widget.HighImportance
		cancel := widget.NewButton("Cancel", close)

		body := container.NewVBox(
			wrapText(msg),
			gap(18),
			replace,
			gap(6),
			cancel,
		)
		return m.formPage("Are you sure?", "", body, nil, close)
	})
}
