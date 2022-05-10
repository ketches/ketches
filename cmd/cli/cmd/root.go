package cmd

import "github.com/spf13/cobra"

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {

}
