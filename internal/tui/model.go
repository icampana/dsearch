// Package tui provides an interactive terminal user interface for dsearch.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
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
	// Use explicit background colors to prevent black-on-black rendering
	docStyle          = lipgloss.NewStyle().Padding(1, 2).Background(lipgloss.Color("0"))
	titleStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true).Background(lipgloss.Color("0"))
	inputStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Background(lipgloss.Color("0"))
	placeholderStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(lipgloss.Color("0"))
	statusStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Italic(true).Background(lipgloss.Color("0"))
	selectedItemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Background(lipgloss.Color("0"))
	normalItemStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Background(lipgloss.Color("0"))
	previewStyle      = lipgloss.NewStyle().Padding(1).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238")).Background(lipgloss.Color("0"))
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
	engine       *search.Engine
	docsets      []docset.Docset
	input        textinput.Model
	list         list.Model
	viewport     viewport.Model
	loading      bool
	lastSearch   string
	width        int
	height       int
	showPreview  bool
	outputFormat render.Format
	previewText  string
	quit         bool
	err          error
}

// NewModel creates a new TUI model.
func NewModel(engine *search.Engine, docsets []docset.Docset) Model {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.PlaceholderStyle = placeholderStyle
	ti.TextStyle = inputStyle
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50

	// Disable Ctrl+C in textinput so it can be used for quitting
	ti.KeyMap.AcceptSuggestion = key.NewBinding(key.WithDisabled())
	// Keep other default keybindings but ensure Ctrl+C passes through

	delegate := list.NewDefaultDelegate()
	delegate.SetSpacing(0)
	delegate.Styles.SelectedTitle = selectedItemStyle
	delegate.Styles.SelectedDesc = selectedItemStyle
	delegate.Styles.NormalTitle = normalItemStyle
	delegate.Styles.NormalDesc = normalItemStyle

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Search Results"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()

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
	// Handle quit keys at the very top level, before any other processing
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// Check for quit keys by both Type and String representation
		if keyMsg.Type == tea.KeyCtrlC || keyMsg.Type == tea.KeyEsc {
			return m, tea.Quit
		}
		keyStr := keyMsg.String()
		if keyStr == "ctrl+c" || keyStr == "esc" {
			return m, tea.Quit
		}
		// 'q' to quit when input is not focused or empty
		if keyStr == "q" && (!m.input.Focused() || m.input.Value() == "") {
			return m, tea.Quit
		}
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If there's an error, allow Enter to dismiss it
		if m.err != nil && msg.Type == tea.KeyEnter {
			m.err = nil
			return m, nil
		}

		switch msg.Type {
		case tea.KeyEnter:
			if m.list.SelectedItem() != nil {
				if item, ok := m.list.SelectedItem().(resultItem); ok {
					return m, m.loadContentCmd(item.result)
				}
			}
			return m, nil

		case tea.KeyTab:
			m.showPreview = !m.showPreview
			m.updateDimensions()
			return m, nil

		case tea.KeyUp, tea.KeyDown:
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd

		case tea.KeyPgUp, tea.KeyPgDown:
			if m.showPreview {
				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)
				return m, cmd
			}
			return m, nil

		case tea.KeyHome:
			if m.showPreview {
				m.viewport.GotoTop()
			}
			return m, nil

		case tea.KeyEnd:
			if m.showPreview {
				m.viewport.GotoBottom()
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateDimensions()

	case searchFinishedMsg:
		m.loading = false
		m.lastSearch = msg.query

		items := make([]list.Item, len(msg.results))
		for i, result := range msg.results {
			items[i] = resultItem{result: result}
		}

		m.list.SetItems(items)
		m.list.ResetFilter()

		if len(items) > 0 {
			m.list.Select(0)
			return m, m.loadContentCmd(items[0].(resultItem).result)
		} else {
			m.previewText = ""
		}

	case contentLoadedMsg:
		m.previewText = msg.content

	case error:
		m.err = msg
		m.loading = false
		// When error occurs, stop processing other updates but still return
		return m, nil
	}

	// If there's an error, only process quit keys (handled above), skip everything else
	if m.err != nil {
		return m, nil
	}

	// Update input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	// Trigger search on typing
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyRunes {
		inputText := m.input.Value()
		if len(inputText) >= 2 && inputText != m.lastSearch {
			m.loading = true
			cmds = append(cmds, m.performSearchCmd(inputText))
		} else if len(inputText) == 0 {
			m.list.SetItems([]list.Item{})
			m.previewText = ""
		}
	}

	// Update viewport content
	if m.showPreview && m.previewText != "" {
		m.viewport.SetContent(m.previewText)
	}

	// Ensure dimensions are updated
	m.updateDimensions()

	return m, tea.Batch(cmds...)
}

