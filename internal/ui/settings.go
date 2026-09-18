package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/config"
)

// settingsItem is one on/off toggle in the Settings tab. get and set both
// go through Options, the single source of truth the rest of Skrin reads,
// so a toggle takes effect at once; set also updates the config so it
// survives a restart.
type settingsItem struct {
	label, help string
	get         func(m *Model) bool
	set         func(m *Model, v bool)
}

func boolPtr(v bool) *bool { return &v }

// settingsItems are the toggles the Settings tab shows, in order. Each one
// already exists in config.toml for hand-editing; this is the same knobs,
// reachable without an editor.
func settingsItems() []settingsItem {
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
			help:  "c and C open the drawer; off hides it entirely, including from the manual and the status line.",
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
				m.renderedW = 0
				if m.split != nil {
					m.split.renderedW = 0
				}
				if m.editor != nil {
					m.editor.SetLineNumbers(v)
				}
			},
		},
		{
			label: "Show spreads",
			help:  "A ```spread (or ```dataview) block shows the table or list its query finds, kept current as notes change. Off shows the query as code.",
			get:   func(m *Model) bool { return m.opts.Spreads },
			set: func(m *Model, v bool) {
				m.opts.Spreads = v
				m.opts.Config.Render.Spreads = boolPtr(v)
				m.renderedW = 0
				if m.split != nil {
					m.split.renderedW = 0
				}
			},
		},
	}
}

// settingsKey handles a key press while the Settings tab is showing.
func (m *Model) settingsKey(k tea.KeyPressMsg) {
	items := settingsItems()
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
			it.set(m, !it.get(m))
			if err := config.Save(m.opts.Config); err != nil {
				m.Flash("couldn't save settings: " + err.Error())
			}
		}
	}
}

// settingsView renders the Settings tab: a table of toggles, the cursor's
// help text underneath.
func (m *Model) settingsView(w int) []string {
	items := settingsItems()
	h := m.manual
	var out []string
	out = append(out, "")
	for i, it := range items {
		box := "[ ]"
		if it.get(m) {
			box = "[x]"
		}
		row := box + "  " + it.label
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
	out = append(out, m.st.muted.Render("Saved to "+config.Path()+" as each toggle changes."))
	return out
}
