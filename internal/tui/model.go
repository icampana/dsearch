

// View renders a TUI with side-by-side layout (list left, preview right).
func (m Model) View() string {
	if m.err != nil {
		return docStyle.Render(
			fmt.Sprintf("Error: %v

Press Ctrl+C to quit", m.err),
		)
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("dsearch - Offline Documentation Search

"))

	// Input section
	b.WriteString("Search: ")
	b.WriteString(m.input.View())
	b.WriteString("

")

	// Status
	if m.loading {
		b.WriteString(statusStyle.Render("Searching...

"))
	} else if m.searchedOnce && len(m.list.Items()) == 0 {
		b.WriteString(statusStyle.Render("No results found

"))
	}

	// Results and Preview sections
	if len(m.list.Items()) > 0 {
		if m.showPreview {
			// Side-by-side layout: list (left) + preview (right)
			listWidth := m.width * 6 / 10
			previewWidth := m.width - listWidth - 2

			// Calculate list height (account for title and input)
			listHeight := m.height - 6
			if m.showPreview {
				listHeight = listHeight / 2 - 1
			}

			// Calculate preview height
			previewHeight := m.height - 6
			if m.showPreview {
				previewHeight = previewHeight - 2
			}

			// Build side-by-side layout manually
			leftSide := strings.Builder{}
			for i, line := range strings.Split(listStr, "
") {
				if i > 0 {
					leftSide.WriteString("
")
				}
				leftSide.WriteString(line)
			}
			leftStr := leftSide.String()

			rightSide := strings.Builder{}
			for i, line := range strings.Split(previewStr, "
") {
				if i > 0 {
					rightSide.WriteString("
")
				}
				rightSide.WriteString(line)
			}
			rightStr := rightSide.String()

			// Join side-by-side with vertical alignment
			maxLines := max(strings.Count(leftStr), strings.Count(rightStr))
			for i := 0; i < maxLines; i++ {
				leftLine := ""
				rightLine := ""
				if i < len(strings.Split(leftStr, "
")) {
					leftLine = strings.Split(leftStr, "
")[i]
				}
				if i < len(strings.Split(rightStr, "
")) {
					rightLine = strings.Split(rightStr, "
")[i]
				}

				b.WriteString(leftLine)
				leftPad := previewHeight - strings.Count(leftLine)
				for j := 0; j < leftPad; j++ {
					b.WriteString(" ")
				}

				b.WriteString(rightLine)
			}

			// Help text below list
			helpText := statusStyle.Render("Tab: Hide Preview | Ctrl+C: Quit")
			b.WriteString("
")
			b.WriteString(helpText)
		} else {
			// Full screen list (preview hidden)
			b.WriteString(m.list.View())
			b.WriteString("

")

			// Help text
			helpText := statusStyle.Render("Tab: Show Preview | Ctrl+C: Quit")
			b.WriteString("
")
			b.WriteString(helpText)
		}
	} else if !m.searchedOnce {
		b.WriteString(statusStyle.Render("Type to search (min 2 characters)

"))
		b.WriteString(statusStyle.Render("Ctrl+C: Quit | Tab: Toggle Preview"))
	} else {
		b.WriteString(statusStyle.Render("Ctrl+C: Quit"))
	}

	return docStyle.Render(b.String())
}
