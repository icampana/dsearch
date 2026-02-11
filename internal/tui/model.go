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
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.quit = true
			return m, tea.Quit

		case tea.KeyEnter:
			if m.list.SelectedItem() != nil {
				if item, ok := m.list.SelectedItem().(resultItem); ok {
					return m, m.loadContentCmd(item.result)
				}
			}
			return m, nil

		case tea.KeyTab:
			m.showPreview = !m.showPreview
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

		inputWidth := msg.Width / 2
		if inputWidth > 100 {
			inputWidth = 100
		}
		m.input.Width = inputWidth

		listWidth := msg.Width / 2
		listHeight := msg.Height - 4
		if m.showPreview {
			listHeight = listHeight - 1
		}

		m.list.SetSize(listWidth, listHeight)

		if m.showPreview {
			m.viewport.Width = listWidth
			m.viewport.Height = listHeight
		}

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

	b.WriteString(titleStyle.Render("dsearch - Offline Documentation Search\n\n"))

	b.WriteString("Search: ")
	b.WriteString(m.input.View())
	b.WriteString("\n")

	if m.loading {
		b.WriteString(statusStyle.Render("Searching...\n"))
	}

	listWidth := m.width / 2
	listHeight := m.height - 3

	leftStyle := lipgloss.NewStyle().Width(listWidth)

	b.WriteString(leftStyle.Render(m.list.View()))

	if m.showPreview && m.previewText != "" {
		rightStyle := lipgloss.NewStyle().Width(listWidth)
		b.WriteString(rightStyle.Render(previewStyle.Render(m.viewport.View())))
	} else {
		rightStyle := lipgloss.NewStyle().Width(listWidth)
		b.WriteString(rightStyle.Render(lipgloss.NewStyle().Height(listHeight).Render("")))
	}

	b.WriteString("\n")

	helpText := "Tab: Toggle Preview | Ctrl+C: Quit"
	if m.showPreview {
		helpText = "Tab: Hide | ↑↓: Navigate | PgUp/PgDn: Scroll | Ctrl+C: Quit"
	} else {
		helpText = "Tab: Show Preview | ↑↓: Navigate | Ctrl+C: Quit"
	}
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
