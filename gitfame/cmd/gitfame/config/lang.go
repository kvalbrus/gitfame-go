package config

import (
	_ "embed"
	"encoding/json"
)

//go:embed language_extensions.json
var data []byte

type Lang struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Extensions []string `json:"extensions"`
}

func LanguageExtensions() ([]Lang, error) {
	var languageExtensions []Lang
	if err := json.Unmarshal(data, &languageExtensions); err != nil {
		return nil, err
	}

	return languageExtensions, nil
}
