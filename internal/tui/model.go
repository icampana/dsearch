// Package tui provides an interactive terminal user interface for dsearch.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/icampana/dsearch/internal/docset"
	"github.com/icampana/dsearch/internal/render"
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

// contentLoadedMsg is sent when documentation content is loaded.
type contentLoadedMsg struct {
	content string
}

// Model represents the TUI application state.
type Model struct {
	// Dependencies
	engine  *search.Engine
	docsets []docset.Docset

	// UI Components
	input    textinput.Model
	list     list.Model
	viewport viewport.Model

	// State
	loading      bool
	lastSearch   string
	searchedOnce bool
	width        int
	height       int
	showPreview  bool

	// Configuration
	outputFormat render.Format

	// Content
	selectedResult *search.Result
	selectedDocset *docset.Docset
	previewText    string
	err            error
}

// NewModel creates a new TUI model with the given search engine and docsets.
func NewModel(engine *search.Engine, docsets []docset.Docset) Model {
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

	// Initialize viewport
	vp := viewport.New(0, 0)

	return Model{
		engine:       engine,
		docsets:      docsets,
		input:        ti,
		list:         l,
		viewport:     vp,
		loading:      false,
		showPreview:  true,
		outputFormat: render.FormatText,
		width:        80,
		height:       24,
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
					m.selectedResult = &item.result
					return m, m.loadContentCmd(item.result)
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

		case tea.KeyPgUp, tea.KeyPgDown:
			// Handle viewport scrolling in preview
			if m.showPreview {
				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)
				cmds = append(cmds, cmd)
				return m, tea.Batch(cmds...)
			}
			return m, nil

		case tea.KeyHome, tea.KeyEnd:
			// Handle viewport scrolling
			if m.showPreview {
				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)
				cmds = append(cmds, cmd)
				return m, tea.Batch(cmds...)
			}
			return m, nil

		case tea.KeyCtrlU:
			// Scroll to top
			if m.showPreview {
				m.viewport.GotoTop()
				return m, nil
			}

		case tea.KeyCtrlD:
			// Scroll to bottom
			if m.showPreview {
				m.viewport.GotoBottom()
				return m, nil
			}
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
			listHeight = msg.Height/2 - 2
		}

		m.list.SetSize(msg.Width-4, listHeight)

		// Update viewport size
		previewHeight := msg.Height/2 - 2
		m.viewport.Width = msg.Width - 6
		m.viewport.Height = previewHeight

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
			item := items[0].(resultItem)
			m.selectedResult = &item.result
			cmds = append(cmds, m.loadContentCmd(item.result))
		} else {
			m.selectedResult = nil
			m.previewText = ""
		}

	case contentLoadedMsg:
		// Handle content load completion
		m.previewText = msg.content

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
			m.selectedResult = nil
			m.previewText = ""
		}
	}

	// Update list
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	// Update viewport
	if m.showPreview && m.previewText != "" {
		m.viewport.SetContent(m.previewText)
		// Only call viewport.Update for non-Window messages
		// WindowSizeMsg is handled separately above
		var vpCmd tea.Cmd
		if _, isWindowMsg := msg.(tea.WindowSizeMsg); !isWindowMsg {
			m.viewport, vpCmd = m.viewport.Update(msg)
			cmds = append(cmds, vpCmd)
		}
	}

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

	// Results and Preview sections
	if len(m.list.Items()) > 0 {
		if m.showPreview {
			// Split view: list on top, preview on bottom
			b.WriteString(m.list.View())

			// Preview panel
			b.WriteString(previewStyle.Render(m.viewport.View()))
		} else {
			// Full screen list
			b.WriteString(m.list.View())
		}
		b.WriteString("\n")

		// Help text
		if !m.showPreview {
			b.WriteString(statusStyle.Render("Tab: Show Preview | Ctrl+C: Quit"))
		} else {
			b.WriteString(statusStyle.Render("Tab: Hide Preview | ↑↓: Scroll | Ctrl+U/D: Top/Bottom | Ctrl+C: Quit"))
		}
	} else if !m.searchedOnce {
		b.WriteString(statusStyle.Render("Type to search (min 2 characters)\n\n"))
		b.WriteString(statusStyle.Render("Ctrl+C: Quit | Tab: Toggle Preview"))
	} else {
		b.WriteString(statusStyle.Render("Ctrl+C: Quit"))
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

// loadContentCmd loads and renders the documentation content for a result.
func (m Model) loadContentCmd(result search.Result) tea.Cmd {
	return func() tea.Msg {
		// Find the docset for this result
		var ds *docset.Docset
		for i := range m.docsets {
			if m.docsets[i].Name == result.Docset {
				ds = &m.docsets[i]
				break
			}
		}

		if ds == nil {
			return fmt.Errorf("docset not found: %s", result.Docset)
		}

		// Get HTML content from the docset (result embeds Entry)
		content, err := ds.GetContent(result.Entry)
		if err != nil {
			return fmt.Errorf("reading content: %w", err)
		}

		// Render the content using the configured format
		renderer := render.New(m.outputFormat)
		rendered, err := renderer.Render(content)
		if err != nil {
			return fmt.Errorf("rendering content: %w", err)
		}

		return contentLoadedMsg{
			content: rendered,
		}
	}
}

// SetOutputFormat sets the output format for the preview.
func (m *Model) SetOutputFormat(format render.Format) {
	m.outputFormat = format
}
