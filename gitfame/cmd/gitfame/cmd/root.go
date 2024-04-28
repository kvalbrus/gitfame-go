package cmd

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

var (
	TABULAR    OutputType = OutputType{name: "tabular"}
	CSV        OutputType = OutputType{name: "csv"}
	JSON       OutputType = OutputType{name: "json"}
	JSON_LINES OutputType = OutputType{name: "json-lines"}
)

func GetOutputType(name string) (OutputType, error) {
	switch name {
	case TABULAR.name:
		return TABULAR, nil

	case CSV.name:
		return CSV, nil

	case JSON.name:
		return JSON, nil

	case JSON_LINES.name:
		return JSON_LINES, nil

	default:
		return TABULAR, errors.New("illegal type")
	}
}

var (
	DEFAULT SortType = SortType{name: "default"}
	LINES   SortType = SortType{name: "lines"}
	COMMITS SortType = SortType{name: "commits"}
	FILES   SortType = SortType{name: "files"}
)

func GetSortType(name string) (SortType, error) {
	switch name {
	case DEFAULT.name:
		return DEFAULT, nil

	case LINES.name:
		return LINES, nil

	case COMMITS.name:
		return COMMITS, nil

	case FILES.name:
		return FILES, nil

	default:
		return DEFAULT, errors.New("illegal type")
	}
}

var (
	repository    string
	revision      string
	format        string
	order_by      string
	use_committer bool
	exclude       []string
	//restrict_to   []string
)

