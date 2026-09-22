package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/config"
	"github.com/lurioso/skrin/internal/theme"
	"github.com/lurioso/skrin/internal/version"
)

// settingsItem is one row of the Settings tab: an on/off toggle, or a
// choice such as a folder. A toggle's get and set go through Options, the
// single source of truth the rest of Skrin reads, so it takes effect at
// once; set also updates the config so it survives a restart.
type settingsItem struct {
	label, help string
	get         func(m *Model) bool
	set         func(m *Model, v bool)
	// A row that holds a choice rather than on/off has value, what it's
	// set to now, and pick, which lets you choose; get and set are nil.
	value func(m *Model) string
	pick  func(m *Model)
	// beta marks an experiment: shown only in beta mode, off unless
	// switched on there, and gone entirely from a build that doesn't
	// allow beta. See version.Beta.
	beta bool
	// vault marks a vault setting: one that belongs to this vault alone
	// and is saved to the vault's own settings file, never config.toml.
	// It sits in its own block, apart from the global settings.
	vault bool
}

func boolPtr(v bool) *bool { return &v }

// settingsItems are the toggles the Settings tab shows, in order. Each one
// already exists in config.toml for hand-editing; this is the same knobs,
// reachable without an editor.
// settingsItems are the rows Settings shows, in order, for this model:
// the ordinary ones, then the beta block when the build allows it, and
// the experiments themselves only once beta mode is on.
func (m *Model) settingsItems() []settingsItem {
	items := ordinarySettings()
	if !version.Beta {
		return items
	}
	items = append(items, settingsItem{
		label: "Beta features",
		help:  "Opens the experiments below: things being tried out, which Skrin doesn't promise to keep. Off by default, and a release can leave every one of them out at once.",
		get:   func(m *Model) bool { return m.opts.Beta },
		set: func(m *Model, v bool) {
			m.opts.Beta = v
			m.opts.Config.Beta.Enabled = boolPtr(v)
		},
		beta: true,
	})
	if !m.opts.Beta {
		return items
	}
	return append(items, betaSettings()...)
}

// betaSettings are the experiments. Each one is off until switched on
// here, and off regardless in a build with version.Beta false.
func betaSettings() []settingsItem {
	return []settingsItem{
		{
			label: "Habit tracker",
			help:  "T shows today's habits, the week and the month, read from the ### Habits block of your daily note. An experiment: either it grows into something that stands on its own, or it goes. Your checkboxes are plain markdown and stay as they are either way.",
			get:   func(m *Model) bool { return m.opts.Habits },
			set: func(m *Model, v bool) {
				m.opts.Habits = v
				m.opts.Config.Habits.Enabled = boolPtr(v)
			},
			beta: true,
		},
	}
}

