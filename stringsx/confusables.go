package stringsx

import (
	"bufio"
	"bytes"
	_ "embed"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Loads confusables mapping from https://www.unicode.org/Public/security/8.0.0/confusables.txt
//
//go:embed confusables.txt
var confusablesSrc []byte
var confusables map[rune]string
var confusablesPattern = regexp.MustCompile(`^([[:xdigit:]]{4,8})\s*;\s*((?:[[:xdigit:]]{4,8}\s+)+);\s*(\w+)\s+.*$`)

func init() {
	parseHex := func(s string) rune {
		cp, err := strconv.ParseUint(s, 16, 64)
		if err != nil {
			panic(err)
		}
		return rune(cp)
	}

	confusables = make(map[rune]string, 1000)
	scanner := bufio.NewScanner(bytes.NewReader(confusablesSrc))
	for scanner.Scan() {
		// trim whitespace or BOM
		line := strings.TrimPrefix(strings.TrimSpace(scanner.Text()), "\uFEFF")

		// ignore comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		groups := confusablesPattern.FindStringSubmatch(line)
		source := parseHex(groups[1])
		var target []rune
		for _, h := range strings.Fields(groups[2]) {
			target = append(target, parseHex(h))
		}

		confusables[source] = string(target)
	}
}

// Implements https://www.unicode.org/reports/tr39/#def-skeleton
func Skeleton(s string) string {
	var sb strings.Builder

	for _, r := range norm.NFD.String(s) {
		// TODO this is not the complete set of Default_Ignorable_Code_Point
		if unicode.In(r, unicode.Other_Default_Ignorable_Code_Point) {
			continue
		}

		if c, ok := confusables[r]; ok {
			sb.WriteString(c)
		} else {
			sb.WriteRune(r)
		}
	}

	return norm.NFD.String(sb.String())
}

// Implements https://www.unicode.org/reports/tr39/#def-confusable
func Confusable(x, y string) bool {
	return Skeleton(x) == Skeleton(y)
}
