package gitfame

import (
	"fmt"
	"github.com/spf13/cobra"
	"strconv"
	"strings"
)

var (
	repository   string
	revision     string
	format       string
	orderBy      string
	useCommitter bool
	exclude      []string
	extensions   []string
	restrictTo   []string
	languages    []string
)

var rootCmd = &cobra.Command{
	Use:   "gitfame",
	Short: "This program prints the statistics of the author of the git repository",
	Long:  "This program prints the statistics of the author of the git repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		formatType, err := GetOutputType(format)
		if err != nil {
			return err
		}

		sortType, err := GetSortType(orderBy)
		if err != nil {
			return err
		}

		parser := NewParser(exclude, extensions, restrictTo, languages)
		git := NewGitConfig(repository, revision)

		files, err := git.Files(parser)
		if err != nil {
			return err
		}

		authors := make([]Author, 0)

		parseAuthors, err := parse(files)
		if err != nil {
			return err
		}

		for _, author := range parseAuthors {
			authors = append(authors, author)
		}

		outParser := &OutputParser{Type: formatType}
		fmt.Println(outParser.GetOutput(Sort(authors, sortType)))

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
	rootCmd.Flags().StringSliceVar(&extensions, "extensions", []string{}, "A list of extensions that narrows down the list of files in the calculation; many restrictions are separated by commas, for example, '.go,.md'")
	rootCmd.Flags().StringSliceVar(&languages, "languages", []string{}, "A list of languages (programming, markup, etc.), narrowing the list of files in the calculation; many restrictions are separated by commas, for example 'go,markdown'")
	rootCmd.Flags().StringSliceVar(&exclude, "exclude", []string{}, "A set of Glob patterns excluding files from the calculation, for example 'foo/*,bar/*'")
	rootCmd.Flags().StringSliceVar(&restrictTo, "restrict-to", []string{}, "A set of Glob patterns that excludes all files that do not satisfy any of the patterns in the set")
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
}

func parse(files []string) (map[string]Author, error) {
	git := &GitConf{
		Repository: repository,
		Revision:   revision,
	}

	commits := make(map[string]Commit)
	for _, file := range files {
		lines, err := git.Blame(file)
		if err != nil {
			return nil, err
		}

		if len(lines) == 0 {
			lines, err = git.Log(file)
			if err != nil {
				return nil, err
			}

			if len(lines) != 2 {
				continue
			}

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