var rootCmd = &cobra.Command{
	Use:   "gitfame",
	Short: "This program prints the statistics of the author of the git repository",
	Long:  "This program prints the statistics of the author of the git repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		//repository = filepath.FromSlash(repository)
		//fmt.Println(repository)
		//repository = strings.Replace(repository, "./", "", 1)

		excludeFlag, err := cmd.Flags().GetString("exclude")
		if err != nil {
			return err
		}

		exclude = strings.Split(excludeFlag, ",")

		files, err := GitFiles(repository, revision)
		if err != nil {
			return err
		}

		formatType, err := GetOutputType(format)
		if err != nil {
			return err
		}

		sortType, err := GetSortType(order_by)
		if err != nil {
			return err
		}
		//restrictToFlag, err := cmd.Flags().GetString("restrict_to")
		//if err != nil {
		//	return err
		//}

		//restrict_to = strings.Split(restrictToFlag, ",")

		authors := make([]Author, 0)
		for _, author := range parse(files) {
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
	rootCmd.Flags().StringVar(&order_by, "order-by", "default", "The method of sorting the results; one of: lines (default), commits, files")
	rootCmd.Flags().BoolVar(&use_committer, "use-committer", false, "A Boolean flag that replaces the author (default) with the committer in the calculations")
	rootCmd.Flags().StringVar(&format, "format", "tabular", "Output format; one of tabular (default), csv, json, json-lines")
	rootCmd.Flags().String("extensions", "", "A list of extensions that narrows down the list of files in the calculation; many restrictions are separated by commas, for example, '.go,.md'")
	rootCmd.Flags().String("languages", "", "A list of languages (programming, markup, etc.), narrowing the list of files in the calculation; many restrictions are separated by commas, for example 'go,markdown'")
	rootCmd.Flags().String("exclude", "", "A set of Glob patterns excluding files from the calculation, for example 'foo/*,bar/*'")
	//rootCmd.Flags().String("restrict-to", "", "A set of Glob patterns that excludes all files that do not satisfy any of the patterns in the set")
}

func GitFiles(repository string, revision string) ([]string, error) {
	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", revision)
	//fmt.Println(revision)
	cmd.Dir = repository
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	files := make([]string, 0)
	scanner := bufio.NewScanner(stdout)

	excludeFilesMap := make(map[string]struct{})
	for _, excl := range exclude {
		path := repository + "/" + excl
		if excludeFiles, err := filepath.Glob(path); err != nil {
			return nil, err
		} else {
			for _, excludeFile := range excludeFiles {
				excludeFilesMap[excludeFile] = struct{}{}
			}
		}
	}

	for scanner.Scan() {
		file := scanner.Text()
		if _, ok := excludeFilesMap[repository+"/"+file]; !ok {
			files = append(files, file)
		}
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

func parse(files []string) map[string]Author {

	commits := make(map[string]Commit)
	for _, file := range files {
		//fmt.Println(file)
		cmd := exec.Command("git", "blame", "-p", file)
		cmd.Dir = repository

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil
		}

		if err := cmd.Start(); err != nil {
			return nil
		}

		scanner := bufio.NewScanner(stdout)
		lines := make([]string, 0)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		//fmt.Println(lines)

		commitHash := ""
		for i, line := range lines {
			//fmt.Println(line)

			if strings.HasPrefix(line, "author ") && i != 0 {
				previousLine := lines[i-1]
				args := strings.Split(previousLine, " ")
				if len(args) != 4 {
					// todo: error
				}

				author := strings.ReplaceAll(line, "author ", "")
				hash := args[0]
				commitHash = args[0]

				countGroup, err := strconv.Atoi(args[3])
				if err != nil {
					// todo: error
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
					// todo: error
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
			} else if use_committer && strings.HasPrefix(line, "committer ") {
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

	return authors
}

func Sort(authors []Author, sortType SortType) []Author {
	switch sortType {
	case DEFAULT:
		sort.Slice(authors, func(i, j int) bool {
			authorI, authorJ := authors[i], authors[j]

			if authorI.lines == authorJ.lines {
				if authorI.commits == authorJ.commits {
					if len(authorI.files) == len(authorJ.files) {
						return strings.Compare(authorI.name, authorJ.name) < 0
					}

					return len(authorI.files) > len(authorJ.files)
				}

				return authorI.commits > authorJ.commits
			}

			return authorI.lines > authorJ.lines
		})

	case COMMITS:
		sort.Slice(authors, func(i, j int) bool {
			if authors[i].commits == authors[j].commits {
				return strings.Compare(authors[i].name, authors[j].name) < 0
			}

			return authors[i].commits > authors[j].commits
		})

	case FILES:
		sort.Slice(authors, func(i, j int) bool {
			if len(authors[i].files) == len(authors[j].files) {
				return strings.Compare(authors[i].name, authors[j].name) < 0
			}

			return len(authors[i].files) > len(authors[j].files)
		})

	case LINES:
		sort.Slice(authors, func(i, j int) bool {
			if authors[i].lines == authors[j].lines {
				return strings.Compare(authors[i].name, authors[j].name) < 0
			}

			return authors[i].lines > authors[j].lines
		})
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
		return GetJson(authors)
	case JSON_LINES:
		return GetJsonLines(authors)
	}

	return ""
}

func GetTabular(authors []Author) string {
	buffer := new(bytes.Buffer)
	writer := tabwriter.NewWriter(buffer, 0, 0, 1, ' ', tabwriter.DiscardEmptyColumns)

	fmt.Fprintf(writer, fmt.Sprintf("Name\tLines\tCommits\tFiles\n"))
	for _, author := range authors {
		fmt.Fprintf(writer, fmt.Sprintf("%s\t%d\t%d\t%d\n", author.name, author.lines, author.commits, len(author.files)))
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

type JsonFormat struct {
	name    string
	lines   int
	commits int
	files   int
}

func GetJsonLines(authors []Author) string {
	builder := strings.Builder{}
	for _, author := range authors {
		line := &JsonFormat{
			name:    author.name,
			lines:   author.lines,
			commits: author.commits,
			files:   len(author.files),
		}

		jsonMap, _ := json.Marshal(line)
		builder.WriteString(string(jsonMap))
		builder.WriteString("\n")
	}

	return builder.String()
}

func GetJson(authors []Author) string {
	builder := strings.Builder{}
	builder.WriteString("[")
	for i, author := range authors {
		line := &JsonFormat{
			name:    author.name,
			lines:   author.lines,
			commits: author.commits,
			files:   len(author.files),
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
