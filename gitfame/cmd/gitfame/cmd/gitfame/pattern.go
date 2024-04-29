package gitfame

import (
	"regexp"
	"strings"
)

type PatternRules struct {
	LanguagesPatterns map[string]struct{}
	IncludePatterns   map[string]struct{}
	ExcludePatterns   map[string]struct{}
}

func (rules *PatternRules) Sweep(files []string) ([]string, error) {

	// exclude
	processFiles := make([]string, 0)

	for _, file := range files {
		flag := true
		for exc := range rules.ExcludePatterns {
			if matched, err := regexp.MatchString(exc, file); err != nil {
				exc = regexp.QuoteMeta(exc)
				matched, err = regexp.MatchString(exc, file)
				if matched {
					flag = false
					break
				}
			} else if matched {
				flag = false
				break
			}
		}

		if flag {
			processFiles = append(processFiles, file)
		}
	}

	files = processFiles
	processFiles = make([]string, 0)

	// languages
	if len(rules.LanguagesPatterns) != 0 {
		for _, file := range files {
			flag := false
			for incl := range rules.LanguagesPatterns {
				if strings.HasSuffix(file, incl) {
					flag = true
					break
				}
			}

			if flag {
				processFiles = append(processFiles, file)
			}
		}

		files = processFiles
		processFiles = make([]string, 0)

	}

	// include
	if len(rules.IncludePatterns) != 0 {
		for _, file := range files {
			flag := false
			for incl := range rules.IncludePatterns {
				if matched, err := regexp.MatchString(incl, file); err != nil {
					incl = regexp.QuoteMeta(incl)
					matched, err = regexp.MatchString(incl, file)
					if matched {
						flag = true
						break
					}
				} else if matched {
					flag = true
					break
				}
			}

			if flag {
				processFiles = append(processFiles, file)
			}
		}
	} else {
		return files, nil
	}

	return processFiles, nil
}
