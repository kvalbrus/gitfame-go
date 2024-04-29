package gitfame

import (
	"fmt"
	"github.com/spf13/cobra"
)

var (
	flagRepository   string
	flagRevision     string
	flagFormat       string
	flagOrderBy      string
	flagUseCommitter bool
	flagExclude      []string
	flagExtensions   []string
	flagRestrictTo   []string
	flagLanguages    []string
)

var rootCmd = &cobra.Command{
	Use:   "gitfame",
	Short: "This program prints the statistics of the author of the git flagRepository",
	Long:  "This program prints the statistics of the author of the git flagRepository",
	RunE: func(cmd *cobra.Command, args []string) error {
		formatType, err := GetOutputType(flagFormat)
		if err != nil {
			return err
		}

		sortType, err := GetSortType(flagOrderBy)
		if err != nil {
			return err
		}

		parser := NewParser(flagExclude, flagExtensions, flagRestrictTo, flagLanguages)
		git := NewGitConfig(flagRepository, flagRevision)

		files, err := git.Files(parser)
		if err != nil {
			return err
		}

		authors := make([]Author, 0)

		gitfameParser := NewGitfameParser(git, flagUseCommitter)
		parseAuthors, err := gitfameParser.parse(files)
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
	rootCmd.Flags().StringVar(&flagRepository, "repository", "./", "The path to the Git flagRepository")
	rootCmd.Flags().StringVar(&flagRevision, "revision", "HEAD", "A pointer to a commit")
	rootCmd.Flags().StringVar(&flagOrderBy, "order-by", "lines", "The method of sorting the results; one of: lines (default), commits, files")
	rootCmd.Flags().BoolVar(&flagUseCommitter, "use-committer", false, "A Boolean flag that replaces the author (default) with the committer in the calculations")
	rootCmd.Flags().StringVar(&flagFormat, "format", "tabular", "Output format; one of tabular (default), csv, json, json-lines")
	rootCmd.Flags().StringSliceVar(&flagExtensions, "extensions", []string{}, "A list of extensions that narrows down the list of files in the calculation; many restrictions are separated by commas, for example, '.go,.md'")
	rootCmd.Flags().StringSliceVar(&flagLanguages, "languages", []string{}, "A list of languages (programming, markup, etc.), narrowing the list of files in the calculation; many restrictions are separated by commas, for example 'go,markdown'")
	rootCmd.Flags().StringSliceVar(&flagExclude, "exclude", []string{}, "A set of Glob patterns excluding files from the calculation, for example 'foo/*,bar/*'")
	rootCmd.Flags().StringSliceVar(&flagRestrictTo, "restrict-to", []string{}, "A set of Glob patterns that excludes all files that do not satisfy any of the patterns in the set")
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
