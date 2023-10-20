/*
Copyright 2023 The Ketches Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generate the autocompletion script for the specified shell",
	Long: fmt.Sprintf(`To load completions:

Bash:
	$ source <(%[1]s completion bash)
	# To load completions for each session, execute once:
	# Linux:
	$ %[1]s completion bash > /etc/bash_completion.d/%[1]s
	# macOS:
	$ %[1]s completion bash > $(brew --prefix)/etc/bash_completion.d/%[1]s

Zsh:
	# If shell completion is not already enabled in your environment,
	# you will need to enable it.  You can execute the following once:
	$ echo "autoload -U compinit; compinit" >> ~/.zshrc
	# To load completions for each session, execute once:
	$ %[1]s completion zsh > "${fpath[1]}/_%[1]s"
	# You will need to start a new shell for this setup to take effect.

Fish:
	$ %[1]s completion fish | source
	# To load completions for each session, execute once:
	$ %[1]s completion fish > ~/.config/fish/completions/%[1]s.fish

PowerShell:
	PS> %[1]s completion powershell | Out-String | Invoke-Expression
	# To load completions for every new session, run:
	PS> %[1]s completion powershell > %[1]s.ps1
	# and source this file from your PowerShell profile.
	`, "ketches"),
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		switch args[0] {
		case "bash":
			err = cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			err = cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			err = cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			err = cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// completionCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// completionCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
