package main

import (
	"regexp"
	"strings"
)

// Meses en español
var mesesEs = map[int]string{
	1: "ene", 2: "feb", 3: "mar", 4: "abr",
	5: "may", 6: "jun", 7: "jul", 8: "ago",
	9: "sep", 10: "oct", 11: "nov", 12: "dic",
}

// Estructura de directorios generada por el script
var subfolders = []string{"Flats", "Lights", "Lights/Rejected", "Logs", "PixInsight", "Final"}

// formatTargetName handles catalogs M, NGC, IC and capitalizes properly
func formatTargetName(name string) string {
	catalogs := []string{"M", "NGC", "IC"}
	reSpaces := regexp.MustCompile(`[-_.]+`)
	cleanName := reSpaces.ReplaceAllString(name, " ")

	words := strings.Fields(cleanName)
	var formattedWords []string

	reNumMatch := regexp.MustCompile(`^(` + strings.Join(catalogs, "|") + `)(\d+)$`)

	for _, w := range words {
		wUp := strings.ToUpper(w)

		match := reNumMatch.FindStringSubmatch(wUp)
		if match != nil {
			cat := match[1]
			num := match[2]
			if cat == "M" {
				formattedWords = append(formattedWords, cat+num)
			} else {
				formattedWords = append(formattedWords, cat+"_"+num)
			}
			continue
		}

		isCatalog := false
		for _, cat := range catalogs {
			if wUp == cat {
				isCatalog = true
				break
			}
		}
		if isCatalog || isDigit(wUp) {
			formattedWords = append(formattedWords, wUp)
			continue
		}

		// Capitalize
		if len(w) > 0 {
			formattedWords = append(formattedWords, strings.ToUpper(w[:1])+strings.ToLower(w[1:]))
		}
	}

	return strings.Join(formattedWords, "_")
}

func isDigit(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func normalizeName(name string) string {
	re := regexp.MustCompile(`[\W_]+`)
	return strings.ToLower(re.ReplaceAllString(name, ""))
}
