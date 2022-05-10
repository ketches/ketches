package cmd

import "github.com/spf13/cobra"

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

var cicdCmd = &cobra.Command{
	Use:   "cicd",
	Short: "show resource image",
	Long:  `show k8s resource image`,
	RunE:  image,
}

func init() {
	rootCmd.AddCommand(cicdCmd)
}
