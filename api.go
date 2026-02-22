package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const sesameAPIURL = "http://cdsweb.u-strasbg.fr/cgi-bin/nph-sesame/-oI/A?"

// querySesame busca el objeto astronómico en el API de CDS Sesame
func querySesame(searchInput string) (commonName string, technicalOptions []string) {
	targetClean := strings.TrimSpace(searchInput)
	apiURL := sesameAPIURL + url.QueryEscape(targetClean)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", nil
	}
	req.Header.Add("User-Agent", "astroquery/0.4.6 (Go-Astro-Session/1.1)")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil
	}

	lines := strings.Split(string(bodyBytes), "\n")
	found := false
	otype := "Objeto astronómico"
	var allCommonNames []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "%C.0 ") {
			otype = strings.TrimPrefix(line, "%C.0 ")
		}

		if strings.HasPrefix(line, "%I ") || strings.HasPrefix(line, "%I.0 ") {
			found = true
			val := strings.TrimPrefix(line, "%I.0 ")
			val = strings.TrimPrefix(val, "%I ")
			valTrim := strings.TrimSpace(val)

			if strings.HasPrefix(valTrim, "NAME ") {
				possibleName := strings.TrimPrefix(valTrim, "NAME ")
				// Ignorar si el "NAME" es en realidad una designación técnica disfrazada (ej. "M 81*")
				isTech := regexp.MustCompile(`^(?i)(M|NGC|IC)\s*\d+\*?$`).MatchString(possibleName)
				if !isTech && !strings.Contains(possibleName, "*") {
					allCommonNames = append(allCommonNames, possibleName)
				}
			}

			for _, cat := range []string{"M ", "NGC ", "IC "} {
				if strings.HasPrefix(valTrim, cat) {
					reMatch := regexp.MustCompile(`^([A-Z]+)\s*(\d+)`)
					match := reMatch.FindStringSubmatch(valTrim)
					if match != nil {
						catStr := strings.ToUpper(match[1])
						numStr := match[2]
						cleanVal := ""
						if catStr == "M" {
							cleanVal = fmt.Sprintf("%s%s", catStr, numStr)
						} else {
							cleanVal = fmt.Sprintf("%s_%s", catStr, numStr)
						}

						exists := false
						for _, existOpt := range technicalOptions {
							if existOpt == cleanVal {
								exists = true
								break
							}
						}
						if !exists {
							technicalOptions = append(technicalOptions, cleanVal)
						}
					}
					break
				}
			}
		}
	}

	commonName = selectBestCommonName(allCommonNames)

	if found {
		fmt.Printf("-> ¡Objeto encontrado! Tipo: %s\n", otype)
		if commonName != "" {
			fmt.Printf("-> Nombre común mapeado: %s\n", commonName)
		}
	}

	return commonName, technicalOptions
}

func selectBestCommonName(names []string) string {
	if len(names) == 0 {
		return ""
	}
	if len(names) == 1 {
		return names[0]
	}

	// Priorizar nombres que tengan palabras clave de objetos astronómicos reales o nombres muy populares.
	keywords := []string{"galaxy", "nebula", "cluster", "group", "object", "bode", "cigar", "andromeda", "orion", "pleiades", "rosette"}
	for _, name := range names {
		lowerName := strings.ToLower(name)
		for _, kw := range keywords {
			if strings.Contains(lowerName, kw) {
				return name
			}
		}
	}

	// Si no hay palabras clave obvias, asume que el nombre más largo es la descripción más "humana" vs acrónimos cortos como UMa A
	best := names[0]
	for _, name := range names {
		if len(name) > len(best) {
			best = name
		}
	}
	return best
}
