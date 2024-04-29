package gitfame

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/slon/shad-go/gitfame/cmd/gitfame/config"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

var (
	TABULAR   OutputType = OutputType{name: "tabular"}
	CSV       OutputType = OutputType{name: "csv"}
	JSON      OutputType = OutputType{name: "json"}
	JSONLINES OutputType = OutputType{name: "json-lines"}
)

func GetOutputType(name string) (OutputType, error) {
	switch name {
	case TABULAR.name:
		return TABULAR, nil

	case CSV.name:
		return CSV, nil

	case JSON.name:
		return JSON, nil

	case JSONLINES.name:
		return JSONLINES, nil

	default:
		return TABULAR, errors.New("illegal type")
	}
}

var (
	//DEFAULT SortType = SortType{name: "default"}
	LINES   SortType = SortType{name: "lines"}
	COMMITS SortType = SortType{name: "commits"}
	FILES   SortType = SortType{name: "files"}
)

func GetSortType(name string) (SortType, error) {
	switch name {
	//case DEFAULT.name:
	//	return DEFAULT, nil

	case LINES.name:
		return LINES, nil

	case COMMITS.name:
		return COMMITS, nil

	case FILES.name:
		return FILES, nil

	default:
		return LINES, errors.New("illegal type")
	}
}

var (
	repository   string
	revision     string
	format       string
	orderBy      string
	useCommitter bool
	exclude      []string
	extensions   []string
	restrictTo   []string
	languages    []config.Lang
)

var rootCmd = &cobra.Command{
	Use:   "gitfame",
	Short: "This program prints the statistics of the author of the git repository",
	Long:  "This program prints the statistics of the author of the git repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		extensionsFlag, err := cmd.Flags().GetString("extensions")
		if err != nil {
			return err
		}

		if extensionsFlag != "" {
			extensions = strings.Split(extensionsFlag, ",")
		}

		languagesFlag, err := cmd.Flags().GetString("languages")
		if err != nil {
			return err
		}

		if languagesFlag != "" {
			var languageExtensions []config.Lang
			languageExtensions, err = config.LanguageExtensions()
			if err != nil {
				return err
			}

			for _, lang := range strings.Split(languagesFlag, ",") {
				for _, langExt := range languageExtensions {
					if strings.EqualFold(langExt.Name, lang) {
						languages = append(languages, langExt)
					}
				}
			}
		}

		excludeFlag, err := cmd.Flags().GetString("exclude")
		if err != nil {
			return err
		}

		if excludeFlag != "" {
			exclude = strings.Split(excludeFlag, ",")
		}

		restrictToFlag, err := cmd.Flags().GetString("restrict-to")
		if err != nil {
			return err
		}

		if restrictToFlag != "" {
			restrictTo = strings.Split(restrictToFlag, ",")
		}

		files, err := GitFiles(repository, revision)
		if err != nil {
			return err
		}

		formatType, err := GetOutputType(format)
		if err != nil {
			return err
		}

		sortType, err := GetSortType(orderBy)
		if err != nil {
			return err
		}
		//restrictToFlag, err := cmd.Flags().GetString("restrictTo")
		//if err != nil {
		//	return err
		//}

		//restrictTo = strings.Split(restrictToFlag, ",")

		authors := make([]Author, 0)

		parseAuthors, err := parse(files)
		if err != nil {
			return err
		}

		for _, author := range parseAuthors {
			authors = append(authors, author)
		}

		fmt.Println(GetOutput(Sort(authors, sortType), formatType))

		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().StringVar(&repository, "repository", "./", "The path to the Git repository")
	rootCmd.Flags().StringVar(&revision, "revision", "HEAD", "A pointer to a commit")
	rootCmd.Flags().StringVar(&orderBy, "order-by", "lines", "The method of sorting the results; one of: lines (default), commits, files")
	rootCmd.Flags().BoolVar(&useCommitter, "use-committer", false, "A Boolean flag that replaces the author (default) with the committer in the calculations")
	rootCmd.Flags().StringVar(&format, "format", "tabular", "Output format; one of tabular (default), csv, json, json-lines")
	rootCmd.Flags().String("extensions", "", "A list of extensions that narrows down the list of files in the calculation; many restrictions are separated by commas, for example, '.go,.md'")
	rootCmd.Flags().String("languages", "", "A list of languages (programming, markup, etc.), narrowing the list of files in the calculation; many restrictions are separated by commas, for example 'go,markdown'")
	rootCmd.Flags().String("exclude", "", "A set of Glob patterns excluding files from the calculation, for example 'foo/*,bar/*'")
	rootCmd.Flags().String("restrict-to", "", "A set of Glob patterns that excludes all files that do not satisfy any of the patterns in the set")
}

