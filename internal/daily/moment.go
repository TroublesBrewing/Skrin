package daily

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// tokens are the moment.js format tokens Format understands, longest first
// so that "YYYY" wins over "YY".
var tokens = []string{
	"YYYY", "GGGG", "gggg", "MMMM", "DDDD", "dddd",
	"MMM", "DDD", "ddd",
	"YY", "MM", "Do", "DD", "dd", "WW", "ww", "HH", "hh", "mm", "ss",
	"Q", "M", "D", "d", "E", "e", "W", "w", "H", "h", "m", "s", "A", "a", "X",
}

// Format renders t with a moment.js format string, the date syntax
// Obsidian uses for daily note names and templates. Text in [brackets] is
// copied literally.
func Format(t time.Time, f string) string {
	var b strings.Builder
	for i := 0; i < len(f); {
		if f[i] == '[' {
			if j := strings.IndexByte(f[i:], ']'); j > 0 {
				b.WriteString(f[i+1 : i+j])
				i += j + 1
				continue
			}
		}
		matched := false
		for _, tok := range tokens {
			if strings.HasPrefix(f[i:], tok) {
				b.WriteString(token(t, tok))
				i += len(tok)
				matched = true
				break
			}
		}
		if !matched {
			r, n := utf8.DecodeRuneInString(f[i:])
			b.WriteRune(r)
			i += n
		}
	}
	return b.String()
}

func token(t time.Time, tok string) string {
	isoYear, week := t.ISOWeek()
	h12 := t.Hour() % 12
	if h12 == 0 {
		h12 = 12
	}
	two := func(n int) string { return fmt.Sprintf("%02d", n) }
	switch tok {
	case "YYYY":
		return fmt.Sprintf("%04d", t.Year())
	case "YY":
		return two(t.Year() % 100)
	case "GGGG", "gggg":
		return fmt.Sprintf("%04d", isoYear)
	case "Q":
		return strconv.Itoa((int(t.Month())-1)/3 + 1)
	case "MMMM":
		return t.Month().String()
	case "MMM":
		return t.Month().String()[:3]
	case "MM":
		return two(int(t.Month()))
	case "M":
		return strconv.Itoa(int(t.Month()))
	case "DDDD":
		return fmt.Sprintf("%03d", t.YearDay())
	case "DDD":
		return strconv.Itoa(t.YearDay())
	case "DD":
		return two(t.Day())
	case "Do":
		return ordinal(t.Day())
	case "D":
		return strconv.Itoa(t.Day())
	case "dddd":
		return t.Weekday().String()
	case "ddd":
		return t.Weekday().String()[:3]
	case "dd":
		return t.Weekday().String()[:2]
	case "d", "e":
		return strconv.Itoa(int(t.Weekday()))
	case "E":
		return strconv.Itoa((int(t.Weekday())+6)%7 + 1)
	case "WW", "ww":
		return two(week)
	case "W", "w":
		return strconv.Itoa(week)
	case "HH":
		return two(t.Hour())
	case "H":
		return strconv.Itoa(t.Hour())
	case "hh":
		return two(h12)
	case "h":
		return strconv.Itoa(h12)
	case "mm":
		return two(t.Minute())
	case "m":
		return strconv.Itoa(t.Minute())
	case "ss":
		return two(t.Second())
	case "s":
		return strconv.Itoa(t.Second())
	case "A":
		if t.Hour() < 12 {
			return "AM"
		}
		return "PM"
	case "a":
		if t.Hour() < 12 {
			return "am"
		}
		return "pm"
	case "X":
		return strconv.FormatInt(t.Unix(), 10)
	}
	return tok
}

func ordinal(n int) string {
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return strconv.Itoa(n) + suffix
}
