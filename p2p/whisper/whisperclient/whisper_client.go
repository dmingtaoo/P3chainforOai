package main

import (
	"os"
	"p3Chain/p2p/whisper/cli"
)

func main() {
	defer os.Exit(0)
	cmd := cli.NewCommandLine()
	cmd.Run()
}
