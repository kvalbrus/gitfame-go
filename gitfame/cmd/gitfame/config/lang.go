package config

import (
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed language_extensions.json
var data []byte

type Lang struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Extensions []string `json:"extensions"`
}

func GetLanguages(str []string) ([]Lang, error) {
	var languageExtensions []Lang
	if err := json.Unmarshal(data, &languageExtensions); err != nil {
		return nil, err
	}

	languages := make([]Lang, 0)
	for _, langName := range str {
		for _, extensions := range languageExtensions {
			if strings.EqualFold(langName, extensions.Name) {
				languages = append(languages, extensions)
				break
			}
		}
	}

	return languages, nil
}
