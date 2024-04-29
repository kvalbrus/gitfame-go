package gitfame

import (
	"strconv"
	"strings"
)

type GitfameParser struct {
	Git          GitConf
	UseCommitter bool
}

func NewGitfameParser(git GitConf, useCommiter bool) GitfameParser {
	return GitfameParser{Git: git, UseCommitter: useCommiter}
}

func (parser *GitfameParser) parse(files []string) (map[string]Author, error) {
	commits := make(map[string]Commit)
	for _, file := range files {
		lines, err := parser.Git.Blame(file)
		if err != nil {
			return nil, err
		}

		if len(lines) == 0 {
			lines, err = parser.Git.Log(file)
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
			} else if parser.UseCommitter && strings.HasPrefix(line, "committer ") {
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
