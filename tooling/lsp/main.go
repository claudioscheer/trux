package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	server := NewServer(bufio.NewReader(os.Stdin), os.Stdout)
	if err := server.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
