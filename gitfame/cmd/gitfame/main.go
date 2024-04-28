//go:build !solution

package main

import (
	"gitlab.com/slon/shad-go/gitfame/cmd/gitfame/cmd"
	"os"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
