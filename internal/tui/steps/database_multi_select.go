package steps

import (
	tea "github.com/charmbracelet/bubbletea"
)

// dbMultiOption represents one entry in the database multi-select list.
type dbMultiOption struct {
	ID        string // registry ID
	Name      string // display name
	Exclusive bool   // true for None and SQLite: selecting deselects all others
	Checked   bool
}

// DatabaseMultiSelect is a checkbox-style step for selecting one or more databases.
// Space toggles; Enter confirms. None and SQLite are exclusive with other options.
type DatabaseMultiSelect struct {
	prompt  string
	options []dbMultiOption
	cursor  int
}

// DatabaseMultiOption describes one database option passed to NewDatabaseMultiSelect.
type DatabaseMultiOption struct {
	ID        string
	Name      string
	Exclusive bool // true for None and SQLite
}

// NewDatabaseMultiSelect creates a DatabaseMultiSelect step.
// "None" is prepended automatically as the first exclusive option.
func NewDatabaseMultiSelect(dbs []DatabaseMultiOption) DatabaseMultiSelect {
	opts := make([]dbMultiOption, 0, len(dbs)+1)
	// None is always first
	opts = append(opts, dbMultiOption{ID: "", Name: "None", Exclusive: true, Checked: true})
	for _, db := range dbs {
		opts = append(opts, dbMultiOption{
			ID:        db.ID,
			Name:      db.Name,
			Exclusive: db.Exclusive,
		})
	}
	return DatabaseMultiSelect{
		prompt:  "Select your databases:",
		options: opts,
		cursor:  0,
	}
}

func (m DatabaseMultiSelect) Init() tea.Cmd { return nil }

func (m DatabaseMultiSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case " ":
			m.options = toggle(m.options, m.cursor)
		case "enter":
			if !anyChecked(m.options) {
				return m, nil
			}
			return m, func() tea.Msg {
				return StepDone{Value: selectedIDs(m.options)}
			}
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m DatabaseMultiSelect) View() string {
	s := stylePrompt.Render(m.prompt) + "\n"
	s += styleMuted.Render("(space to toggle, enter to confirm)") + "\n\n"
	for i, opt := range m.options {
		checkbox := "[ ]"
		if opt.Checked {
			checkbox = "[x]"
		}
		line := checkbox + " " + opt.Name
		if i == m.cursor {
			s += styleCursor.Render("›") + " " + styleSelected.Render(line) + "\n"
		} else {
			s += "  " + styleOption.Render(line) + "\n"
		}
	}
	return s + "\n" + styleMuted.Render("↑↓ navigate  ·  space to toggle  ·  enter to confirm  ·  ctrl+c to exit") + "\n"
}

// toggle handles the exclusivity rules and returns an updated options slice.
func toggle(opts []dbMultiOption, idx int) []dbMultiOption {
	result := make([]dbMultiOption, len(opts))
	copy(result, opts)

	target := result[idx]
	newChecked := !target.Checked

	if newChecked {
		if target.Exclusive {
			// Selecting an exclusive option deselects everything else.
			for i := range result {
				result[i].Checked = false
			}
		} else {
			// Selecting a non-exclusive option deselects all exclusive options.
			for i := range result {
				if result[i].Exclusive {
					result[i].Checked = false
				}
			}
		}
	}
	result[idx].Checked = newChecked
	return result
}

// anyChecked reports whether at least one option is checked.
func anyChecked(opts []dbMultiOption) bool {
	for _, o := range opts {
		if o.Checked {
			return true
		}
	}
	return false
}

// selectedIDs returns the registry IDs of all checked, non-exclusive options.
// If only None (exclusive, ID="") is checked, returns an empty slice.
func selectedIDs(opts []dbMultiOption) []string {
	var ids []string
	for _, o := range opts {
		if o.Checked && !o.Exclusive {
			ids = append(ids, o.ID)
		}
	}
	return ids
}
