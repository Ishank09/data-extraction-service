package main

import (
	"fmt"
	"os"
)

func main() {
	command := NewCommand()

	if err := command.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