func GitFiles(repository string, revision string) ([]string, error) {
	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", revision)
	//fmt.Println(revision)
	cmd.Dir = repository

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err = cmd.Start(); err != nil {
		return nil, err
	}

	files := make([]string, 0)
	scanner := bufio.NewScanner(stdout)

	excPatterns := make(map[string]struct{})
	inclPatterns := make(map[string]struct{})
	langPatterns := make(map[string]struct{})

	for _, exc := range exclude {
		excPatterns[exc] = struct{}{}
	}

	for _, language := range languages {
		for _, incl := range language.Extensions {
			langPatterns[incl] = struct{}{}
		}
	}

	for _, incl := range extensions {
		langPatterns[incl] = struct{}{}
	}

	for _, incl := range restrictTo {
		inclPatterns[incl] = struct{}{}
	}

	patternRules := &PatternRules{
		IncludePatterns:   inclPatterns,
		ExcludePatterns:   excPatterns,
		LanguagesPatterns: langPatterns,
	}

	for scanner.Scan() {
		files = append(files, scanner.Text())
	}

	if err = scanner.Err(); err != nil {
		return nil, err
	}

	if err = cmd.Wait(); err != nil {
		return nil, err
	}

	files, err = patternRules.Sweep(files)
	if err != nil {
		return nil, err
	}

	return files, nil
}

type OutputType struct {
	name string
}

type SortType struct {
	name string
}

type Author struct {
	name    string
	lines   int
	commits int
	files   map[string]struct{}
}

type Commit struct {
	hash   string
	author string
	files  map[string]struct{}
	lines  int
	//change Change
}

