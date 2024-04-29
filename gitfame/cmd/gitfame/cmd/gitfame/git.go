package gitfame

type GitConf struct {
	Repository string
	Revision   string
}

func (gc *GitConf) LsTree() ([]string, error) {
	cmd := &LineCMD{Name: "git", Args: []string{"ls-tree", "-r", "--name-only", gc.Revision}, Dir: gc.Repository}
	return cmd.Exec()
}

func (gc *GitConf) Blame(file string) ([]string, error) {
	cmd := &LineCMD{Name: "git", Args: []string{"blame", "-p", gc.Revision, "--", file}, Dir: gc.Repository}
	return cmd.Exec()
}

func (gc *GitConf) Log(file string) ([]string, error) {
	cmd := &LineCMD{Name: "git", Args: []string{"log", "-1", "--pretty=format:%H\n%an", gc.Revision, "--", file}, Dir: gc.Repository}
	return cmd.Exec()
}

func (gc *GitConf) Files(parser FileParser) ([]string, error) {
	files, err := gc.LsTree()
	if err != nil {
		return nil, err
	}

	return parser.Parse(files)
}

func NewGitConfig(repository string, revision string) GitConf {
	return GitConf{Repository: repository, Revision: revision}
}
