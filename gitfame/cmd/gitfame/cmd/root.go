package cmd

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"os/exec"
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

var (
	DEFAULT SortType = SortType{name: "default"}
	LINES   SortType = SortType{name: "lines"}
	COMMITS SortType = SortType{name: "commits"}
	FILES   SortType = SortType{name: "files"}
)

var (
	repository string
)

var rootCmd = &cobra.Command{
	Use:   "gitfame",
	Short: "This program prints the statistics of the author of the git repository",
	Long:  "This program prints the statistics of the author of the git repository",
	Run: func(cmd *cobra.Command, args []string) {

		//fmt.Println()

		flagFormat, err := cmd.Flags().GetString("format")
		if err != nil {
			return
		}

		var format OutputType
		switch flagFormat {
		case "tabular":
			format = TABULAR

		case "csv":
			format = CSV

		case "json":
			format = JSON

		case "json-lines":
			format = JSON_LINES

		default:
			return
		}

		files, err := GitFiles(repository)
		if err != nil {
			fmt.Println(err.Error())
			panic(err)
		}

		authors := make([]Author, 0)
		for _, author := range parse(files) {
			authors = append(authors, author)
		}

		fmt.Println(GetOutput(Sort(authors, DEFAULT), format))
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&repository, "repository", "./", "The path to the Git repository")
	rootCmd.PersistentFlags().String("revision", "HEAD", "A pointer to a commit")
	rootCmd.PersistentFlags().String("order-by", "lines", "The method of sorting the results; one of: lines (default), commits, files")
	rootCmd.PersistentFlags().Bool("use-committer", true, "A Boolean flag that replaces the author (default) with the committer in the calculations")
	rootCmd.PersistentFlags().String("format", "tabular", "Output format; one of tabular (default), csv, json, json-lines")
	rootCmd.PersistentFlags().String("extensions", "", "A list of extensions that narrows down the list of files in the calculation; many restrictions are separated by commas, for example, '.go,.md'")
	rootCmd.PersistentFlags().String("languages", "", "A list of languages (programming, markup, etc.), narrowing the list of files in the calculation; many restrictions are separated by commas, for example 'go,markdown'")
	rootCmd.PersistentFlags().String("exclude", "", "A set of Glob patterns excluding files from the calculation, for example 'foo/*,bar/*'")
	rootCmd.PersistentFlags().String("restrict-to", "", "A set of Glob patterns that excludes all files that do not satisfy any of the patterns in the set")
}

func GitFiles(repository string) ([]string, error) {
	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", "HEAD")
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
	for scanner.Scan() {
		files = append(files, scanner.Text())
	}

	return files, nil
}

type OutputType struct {
	name string
}

type SortType struct {
	name string
}

//type Change struct {
//	countGroup int
//}

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

		//hash := ""
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
				//fmt.Println(hash)

				//originalNumberLine, err := strconv.Atoi(args[1])
				//if err != nil {
				//	// todo: error
				//}
				//
				//finalNumberLine, err := strconv.Atoi(args[2])
				//if err != nil {
				//	// todo: error
				//}

				countGroup, err := strconv.Atoi(args[3])
				if err != nil {
					// todo: error
				}

				//change := Change{
				//	hash:       args[0],
				//	countGroup: countGroup,
				//}

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
			}
		}

		//fmt.Println("\n\n")
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

			for file, _ := range commit.files {
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
	//return ""
	buffer := new(bytes.Buffer)
	writer := tabwriter.NewWriter(buffer, 0, 0, 1, ' ', tabwriter.DiscardEmptyColumns)
	//writer.Write(make([]string, 0))

	//builder.WriteString("Name         Lines Commits Files")
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

func GetJsonLines(authors []Author) string {
	builder := strings.Builder{}
	for _, author := range authors {
		lineMap := map[string]string{
			"name":    author.name,
			"lines":   strconv.Itoa(author.lines),
			"commits": strconv.Itoa(author.commits),
			"files":   strconv.Itoa(len(author.files)),
		}

		jsonMap, _ := json.Marshal(lineMap)
		builder.WriteString(string(jsonMap))
		builder.WriteString("\n")
	}

	return builder.String()
}

func GetJson(authors []Author) string {
	builder := strings.Builder{}
	builder.WriteString("[")
	for i, author := range authors {
		lineMap := map[string]string{
			"name":    author.name,
			"lines":   strconv.Itoa(author.lines),
			"commits": strconv.Itoa(author.commits),
			"files":   strconv.Itoa(len(author.files)),
		}

		jsonMap, _ := json.Marshal(lineMap)
		builder.WriteString(string(jsonMap))

		if i != len(authors)-1 {
			builder.WriteString(",")
		}
	}

	builder.WriteString("]")

	return builder.String()
}
