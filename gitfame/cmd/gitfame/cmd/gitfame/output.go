package gitfame

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"
)

var (
	TABULAR   OutputType = OutputType{Name: "tabular", GetOutput: GetTabular}
	CSV       OutputType = OutputType{Name: "csv", GetOutput: GetCSV}
	JSON      OutputType = OutputType{Name: "json", GetOutput: GetJSON}
	JSONLINES OutputType = OutputType{Name: "json-lines", GetOutput: GetJSONLines}
)

type OutputParser struct {
	Type OutputType
}

type OutputType struct {
	Name      string
	GetOutput func(authors []Author) string
}

type JSONFormat struct {
	Name    string `json:"name"`
	Lines   int    `json:"lines"`
	Commits int    `json:"commits"`
	Files   int    `json:"files"`
}

func (parser *OutputParser) GetOutput(authors []Author) string {
	return parser.Type.GetOutput(authors)
}

func GetOutputType(name string) (OutputType, error) {
	switch name {
	case TABULAR.Name:
		return TABULAR, nil

	case CSV.Name:
		return CSV, nil

	case JSON.Name:
		return JSON, nil

	case JSONLINES.Name:
		return JSONLINES, nil

	default:
		return TABULAR, errors.New("illegal type")
	}
}

func GetTabular(authors []Author) string {
	buffer := new(bytes.Buffer)
	writer := tabwriter.NewWriter(buffer, 0, 0, 1, ' ', tabwriter.DiscardEmptyColumns)

	fmt.Fprintf(writer, "Name\tLines\tCommits\tFiles\n")
	for i, author := range authors {
		fmt.Fprintf(writer, "%s\t%d\t%d\t%d", author.name, author.lines, author.commits, len(author.files))

		if i != len(authors)-1 {
			fmt.Fprintf(writer, "\n")
		}
	}

	writer.Flush()

	return buffer.String()
}

func GetCSV(authors []Author) string {
	buffer := new(bytes.Buffer)
	writer := csv.NewWriter(buffer)
	err := writer.Write([]string{"Name", "Lines", "Commits", "Files"})
	if err != nil {
		return ""
	}

	for _, author := range authors {
		err := writer.Write([]string{author.name, strconv.Itoa(author.lines), strconv.Itoa(author.commits), strconv.Itoa(len(author.files))})
		if err != nil {
			return ""
		}
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return ""
	}

	return strings.TrimSuffix(buffer.String(), "\n")
}

func GetJSONLines(authors []Author) string {
	builder := strings.Builder{}
	for _, author := range authors {
		line := &JSONFormat{
			Name:    author.name,
			Lines:   author.lines,
			Commits: author.commits,
			Files:   len(author.files),
		}

		jsonMap, _ := json.Marshal(line)
		builder.WriteString(string(jsonMap))
		builder.WriteString("\n")
	}

	return builder.String()
}

func GetJSON(authors []Author) string {
	builder := strings.Builder{}
	builder.WriteString("[")
	for i, author := range authors {
		line := &JSONFormat{
			Name:    author.name,
			Lines:   author.lines,
			Commits: author.commits,
			Files:   len(author.files),
		}

		jsonMap, _ := json.Marshal(line)
		builder.WriteString(string(jsonMap))

		if i != len(authors)-1 {
			builder.WriteString(",")
		}
	}

	builder.WriteString("]")

	return builder.String()
}
