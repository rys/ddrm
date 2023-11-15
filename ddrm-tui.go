//go:build client
// +build client

package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TUI state
type UiModel struct {
	recordTable table.Model
}

// TUI msg struct that supports String()
type uiProcessRecords struct {
	process bool
}

func (r uiProcessRecords) String() string {
	return ""
}

// TUI runtime objects
var (
	ddrmTui *tea.Program

	uiBaseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	uiTermColor = termenv.EnvColorProfile().Color

	helpText      = termenv.Style{}.Foreground(uiTermColor("241")).Styled
	highlightText = termenv.Style{}.Foreground(uiTermColor("204")).Background(uiTermColor("235")).Styled
)

func (ui UiModel) Init() tea.Cmd {
	return nil
}

func (ui UiModel) View() string {
	return uiBaseStyle.Render(ui.recordTable.View()) + "\n\n" +
		helpText("q: exit • x: toggle altscreen mode  ") + highlightText(fmt.Sprint(len(ddrmRecordStates))+" records\n")
}

func (ui UiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			cronScheduler.Shutdown()
			return ui, tea.Quit
		case "x":
			var cmd tea.Cmd
			if stateAltScreenMode {
				cmd = tea.ExitAltScreen
			} else {
				cmd = tea.EnterAltScreen
			}
			stateAltScreenMode = !stateAltScreenMode
			return ui, cmd
		}
	case uiProcessRecords:
		return updateUi(), nil
	}

	return updateUi(), nil
}

func updateUi() UiModel {
	columns := []table.Column{
		{Title: "-", Width: 1},
		{Title: "FQDN", Width: 30},
		{Title: "RR", Width: 4},
		{Title: "Expected", Width: 20},
		{Title: "Currently", Width: 20},
		{Title: "✉︎", Width: 1},
		{Title: "↓", Width: 1},
		{Title: "⚠️", Width: 1},
	}

	rows := []table.Row{}

	// golang doesn't have an ordered map type, so we need to extract the keys,
	// sort those, and then index into the record states map with the sorted key
	keys := make([]string, 0)
	for k := range ddrmRecordStates {
		keys = append(keys, k)
	}

	slices.Sort(keys)

	// turn each record state into a row
	for _, k := range keys {
		rowState := ddrmRecordStates[k]

		processing := ""
		if rowState.Processing {
			processing = "x"
		}

		email := ""
		if rowState.SentEmail {
			email = "x"
		}

		changed := ""
		if rowState.Changed {
			changed = "x"
		}

		errored := ""
		if rowState.Errored {
			errored = "x"
		}

		prior := ""
		if len(rowState.PriorValues) == 1 {
			prior = rowState.PriorValues[0]
		} else if len(rowState.PriorValues) > 1 {
			prior = rowState.PriorValues[0] + ", ..."
		}

		current := ""
		if len(rowState.CurrentValues) == 1 {
			current = rowState.CurrentValues[0]
		} else if len(rowState.CurrentValues) > 1 {
			current = rowState.CurrentValues[0] + ", ..."
		}

		row := table.Row{
			processing,
			rowState.FQDN,
			string(rowState.Type),
			prior,
			current,
			email,
			changed,
			errored,
		}

		rows = append(rows, row)
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(len(rows)),
	)

	style := table.DefaultStyles()
	style.Header = style.Header.BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)

	style.Selected = style.Selected.Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	t.SetStyles(style)

	uiModel := UiModel{t}

	return uiModel
}

func sendUpdateUIMsg() {
	if stateUI {
		ddrmTui.Send(uiProcessRecords{process: true})
	}
}

func setupTui() {
	if stateUI {
		ddrmTui = tea.NewProgram(updateUi())
	}
}

func runTui() {
	if stateUI {
		if _, err := ddrmTui.Run(); err != nil {
			os.Exit(ddrmExitErrorRunningTUI)
		}
	}
}
