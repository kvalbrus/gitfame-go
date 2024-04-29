//go:build !solution

package main

import (
	"gitlab.com/slon/shad-go/gitfame/cmd/gitfame/cmd/gitfame"
	"os"
)

func main() {
	if err := gitfame.Execute(); err != nil {
		os.Exit(1)
	}
}