func ordinarySettings() []settingsItem {
	return []settingsItem{
		{
			label: "Remember last open note",
			help:  "Reopen the note that was open when Skrin last quit. Off by default: a fresh run starts at the welcome screen, so a vault with private notes never opens one by surprise.",
			get:   func(m *Model) bool { return m.opts.RestoreLastNote },
			set: func(m *Model, v bool) {
				m.opts.RestoreLastNote = v
				m.opts.Config.General.RestoreLastNote = boolPtr(v)
			},
		},
		{
			label: "Open notes on the cursor, not only on l/→ or Enter",
			help:  "On, moving the Files cursor opens the note under it at once — instant open. Off is Obsidian's way: the note pane only changes when you open one explicitly (l/→ to read, Enter to edit), and otherwise keeps showing whatever was open last.",
			get:   func(m *Model) bool { return m.opts.InstantOpen },
			set: func(m *Model, v bool) {
				m.opts.InstantOpen = v
				m.opts.Config.General.InstantOpen = boolPtr(v)
			},
		},
		{
			label: "Colours when there is no system theme",
			help:  "The palette Skrin carries itself, for a machine with no Omarchy theme to follow — a Mac, say. gruvbox is dark, flexoki-light is paper and ink. A system theme always wins over this. Enter chooses.",
			value: func(m *Model) string { return m.opts.Config.ThemeBuiltin() },
			pick:  (*Model).pickBuiltinTheme,
		},
		{
			label: "Cursor blink",
			help:  "How fast the input cursor blinks, everywhere there is a field to type in — the editor, quick note, go-to-a-note, the Book Card. off leaves it steady; medium is the default. Enter chooses.",
			value: func(m *Model) string { return m.opts.CursorBlink },
			pick:  (*Model).pickCursorBlink,
		},
		{
			label: "Carry over yesterday's todos",
			help:  "t carries unfinished todos from the last daily note into a new one.",
			get:   func(m *Model) bool { return m.opts.RolloverTodos },
			set: func(m *Model, v bool) {
				m.opts.RolloverTodos = v
				m.opts.Config.Daily.RolloverTodos = boolPtr(v)
			},
		},
		{
			label: "Vim keys in the built-in editor",
			help:  "hjkl, dd and friends inside e.",
			get:   func(m *Model) bool { return m.opts.Vim },
			set: func(m *Model, v bool) {
				m.opts.Vim = v
				m.opts.Config.Editor.Vim = v
			},
		},
		{
			label: "Claude drawer",
			help:  "c and C open the drawer; off hides it from the palette, the tips and the status line, and the keys say it is off.",
			get:   func(m *Model) bool { return m.opts.Assistant.Enabled },
			set: func(m *Model, v bool) {
				m.opts.Assistant.Enabled = v
				m.opts.Config.Assistant.Enabled = boolPtr(v)
			},
		},
		{
			label: "Image previews",
			help:  "Block-art previews of image embeds. Off shows the placeholder frame only; the embed's name, dimensions and size still show either way.",
			get:   func(m *Model) bool { return m.opts.Images },
			set: func(m *Model, v bool) {
				m.opts.Images = v
				m.opts.Config.Render.Images = boolPtr(v)
			},
		},
		{
			label: "Line numbers in notes",
			help:  "Show line numbers along the left edge of notes and the editor. L in main toggles this too.",
			get:   func(m *Model) bool { return m.opts.LineNumbers },
			set: func(m *Model, v bool) {
				m.opts.LineNumbers = v
				m.opts.Config.Render.LineNumbers = boolPtr(v)
				m.rerender()
				if m.editor != nil {
					m.editor.SetLineNumbers(v)
				}
			},
		},
		{
			label: "Readable line length",
			help:  "Notes stay at most 80 characters wide, centred in their pane, the way they read in zen and in Obsidian's setting of the same name. Off, they fill the pane however wide it is.",
			get:   func(m *Model) bool { return m.opts.ReadableWidth },
			set: func(m *Model, v bool) {
				m.opts.ReadableWidth = v
				m.opts.Config.Render.ReadableWidth = boolPtr(v)
				m.rerender()
			},
		},
		{
			label: "Show spreads",
			help:  "A ```spread (or ```dataview) block shows the table or list its query finds, kept current as notes change. Off shows the query as code.",
			get:   func(m *Model) bool { return m.opts.Spreads },
			set: func(m *Model, v bool) {
				m.opts.Spreads = v
				m.opts.Config.Render.Spreads = boolPtr(v)
				m.rerender()
			},
		},
		{
			label: "Templates folder",
			help:  "Where Insert template (Ctrl+P, \"template\") finds its templates: every note in this folder is one. Unset, it's the folder Obsidian's Templates plugin uses. Enter chooses a folder.",
			value: func(m *Model) string {
				f, whose := m.templatesFolder()
				if f == "" {
					return "none yet"
				}
				return f + " (" + whose + ")"
			},
			pick:  func(m *Model) { m.pickTemplatesFolder() },
			vault: true,
		},
		{
			label: "Folder templates",
			help:  "New notes start from a template paired with the folder they're created in. Add a folder and pick its template; enter on a row changes its template, d removes it. No pairing, the note is created empty. A folder matches exactly — Personer and Personer/Vänner can each have their own.",
			value: func(m *Model) string {
				n := len(m.opts.FolderTemplates)
				switch n {
				case 0:
					return "none"
				case 1:
					return "1 folder"
				default:
					return fmt.Sprintf("%d folders", n)
				}
			},
			pick:  func(m *Model) { m.pickFolderTemplates() },
			vault: true,
		},
	}
}

