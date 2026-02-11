// Package tui provides the interactive terminal user interface for dsearch.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/icampana/dsearch/internal/search"
)

var (
	// Styles for the UI
	docStyle          = lipgloss.NewStyle().Padding(1, 2)
	titleStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
	inputStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	placeholderStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	statusStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Italic(true)
	selectedItemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	normalItemStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	previewStyle      = lipgloss.NewStyle().Padding(1).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238"))
)

// resultItem implements list.Item for search results.
type resultItem struct {
	result search.Result
}

// Title returns the title for the list item.
func (i resultItem) Title() string {
	return fmt.Sprintf("%s", i.result.Name)
}

// Description returns the description for the list item.
func (i resultItem) Description() string {
	return fmt.Sprintf("%s [%s]", i.result.Docset, i.result.Type)
}

// FilterValue returns the value to filter by.
func (i resultItem) FilterValue() string {
	return i.result.Name
}

// searchFinishedMsg is sent when a search operation completes.
type searchFinishedMsg struct {
	results []search.Result
	query   string
}

// Model represents the TUI application state.
type Model struct {
	// Dependencies
	engine *search.Engine

	// UI Components
	input textinput.Model
	list  list.Model

	// State
	loading      bool
	lastSearch   string
	searchedOnce bool
	width        int
	height       int
	previewText  string
	showPreview  bool

	// Error state
	err error
}

// NewModel creates a new TUI model with the given search engine.
func NewModel(engine *search.Engine) Model {
	// Initialize text input
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.PlaceholderStyle = placeholderStyle
	ti.TextStyle = inputStyle
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50

	// Initialize list with empty items
	items := []list.Item{}
	delegate := list.NewDefaultDelegate()
	delegate.SetSpacing(0)
	delegate.Styles.SelectedTitle = selectedItemStyle
	delegate.Styles.SelectedDesc = selectedItemStyle
	delegate.Styles.NormalTitle = normalItemStyle
	delegate.Styles.NormalDesc = normalItemStyle

	l := list.New(items, delegate, 0, 0)
	l.Title = "Search Results"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()

	return Model{
		engine:      engine,
		input:       ti,
		list:        l,
		loading:     false,
		showPreview: true,
		width:       80,
		height:      24,
	}
}

// Init returns the initial command for the TUI.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles incoming messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyEnter:
			// Handle selection
			if m.list.SelectedItem() != nil {
				if item, ok := m.list.SelectedItem().(resultItem); ok {
					m.previewText = fmt.Sprintf("Selected: %s\n\nType: %s\nDocset: %s\nPath: %s",
						item.result.Name,
						item.result.Type,
						item.result.Docset,
						item.result.Path,
					)
				}
			}
			return m, nil

		case tea.KeyTab:
			// Toggle preview
			m.showPreview = !m.showPreview
			return m, nil

		case tea.KeyDown, tea.KeyUp:
			// Let the list handle arrow keys if it has items
			if m.list.SelectedItem() != nil {
				var cmd tea.Cmd
				m.list, cmd = m.list.Update(msg)
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

	case tea.WindowSizeMsg:
		// Handle window resize
		m.width = msg.Width
		m.height = msg.Height

		// Update input width
		inputWidth := msg.Width - 4
		if inputWidth > 100 {
			inputWidth = 100
		}
		m.input.Width = inputWidth

		// Update list size
		listHeight := msg.Height - 6
		if m.showPreview {
			listHeight = msg.Height/2 - 4
		}

		m.list.SetSize(msg.Width-4, listHeight)

	case searchFinishedMsg:
		// Handle search completion
		m.loading = false
		m.lastSearch = msg.query
		m.searchedOnce = true

		// Convert results to list items
		items := make([]list.Item, len(msg.results))
		for i, result := range msg.results {
			items[i] = resultItem{result: result}
		}

		m.list.SetItems(items)
		m.list.ResetFilter()

		// If we have results, select the first one
		if len(items) > 0 {
			m.list.Select(0)
			m.previewText = m.buildPreviewText(items[0].(resultItem))
		} else {
			m.previewText = ""
		}

	case error:
		// Handle errors
		m.err = msg
		m.loading = false
	}

	// Update input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	// Check for search trigger (debounced)
	if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyRunes {
		// Trigger search after debounce
		inputText := m.input.Value()
		if len(inputText) >= 2 && inputText != m.lastSearch {
			m.loading = true
			cmds = append(cmds, m.performSearchCmd(inputText))
		} else if len(inputText) == 0 {
			// Clear results on empty input
			m.list.SetItems([]list.Item{})
			m.previewText = ""
		}
	}

	// Update list
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the TUI.
func (m Model) View() string {
	if m.err != nil {
		return docStyle.Render(
			fmt.Sprintf("Error: %v\n\nPress Ctrl+C to quit", m.err),
		)
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("dsearch - Offline Documentation Search\n\n"))

	// Input section
	b.WriteString("Search: ")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")

	// Status
	if m.loading {
		b.WriteString(statusStyle.Render("Searching...\n\n"))
	} else if m.searchedOnce && len(m.list.Items()) == 0 {
		b.WriteString(statusStyle.Render("No results found\n\n"))
	}

	// Results list
	if len(m.list.Items()) > 0 {
		b.WriteString(m.list.View())
		b.WriteString("\n")

		// Preview
		if m.showPreview && m.previewText != "" {
			b.WriteString(previewStyle.Render(m.previewText))
		}
	} else if !m.searchedOnce {
		b.WriteString(statusStyle.Render("Type to search (min 2 characters)\n\n"))
		b.WriteString(statusStyle.Render("Ctrl+C: Quit | Tab: Toggle Preview | Enter: Select\n"))
	} else {
		b.WriteString(statusStyle.Render("Ctrl+C: Quit | Tab: Toggle Preview"))
	}

	return docStyle.Render(b.String())
}

// performSearchCmd performs a search and returns a command.
func (m Model) performSearchCmd(query string) tea.Cmd {
	return func() tea.Msg {
		// Simulate debounce with small delay
		time.Sleep(300 * time.Millisecond)

		results, err := m.engine.Search(query, nil)
		if err != nil {
			return err
		}

		return searchFinishedMsg{
			results: results,
			query:   query,
		}
	}
}

// buildPreviewText builds the preview text for a result.
func (m Model) buildPreviewText(item resultItem) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Name: %s\n", item.result.Name))
	b.WriteString(fmt.Sprintf("Type: %s\n", item.result.Type))
	b.WriteString(fmt.Sprintf("Docset: %s\n", item.result.Docset))
	b.WriteString(fmt.Sprintf("Score: %.2f\n", item.result.Score))
	b.WriteString(fmt.Sprintf("Path: %s\n", item.result.Path))

	return b.String()
}
