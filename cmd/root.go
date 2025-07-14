package main

import (
	"os"

	//nolint:depguard

	"github.com/charmbracelet/log" //nolint:depguard
	"github.com/ivasania/data-extraction-service/pkg/logging"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	var verbose bool

	command := &cobra.Command{
		Use:   "data-extraction-service",
		Short: "Data extraction service",
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			if verbose {
				logging.SetLogLevel(log.DebugLevel)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				panic(err)
			}
			os.Exit(1)
		},
	}

	command.AddCommand(getVersionCmd())
	command.AddCommand(server.GetServerCmd())
	command.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	return command
}
