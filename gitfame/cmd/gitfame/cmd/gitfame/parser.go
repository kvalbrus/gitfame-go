package gitfame

import (
	"gitlab.com/slon/shad-go/gitfame/cmd/gitfame/config"
	"regexp"
	"strings"
)

type FileParser struct {
	IncludeRules  map[PatternRule]struct{}
	ExcludeRules  map[PatternRule]struct{}
	LanguageRules map[PatternRule]struct{}
}

func NewParser(exclude []string, extensions []string, restrictTo []string, language []string) FileParser {
	includeRules := make(map[PatternRule]struct{})
	excludeRules := make(map[PatternRule]struct{})
	languageRules := make(map[PatternRule]struct{})
	for _, e := range exclude {
		excludeRules[&MatchPatternRule{pattern: e}] = struct{}{}
	}

	for _, e := range extensions {
		includeRules[&MatchPatternRule{pattern: e}] = struct{}{}
	}

	for _, r := range restrictTo {
		includeRules[&MatchPatternRule{pattern: r}] = struct{}{}
	}

	languages, _ := config.GetLanguages(language)
	for _, l := range languages {
		for _, lang := range l.Extensions {
			languageRules[&SuffixPatternRule{pattern: lang}] = struct{}{}
		}
	}

	return FileParser{IncludeRules: includeRules, ExcludeRules: excludeRules, LanguageRules: languageRules}
}

type PatternRule interface {
	IsAgree(string) (bool, error)
}

type MatchPatternRule struct {
	pattern string
}

type SuffixPatternRule struct {
	pattern string
}

func (rule *MatchPatternRule) IsAgree(file string) (bool, error) {
	if matched, err := regexp.MatchString(rule.pattern, file); err != nil {
		if matched, err = regexp.MatchString(regexp.QuoteMeta(rule.pattern), file); err != nil {
			return false, err
		} else {
			return matched, nil
		}
	} else {
		return matched, nil
	}

}

func (rule *SuffixPatternRule) IsAgree(file string) (bool, error) {
	return strings.HasSuffix(file, rule.pattern), nil
}

func (parser *FileParser) Parse(files []string) ([]string, error) {
	processFiles := make([]string, 0)

	for _, file := range files {
		flag := true
		for rule := range parser.ExcludeRules {
			if matched, err := rule.IsAgree(file); err != nil {
				return nil, err
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
	if len(parser.LanguageRules) != 0 {
		for _, file := range files {
			for rule := range parser.LanguageRules {
				if matched, err := rule.IsAgree(file); err != nil {
					return nil, err
				} else if matched {
					processFiles = append(processFiles, file)
					break
				}
			}
		}

		files = processFiles
		processFiles = make([]string, 0)
	}

	// include
	if len(parser.IncludeRules) != 0 {
		for _, file := range files {
			for rule := range parser.IncludeRules {
				if matched, err := rule.IsAgree(file); err != nil {
					return nil, err
				} else if matched {
					processFiles = append(processFiles, file)
					break
				}
			}
		}
	} else {
		return files, nil
	}

	return processFiles, nil
}
