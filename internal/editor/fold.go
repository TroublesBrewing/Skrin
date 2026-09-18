package editor

// Fold is a run of lines the editor shows as other rows — a spread's
// answer — for as long as the cursor is outside it. Moving onto any of its
// lines opens it back into text; moving out folds it again. The cursor can
// never sit on a folded row, so nothing that moves the cursor needs to know
// folds exist: only what counts and draws rows does.
type Fold struct {
	Start, End int      // first and last line it covers, the fences
	Rows       []string // what shows instead, styled and fitted to the width; at least one
}

// SetFolds replaces the folds. They're recomputed from the text by the
// host whenever it changes, so stale ones are simply dropped here.
func (e *Editor) SetFolds(folds []Fold) {
	e.folds = e.folds[:0]
	for _, f := range folds {
		if f.Start >= 0 && f.End >= f.Start && f.End < len(e.lines) && len(f.Rows) > 0 {
			e.folds = append(e.folds, f)
		}
	}
	e.scroll()
}

// Line is the line the cursor is on.
func (e *Editor) Line() int { return e.row }

// folded reports the fold that row is in while it's folded — that is,
// while the cursor is outside it.
func (e *Editor) folded(row int) (Fold, bool) {
	for _, f := range e.folds {
		if row >= f.Start && row <= f.End {
			if e.row >= f.Start && e.row <= f.End {
				return Fold{}, false
			}
			return f, true
		}
	}
	return Fold{}, false
}

// rowsOf is how many display rows line row takes: its wrapped segments,
// or, folded, all of the fold's rows on its first line and none after.
func (e *Editor) rowsOf(row int) int {
	if f, ok := e.folded(row); ok {
		if row == f.Start {
			return len(f.Rows)
		}
		return 0
	}
	return len(e.segments(row))
}
