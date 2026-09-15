package index

import (
	"net/url"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	headingRE  = regexp.MustCompile(`^ {0,3}(#{1,6})[ \t]+(.*?)(?:[ \t]+#+)?[ \t]*$`)
	wikiRE     = regexp.MustCompile(`(!?)\[\[([^\[\]]+?)\]\]`)
	mdLinkRE   = regexp.MustCompile(`(!?)\[([^\[\]]*)\]\(([^()\s]+)\)`)
	blockRE    = regexp.MustCompile(`(?:^|\s)\^([A-Za-z0-9-]+)\s*$`)
	codeSpanRE = regexp.MustCompile("`[^`]*`")
)

// note is what the index keeps about one markdown note.
type note struct {
	links    []Link
	headings []Heading
	aliases  []string
	blocks   map[string]int
}

// parse reads links, headings, aliases and block ids from a note. Links in
// code are ignored, as in Obsidian; links in frontmatter properties count.
func parse(content string) note {
	n := note{blocks: map[string]int{}}
	lines := strings.Split(content, "\n")
	frontEnd := frontmatterEnd(lines)
	fence := ""
	offset := 0
	for i, raw := range lines {
		l := strings.TrimSuffix(raw, "\r")
		t := strings.TrimSpace(l)
		switch {
		case frontEnd > 0 && i == 0:
		case frontEnd > 0 && i < frontEnd:
			n.links = append(n.links, findLinks(l, i, offset)...)
		case frontEnd > 0 && i == frontEnd:
			n.aliases = aliases(strings.Join(lines[1:frontEnd], "\n"))
		case fence != "":
			if strings.HasPrefix(t, fence) && strings.Trim(t, fence[:1]) == "" {
				fence = ""
			}
		case strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~"):
			fence = t[:len(t)-len(strings.TrimLeft(t, t[:1]))]
		default:
			if m := headingRE.FindStringSubmatch(l); m != nil {
				n.headings = append(n.headings, Heading{Level: len(m[1]), Text: m[2], Line: i})
			}
			if m := blockRE.FindStringSubmatch(l); m != nil {
				n.blocks[m[1]] = i
			}
			n.links = append(n.links, findLinks(l, i, offset)...)
		}
		offset += len(raw) + 1
	}
	return n
}

func frontmatterEnd(lines []string) int {
	if len(lines) < 2 || strings.TrimRight(lines[0], " \r") != "---" {
		return 0
	}
	for i := 1; i < len(lines); i++ {
		if t := strings.TrimRight(lines[i], " \r"); t == "---" || t == "..." {
			return i
		}
	}
	return 0
}

// findLinks finds the wikilinks and internal markdown links on one line.
// offset is the line's byte offset in the note.
func findLinks(l string, line, offset int) []Link {
	// Blank out code spans, keeping byte offsets.
	masked := codeSpanRE.ReplaceAllStringFunc(l, func(s string) string { return strings.Repeat(" ", len(s)) })
	context := strings.TrimSpace(l)
	var out []Link
	for _, m := range wikiRE.FindAllStringSubmatchIndex(masked, -1) {
		lk := Link{Embed: m[3] > m[2], Line: line, Start: offset + m[0], End: offset + m[1], Context: context}
		lk.Target, lk.Sub, lk.Alias, lk.Sep, lk.HasAlias = splitWiki(l[m[4]:m[5]])
		out = append(out, lk)
	}
	for _, m := range mdLinkRE.FindAllStringSubmatchIndex(masked, -1) {
		u := l[m[6]:m[7]]
		if strings.Contains(u, "://") || strings.HasPrefix(u, "#") || strings.HasPrefix(u, "mailto:") {
			continue
		}
		dec, err := url.PathUnescape(u)
		if err != nil {
			dec = u
		}
		target, sub, _ := strings.Cut(dec, "#")
		out = append(out, Link{
			Markdown: true, Embed: m[3] > m[2], Target: target, Sub: sub,
			Alias: l[m[4]:m[5]], HasAlias: true,
			Line: line, Start: offset + m[0], End: offset + m[1], Context: context,
		})
	}
	return out
}

// splitWiki splits the inside of [[...]] into target, #sub and |alias.
// Inside tables Obsidian writes the alias separator as \|.
func splitWiki(inner string) (target, sub, alias, sep string, hasAlias bool) {
	sep = "|"
	target = inner
	if i := strings.Index(inner, `\|`); i >= 0 {
		sep, target, alias, hasAlias = `\|`, inner[:i], inner[i+2:], true
	} else if i := strings.Index(inner, "|"); i >= 0 {
		target, alias, hasAlias = inner[:i], inner[i+1:], true
	}
	target, sub, _ = strings.Cut(target, "#")
	return strings.TrimSpace(target), strings.TrimSpace(sub), alias, sep, hasAlias
}

// aliases reads the aliases property from frontmatter.
func aliases(front string) []string {
	var props map[string]any
	if yaml.Unmarshal([]byte(front), &props) != nil {
		return nil
	}
	var out []string
	for _, key := range []string{"aliases", "alias"} {
		switch v := props[key].(type) {
		case string:
			for _, a := range strings.Split(v, ",") {
				if a = strings.TrimSpace(a); a != "" {
					out = append(out, a)
				}
			}
		case []any:
			for _, a := range v {
				if s, ok := a.(string); ok && s != "" {
					out = append(out, s)
				}
			}
		}
	}
	return out
}