// settingsKey handles a key press while the Settings tab is showing.
func (m *Model) settingsKey(k tea.KeyPressMsg) {
	items := m.settingsItems()
	h := m.manual
	switch k.String() {
	case "esc", "q":
		m.manual = nil
	case "j", "down":
		h.setCur = min(h.setCur+1, len(items)-1)
	case "k", "up":
		h.setCur = max(h.setCur-1, 0)
	case "enter", " ", "space":
		if h.setCur < len(items) {
			it := items[h.setCur]
			if it.pick != nil {
				it.pick(m)
				return
			}
			it.set(m, !it.get(m))
			if err := config.Save(m.opts.Config); err != nil {
				m.Flash("couldn't save settings: " + err.Error())
			}
		}
	}
}

// settingsView renders the Settings tab: a table of toggles, the cursor's
// help text underneath. Vault settings stand apart under their own heading.
func (m *Model) settingsView(w int) []string {
	items := m.settingsItems()
	h := m.manual
	var out []string
	out = append(out, "")
	beta := false
	vault := false
	for i, it := range items {
		if it.beta && !beta {
			// The experiments stand apart, and say so once.
			beta = true
			out = append(out, "", m.st.muted.Render("BETA · experiments, which a release can leave out"))
		}
		if it.vault && !vault {
			// Vault settings belong to this vault alone, and are saved
			// to its own file, not config.toml.
			vault = true
			out = append(out, "", m.st.muted.Render("VAULT · this vault's own settings, saved to its "+config.SettingsFile))
		}
		var row string
		switch {
		case it.pick != nil:
			row = "[…]  " + it.label + ": " + it.value(m)
		case it.get(m):
			row = "[x]  " + it.label
		default:
			row = "[ ]  " + it.label
		}
		style := m.st.text
		if i == h.setCur {
			style = m.st.flash
		}
		out = append(out, style.Render(row))
	}
	out = append(out, "")
	if h.setCur < len(items) {
		for _, l := range wrap(items[h.setCur].help, w) {
			out = append(out, m.st.muted.Render(l))
		}
	}
	out = append(out, "")
	out = append(out, m.st.muted.Render("Global settings save to "+config.Path()+"; this vault's own settings to its "+config.SettingsFile+"."))
	return out
}

// pickBuiltinTheme is the Settings row for the palette Skrin falls back on
// when no system theme is there to follow. Choosing one takes effect at
// once on a machine that has no system theme; where one is in force it is
// kept for later, and the row says so.
func (m *Model) pickBuiltinTheme() {
	setCur := 0
	if m.manual != nil {
		setCur = m.manual.setCur
	}
	back := func() {
		m.openManual()
		m.manualGoTab(manualTabSettings)
		m.manual.setCur = setCur
	}
	var items []choice
	for _, name := range theme.Names() {
		items = append(items, choice{label: name, detail: builtinNote(name), do: func() {
			m.opts.Config.Theme.Builtin = name
			back()
			if err := config.Save(m.opts.Config); err != nil {
				m.flash = "couldn't save settings: " + err.Error()
				return
			}
			if strings.HasSuffix(m.pal.Name, "(built-in)") {
				m.setPalette(theme.Builtin(name))
				m.flash = "Colours: " + name
				return
			}
			m.flash = "Colours: " + name + " · your system theme (" + m.pal.Name + ") is in force, so this waits for a machine without one"
		}})
	}
	m.manual = nil
	m.openChooser(&chooser{
		title:  "Colours when there is no system theme",
		prompt: "Palette",
		empty:  "No palette by that name",
		verb:   "choose",
		items:  items,
		cancel: back,
	})
}

func builtinNote(name string) string {
	if theme.Builtin(name).Dark {
		return "dark"
	}
	return "light"
}
