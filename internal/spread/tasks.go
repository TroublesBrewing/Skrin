package spread

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// taskRow is one task a TASK spread shows at the top of its own tree.
type taskRow struct {
	n    *Note
	i    int     // index into n.Tasks
	keys []value // the sort keys, in order
}

// runTasks answers a TASK spread. FROM picks notes, WHERE picks tasks,
// each seeing its own fields (text, status, completed, checked, line and
// its inline fields) before its note's. A task shows with everything
// nested under it; one whose parent already shows isn't repeated.
func (q *Query) runTasks(v Vault, from string) Result {
	c := newCtx(v, from)
	notes := v.Notes()
	sort.Slice(notes, func(i, j int) bool { return notes[i].Rel < notes[j].Rel })
	var rows []taskRow
	for i := range notes {
		n := &notes[i]
		if len(n.Tasks) == 0 || q.from != nil && !q.from.match(c, n) {
			continue
		}
		match := make([]bool, len(n.Tasks))
		for j := range n.Tasks {
			c.task = &n.Tasks[j]
			match[j] = q.where == nil || truthy(q.where.eval(c, n))
		}
		for j := range n.Tasks {
			if !match[j] || ancestorMatches(n.Tasks, j, match) {
				continue
			}
			c.task = &n.Tasks[j]
			r := taskRow{n: n, i: j}
			for _, k := range q.sort {
				r.keys = append(r.keys, k.e.eval(c, n))
			}
			rows = append(rows, r)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		for k, key := range q.sort {
			if d := order(rows[i].keys[k], rows[j].keys[k], key.desc); d != 0 {
				return d < 0
			}
		}
		return false
	})
	if q.Limit >= 0 && len(rows) > q.Limit {
		rows = rows[:q.Limit]
	}
	if len(rows) == 0 {
		return Result{Note: "No tasks match"}
	}
	var note string
	if q.Limit < 0 && len(rows) > maxRows {
		note = fmt.Sprintf("+%d more · add LIMIT or narrow FROM", len(rows)-maxRows)
		rows = rows[:maxRows]
	}
	return Result{Markdown: tasksMarkdown(rows), Note: note}
}

func ancestorMatches(tasks []Task, j int, match []bool) bool {
	for p := tasks[j].Parent; p >= 0; p = tasks[p].Parent {
		if match[p] {
			return true
		}
	}
	return false
}

// tasksMarkdown lists the tasks under a link to each note they're in.
// Notes come in the order of their first task, so a SORT still decides
// which note leads; within a note, tasks keep the SORT's order.
func tasksMarkdown(rows []taskRow) string {
	var order []*Note
	byNote := map[*Note][]int{}
	for _, r := range rows {
		if _, ok := byNote[r.n]; !ok {
			order = append(order, r.n)
		}
		byNote[r.n] = append(byNote[r.n], r.i)
	}
	var b strings.Builder
	for k, n := range order {
		if k > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(linkTo(n.Rel) + "\n")
		for _, i := range byNote[n] {
			writeTask(&b, n, i, 0)
		}
	}
	return b.String()
}

func writeTask(b *strings.Builder, n *Note, i, depth int) {
	t := n.Tasks[i]
	b.WriteString(strings.Repeat("    ", depth) + "- [" + t.Status + "] ")
	if t.Text != "" {
		b.WriteString(t.Text + " ")
	}
	b.WriteString(lineLink(n.Rel, t.Line) + "\n")
	for j := i + 1; j < len(n.Tasks); j++ {
		if n.Tasks[j].Parent == i {
			writeTask(b, n, j, depth+1)
		}
	}
}

// lineLink is a small "↗" link that opens rel at a line — the index's own
// "#:n" anchor, counted from 1. It lives only in the rendered answer;
// nothing writes it into a note.
func lineLink(rel string, line int) string {
	return "[[" + strings.TrimSuffix(rel, ".md") + "#:" + strconv.Itoa(line+1) + "|↗]]"
}
