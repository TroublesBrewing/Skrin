package index

import (
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

var (
	headingRE  = regexp.MustCompile(`^ {0,3}(#{1,6})[ \t]+(.*?)(?:[ \t]+#+)?[ \t]*$`)
	wikiRE     = regexp.MustCompile(`(!?)\[\[([^\[\]]+?)\]\]`)
	mdLinkRE   = regexp.MustCompile(`(!?)\[([^\[\]]*)\]\(([^()\s]+)\)`)
	blockRE    = regexp.MustCompile(`(?:^|\s)\^([A-Za-z0-9-]+)\s*$`)
	codeSpanRE = regexp.MustCompile("`[^`]*`")
	tagRE      = regexp.MustCompile(`(?:^|\s)#([\p{L}\p{N}_/-]*[\p{L}_/-][\p{L}\p{N}_/-]*)`)
)

// note is what the index keeps about one markdown note.
type note struct {
	links    []Link
	headings []Heading
	aliases  []string
	tags     []string            // lower-case, without '#', from the body and the tags property
	props    map[string][]string // frontmatter, lower-case keys, values as written
	blocks   map[string]int
}

// parse reads links, headings, tags, properties, aliases and block ids
// from a note. Links and tags in code are ignored, as in Obsidian; links in
// frontmatter properties count.
func parse(content string) note {
	n := note{blocks: map[string]int{}}
	lines := strings.Split(content, "\n")
	frontEnd := frontmatterEnd(lines)
	fence := ""
	offset := 0
	var tags []string
	for i, raw := range lines {
		l := strings.TrimSuffix(raw, "\r")
		t := strings.TrimSpace(l)
		switch {
		case frontEnd > 0 && i == 0:
		case frontEnd > 0 && i < frontEnd:
			n.links = append(n.links, findLinks(l, i, offset)...)
		case frontEnd > 0 && i == frontEnd:
			var fmTags []string
			n.props, n.aliases, fmTags = frontmatter(strings.Join(lines[1:frontEnd], "\n"))
			tags = append(tags, fmTags...)
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
			tags = append(tags, inlineTags(l)...)
		}
		offset += len(raw) + 1
	}
	n.tags = unique(tags)
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

// maskCode blanks out code spans, keeping byte offsets.
func maskCode(l string) string {
	return codeSpanRE.ReplaceAllStringFunc(l, func(s string) string { return strings.Repeat(" ", len(s)) })
}

// findLinks finds the wikilinks and internal markdown links on one line.
// offset is the line's byte offset in the note.
func findLinks(l string, line, offset int) []Link {
	masked := maskCode(l)
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

func inlineTags(l string) []string {
	var out []string
	for _, m := range tagRE.FindAllStringSubmatch(maskCode(l), -1) {
		out = append(out, strings.ToLower(m[1]))
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

// frontmatter reads properties, aliases and tags from YAML frontmatter.
// Values keep the text as written, so dates stay "2026-09-15".
func frontmatter(front string) (props map[string][]string, aliases, tags []string) {
	var doc yaml.Node
	if yaml.Unmarshal([]byte(front), &doc) != nil || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, nil, nil
	}
	props = map[string][]string{}
	pairs := doc.Content[0].Content
	for k := 0; k+1 < len(pairs); k += 2 {
		key, val := strings.ToLower(pairs[k].Value), pairs[k+1]
		var vals []string
		switch val.Kind {
		case yaml.ScalarNode:
			if val.Tag != "!!null" && val.Value != "" {
				vals = []string{val.Value}
			}
		case yaml.SequenceNode:
			for _, c := range val.Content {
				if c.Kind == yaml.ScalarNode && c.Value != "" {
					vals = append(vals, c.Value)
				}
			}
		}
		props[key] = vals
		switch key {
		case "aliases", "alias":
			for _, v := range vals {
				if val.Kind != yaml.ScalarNode {
					aliases = append(aliases, v)
					continue
				}
				for _, a := range strings.Split(v, ",") {
					if a = strings.TrimSpace(a); a != "" {
						aliases = append(aliases, a)
					}
				}
			}
		case "tags", "tag":
			for _, v := range vals {
				for _, t := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' || unicode.IsSpace(r) }) {
					tags = append(tags, strings.ToLower(strings.TrimPrefix(t, "#")))
				}
			}
		}
	}
	return props, aliases, tags
}

func unique(s []string) []string {
	sort.Strings(s)
	var out []string
	for i, v := range s {
		if v != "" && (i == 0 || v != s[i-1]) {
			out = append(out, v)
		}
	}
	return out
}
