package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lithammer/fuzzysearch/fuzzy"
)

const (
	INSERT = iota
	SEARCH
	UNACTIVE
)

type model struct {
	all       []string
	choices   []string
	cursor    int
	selected  map[int]struct{}
	textInput textinput.Model
	err       error
	T1        int64
}

func remove(slice []string, s int) []string {
	return append(slice[:s], slice[s+1:]...)
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Maths"
	ti.CharLimit = 156
	ti.Width = 20
	var files []string
	root := os.Getenv("Notes")
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {

			fmt.Println(err)
			return nil
		}

		if !info.IsDir() && filepath.Ext(path) == ".md" {
			files = append(files, strings.TrimSuffix(filepath.Base(path), filepath.Ext(filepath.Base(path))))
		}

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
	return model{
		// Our to-do list is a grocery list
		choices:   files,
		all:       files,
		textInput: ti,
		err:       nil,
		// A map which indicates which choices are selected. We're using
		// the  map like a mathematical set. The keys refer to the indexes
		// of the `choices` slice, above.
		selected: make(map[int]struct{}),
		T1:       UNACTIVE,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:
		if m.textInput.Focused() && m.T1 == INSERT {
			switch msg.Type {
			case tea.KeyEnter:
				m.textInput.Blur()
				root := filepath.Join(os.Getenv("Notes"), strings.Trim(m.textInput.Value(), " ")+".md")
				m.all = append(m.all, strings.Trim(m.textInput.Value(), " "))
				m.choices = m.all
				m.T1 = UNACTIVE
				m.textInput.SetValue("")
				return m, openEditor([]string{root})
			case tea.KeyEsc, tea.KeyCtrlC:
				m.textInput.Blur()
				m.T1 = UNACTIVE
				m.textInput.SetValue("")
				return m, nil
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
		if m.textInput.Focused() && m.T1 == SEARCH {
			switch msg.Type {
			case tea.KeyEnter:
				m.textInput.Blur()
				m.T1 = UNACTIVE
				m.choices = fuzzy.FindNormalizedFold(m.textInput.Value(), m.all)
				m.cursor = 0
				m.textInput.SetValue("")
				return m, nil
			case tea.KeyEsc, tea.KeyCtrlC:
				m.textInput.Blur()
				m.T1 = UNACTIVE
				m.textInput.SetValue("")
				return m, nil
			}

			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd

		}
		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit
		case "a":
			if len(m.selected) != len(m.choices) {
				for i := range m.choices {
					m.selected[i] = struct{}{}
				}
			} else {
				m.selected = make(map[int]struct{})
			}
			return m, nil
		// The "up" and "k" keys move the cursor up
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "/":
			m.T1 = SEARCH
			m.textInput.Focus()
		// The "down" and "j" keys move the cursor down
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "d", "delete":

			for i := range m.selected {
				m.all = remove(m.all, i)
			}

			m.choices = m.all
		case "enter":
			if len(m.selected) < 1 {
				return m, openEditor([]string{m.choices[m.cursor]})
			}
			list := make([]string, 1)
			for i := range m.selected {
				list = append(list, m.choices[i])
			}

			m.selected = make(map[int]struct{})
			return m, openEditor(list)

		case "i":
			m.T1 = INSERT
			m.textInput.Focus()

		case ":":
			m.choices = m.all
			m.selected = make(map[int]struct{})

		// The "enter" key and the spacebar (a literal space) toggle
		// the selected state for the item that the cursor is pointing at.
		case " ":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}

	case error:
		m.err = msg
		return m, nil
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func openEditor(a []string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nvim"
	}
	c := exec.Command(editor, a...) //nolint:gosec
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return nil
	})
}

var ()

func (m model) View() string {
	// The header
	if m.textInput.Focused() && m.T1 == INSERT {
		return fmt.Sprintf("How should I name the note ?\n\n%s\n\n%s", m.textInput.View(), "(esc to quit)\n")
	}
	if m.textInput.Focused() && m.T1 == SEARCH {
		return fmt.Sprintf("How is the note named ?\n\n%s\n\n%s", m.textInput.View(), "(esc to quit)\n")
	}
	bar := lipgloss.NewStyle().Bold(true).Align(lipgloss.Left).Inline(true).Background(lipgloss.Color("#569C6C")).Foreground(lipgloss.Color("#F9D6CA"))
	s := fmt.Sprintf("\n %s%s\n\n", bar.Render(" What notes should i open ? "), bar.Reverse(true).Foreground(lipgloss.NoColor{}).Render(" Deleted Drill "))
	s += fmt.Sprintf(" %s \n\n", lipgloss.NewStyle().Foreground(lipgloss.Color("#B39393")).Render(strconv.Itoa(len(m.all))+" notes"))
	// Iterate over our choices
	for i, choice := range m.choices {

		// Is the cursor pointing at this choice?
		cursor := " " // no cursor
		if m.cursor == i {
			cursor = ">" // cursor!
		}

		// Is this choice selected?
		style := lipgloss.NewStyle().Foreground(lipgloss.NoColor{}) // not selected
		if _, ok := m.selected[i]; ok {
			style.Foreground(lipgloss.Color("#FF6969")) // selected!
		}

		// Render the row
		s += fmt.Sprintf("%s %s\n\n", lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6969")).Render(cursor), style.Render(choice))
	}

	// The footer
	s += fmt.Sprintf("%s", lipgloss.NewStyle().Foreground(lipgloss.Color("#B39393")).Render("\nPress q to quit.\n"))
	// Send the UI for rendering
	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Shit another silly mistake: %v", err)
		os.Exit(1)
	}
}
