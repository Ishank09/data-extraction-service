package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func getVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version of the application",
		Long:  "Print the version of the application",
		Run: func(*cobra.Command, []string) {
			fmt.Println("Data extraction service v0.1.0")
		},
	}
}
