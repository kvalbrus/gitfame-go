package gitfame

import (
	"bufio"
	"os/exec"
)

type LineCMD struct {
	Name string
	Dir  string
	Args []string
}

func (lcmd *LineCMD) Exec() ([]string, error) {
	cmd := exec.Command(lcmd.Name, lcmd.Args...)
	cmd.Dir = lcmd.Dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err = cmd.Start(); err != nil {
		return nil, err
	}

	lines := make([]string, 0)
	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err = scanner.Err(); err != nil {
		return nil, err
	}

	if err = cmd.Wait(); err != nil {
		return nil, err
	}

	return lines, nil
}