func parse(files []string) (map[string]Author, error) {

	commits := make(map[string]Commit)
	for _, file := range files {
		//fmt.Println(file)
		cmd := exec.Command("git", "blame", "-p", revision, "--", file)
		cmd.Dir = repository

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, err
		}

		if err = cmd.Start(); err != nil {
			return nil, err
		}

		scanner := bufio.NewScanner(stdout)
		lines := make([]string, 0)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		if len(lines) == 0 {
			cmd := exec.Command("git", "log", "-1", "--pretty=format:%H\n%an", revision, "--", file)
			cmd.Dir = repository

			cmd.Dir = repository

			stdout, err = cmd.StdoutPipe()
			if err != nil {
				return nil, err
			}

			if err = cmd.Start(); err != nil {
				return nil, err
			}

			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}

			//line := strings.Split(lines[0], ";")

			hash, author := lines[0], lines[1]

			if commit, ok := commits[hash]; ok {
				commit.files[file] = struct{}{}

				commits[hash] = commit
			} else {
				files := make(map[string]struct{})
				files[file] = struct{}{}

				commits[hash] = Commit{
					hash:   hash,
					author: author,
					lines:  0,
					files:  files,
				}
			}
		}

		commitHash := ""
		for i, line := range lines {
			//fmt.Println(line)
			if strings.HasPrefix(line, "author ") && i != 0 {
				previousLine := lines[i-1]
				args := strings.Split(previousLine, " ")
				if len(args) != 4 {
					return nil, err
				}

				author := strings.ReplaceAll(line, "author ", "")
				hash := args[0]
				commitHash = args[0]

				countGroup, err := strconv.Atoi(args[3])
				if err != nil {
					return nil, err
				}

				if commit, ok := commits[hash]; ok {
					commit.lines += countGroup
					commit.files[file] = struct{}{}

					commits[hash] = commit
				} else {
					files := make(map[string]struct{})
					files[file] = struct{}{}

					commits[hash] = Commit{
						hash:   hash,
						author: author,
						lines:  countGroup,
						files:  files,
					}
				}
			} else if strings.HasPrefix(line, "\t") && i != len(lines)-1 {
				nextLine := lines[i+1]
				args := strings.Split(nextLine, " ")

				if len(args) != 4 {
					continue
				}

				if i != len(lines)-2 && strings.HasPrefix(lines[i+2], "author ") {
					continue
				}

				hash := args[0]

				countGroup, err := strconv.Atoi(args[3])
				if err != nil {
					return nil, err
				}

				if commit, ok := commits[hash]; ok {
					commit.lines += countGroup
					commit.files[file] = struct{}{}

					commits[hash] = commit
				} else {
					files := make(map[string]struct{})
					files[file] = struct{}{}

					commits[hash] = Commit{
						hash:   hash,
						author: "",
						lines:  countGroup,
						files:  files,
					}
				}
			} else if useCommitter && strings.HasPrefix(line, "committer ") {
				commit := commits[commitHash]
				commit.author = strings.ReplaceAll(line, "committer ", "")
				commits[commitHash] = commit
			}
		}
	}

	authors := make(map[string]Author)
	for _, commit := range commits {
		if _, ok := authors[commit.author]; !ok {
			authors[commit.author] = Author{
				name:    commit.author,
				lines:   commit.lines,
				commits: 1,
				files:   commit.files,
			}
		} else {
			author := authors[commit.author]

			for file := range commit.files {
				author.files[file] = struct{}{}
			}

			if author.name == "" {
				author.name = commit.author
			}

			author.lines += commit.lines
			author.commits++

			authors[commit.author] = author
		}
	}

	return authors, nil
}

func Sort1(authors []Author, sort1 SortType, sort2 SortType, sort3 SortType) []Author {
	sort.Slice(authors, func(i, j int) bool {
		var a, b int
		switch sort1 {
		case LINES:
			a, b = authors[i].lines, authors[j].lines

		case COMMITS:
			a, b = authors[i].commits, authors[j].commits

		case FILES:
			a, b = len(authors[i].files), len(authors[j].files)
		}

		if a == b {
			switch sort2 {
			case LINES:
				a, b = authors[i].lines, authors[j].lines

			case COMMITS:
				a, b = authors[i].commits, authors[j].commits

			case FILES:
				a, b = len(authors[i].files), len(authors[j].files)
			}

			if a == b {
				switch sort3 {
				case LINES:
					a, b = authors[i].lines, authors[j].lines

				case COMMITS:
					a, b = authors[i].commits, authors[j].commits

				case FILES:
					a, b = len(authors[i].files), len(authors[j].files)
				}

				if a == b {
					return strings.Compare(authors[i].name, authors[j].name) < 0
				} else {
					return a > b
				}
			} else {
				return a > b
			}

		} else {
			return a > b
		}
	})

	return authors
}

func Sort(authors []Author, sortType SortType) []Author {
	switch sortType {
	case COMMITS:
		return Sort1(authors, COMMITS, LINES, FILES)
	case LINES:
		return Sort1(authors, LINES, COMMITS, FILES)
	case FILES:
		return Sort1(authors, FILES, LINES, COMMITS)
	}

	return authors
}

func GetOutput(authors []Author, outputType OutputType) string {
	switch outputType {
	case TABULAR:
		return GetTabular(authors)
	case CSV:
		return GetCSV(authors)
	case JSON:
		return GetJSON(authors)
	case JSONLINES:
		return GetJSONLines(authors)
	}

	return ""
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

type JSONFormat struct {
	Name    string `json:"name"`
	Lines   int    `json:"lines"`
	Commits int    `json:"commits"`
	Files   int    `json:"files"`
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
