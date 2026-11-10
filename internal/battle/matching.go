package battle

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

var titleAnnotations = regexp.MustCompile(`\([^)]*\)|\[[^\]]*\]`)
var featuredArtist = regexp.MustCompile(`(?:^|[\s\p{Z}])(?:feat\.|ft\.|featuring)[\s\p{Z}]+.*$`)

const minFuzzyTitleRunes = 5

func normalizeTitle(title string) string {
	title = strings.ToLower(title)
	withoutAnnotations := titleAnnotations.ReplaceAllString(title, " ")
	if withoutCredit := canonicalTitle(featuredArtist.ReplaceAllString(withoutAnnotations, "")); utf8.RuneCountInString(withoutCredit) >= minFuzzyTitleRunes {
		return withoutCredit
	}
	if normalized := canonicalTitle(withoutAnnotations); normalized != "" {
		return normalized
	}
	return canonicalTitle(title)
}

func canonicalTitle(title string) string {
	var normalized strings.Builder
	for _, r := range norm.NFD.String(title) {
		switch {
		case unicode.Is(unicode.Mn, r):
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			normalized.WriteRune(r)
		case unicode.IsSpace(r):
			normalized.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(normalized.String()), " ")
}

func matchesTitle(guess, title string) bool {
	a, b := []rune(normalizeTitle(guess)), []rune(normalizeTitle(title))
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	if string(a) == string(b) {
		return true
	}
	if len(b) < minFuzzyTitleRunes {
		return false
	}
	tolerance := max(1, len(b)*15/100)
	if abs(len(a)-len(b)) > tolerance {
		return false
	}
	row := make([]int, len(b)+1)
	for j := range row {
		row[j] = j
	}
	for i, r := range a {
		previous := row[0]
		row[0] = i + 1
		for j, target := range b {
			diagonal := previous
			previous = row[j+1]
			cost := 1
			if r == target {
				cost = 0
			}
			row[j+1] = min(row[j]+1, previous+1, diagonal+cost)
		}
	}
	return row[len(b)] <= tolerance
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