// updateDimensions updates the dimensions of child components based on current window size.
func (m *Model) updateDimensions() {
	listWidth := m.width / 2
	listHeight := m.height - 6 // Account for title, search, status bars, and padding
	if listHeight < 5 {
		listHeight = 5
	}

	// Update input width
	inputWidth := m.width / 2
	if inputWidth > 100 {
		inputWidth = 100
	}
	m.input.Width = inputWidth

	// Update list dimensions
	m.list.SetSize(listWidth-2, listHeight)

	// Update viewport dimensions
	if m.showPreview {
		m.viewport.Width = listWidth - 4
		m.viewport.Height = listHeight - 2
	}
}

// View renders the TUI.
func (m Model) View() string {
	if m.err != nil {
		return docStyle.Render(
			fmt.Sprintf("Error: %v\n\nPress Enter to dismiss or Ctrl+C to quit", m.err),
		)
	}

	// Calculate dimensions (pure computation, no state mutation)
	listWidth := m.width / 2
	listHeight := m.height - 6 // Account for title, search, status bars, and padding
	if listHeight < 5 {
		listHeight = 5
	}

	// Build the header
	var header strings.Builder
	header.WriteString(titleStyle.Render("dsearch - Offline Documentation Search"))
	header.WriteString("\n\n")
	header.WriteString("Search: ")
	header.WriteString(m.input.View())
	header.WriteString("\n")

	if m.loading {
		header.WriteString(statusStyle.Render("Searching..."))
		header.WriteString("\n")
	}

	// Render left pane (results list) - dimensions already set in updateDimensions()
	leftStyle := lipgloss.NewStyle().
		Width(listWidth).
		Height(listHeight).
		Background(lipgloss.Color("0"))
	leftPane := leftStyle.Render(m.list.View())

	// Render right pane (preview or empty) - dimensions already set in updateDimensions()
	var rightPane string
	if m.showPreview {
		previewContent := previewStyle.Render(m.viewport.View())
		rightStyle := lipgloss.NewStyle().
			Width(listWidth).
			Height(listHeight).
			Background(lipgloss.Color("0"))
		rightPane = rightStyle.Render(previewContent)
	} else {
		rightStyle := lipgloss.NewStyle().
			Width(listWidth).
			Height(listHeight).
			Background(lipgloss.Color("0"))
		rightPane = rightStyle.Render("")
	}

	// Join panes horizontally
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// Build help text
	helpText := "Tab: Toggle Preview | ↑↓: Navigate | Ctrl+C/Esc/q: Quit"
	if m.showPreview {
		helpText = "Tab: Hide Preview | ↑↓: Navigate | PgUp/PgDn: Scroll | Ctrl+C/Esc: Quit"
	}

	// Combine all sections
	var b strings.Builder
	b.WriteString(header.String())
	b.WriteString(content)
	b.WriteString("\n")
	b.WriteString(statusStyle.Render(helpText))

	return docStyle.Render(b.String())
}

// performSearchCmd performs a search and returns a command.
func (m Model) performSearchCmd(query string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(300 * time.Millisecond)
		results, err := m.engine.Search(query, nil)
		if err != nil {
			return err
		}
		return searchFinishedMsg{results: results, query: query}
	}
}

// loadContentCmd loads and renders the documentation content for a result.
func (m Model) loadContentCmd(result search.Result) tea.Cmd {
	return func() tea.Msg {
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

		content, err := ds.GetContent(result.Entry)
		if err != nil {
			return fmt.Errorf("reading content: %w", err)
		}

		renderer := render.New(m.outputFormat)
		rendered, err := renderer.Render(content)
		if err != nil {
			return fmt.Errorf("rendering content: %w", err)
		}

		return contentLoadedMsg{content: rendered}
	}
}

// SetOutputFormat sets the output format for the preview.
func (m *Model) SetOutputFormat(format render.Format) {
	m.outputFormat = format
}
