package words

import (
	"strings"
	"unicode"

	"github.com/kljensen/snowball/english"
)

func Norm(phrase string) []string {
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return ' '
	}, phrase)

	tokens := strings.Fields(cleaned)

	seen := make(map[string]bool)
	var result []string

	for _, w := range tokens {
		if len(w) <= 2 {
			continue
		}
		stemmed := english.Stem(w, false)
		if !seen[stemmed] {
			seen[stemmed] = true
			result = append(result, stemmed)
		}
	}
	return result
}
