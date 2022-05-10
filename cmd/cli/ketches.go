package main

import (
	"github.com/spf13/cobra"
)

func main() {
	root := cobra.Command{
		Use:     "ketches",
		Short:   "ketches is a command line tool for managing ketches cicd workflow.",
		Long:    "ketches is a command line tool for managing ketches cicd workflow.",
		Version: "v0.0.1",
	}

	cicd := cobra.Command{
		Use:   "cicd",
		Short: "cicd is a command line tool for build and deploy ketches applications.",
		Long:  "cicd is a command line tool for build and deploy ketches applications.",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	log := cobra.Command{
		Use:   "log",
		Short: "log is a command line tool for log ketchies cicd workflow.",
		Long:  "log is a command line tool for log ketchies cicd workflow.",
	}

	root.AddCommand(&cicd)
	root.AddCommand(&log)
}
