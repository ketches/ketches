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
	"context"
	"fmt"

	"github.com/ketches/ketches/internal/service"
	"github.com/ketches/ketches/pkg/global"
	"github.com/spf13/cobra"
)

// spacesCmd represents the spaces command
var spacesCmd = &cobra.Command{
	Use:   "spaces",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		spaceService := service.NewSpaceService()
		ctx := context.WithValue(context.Background(), global.ContextKeyAccountID, "admin")
		ctx = context.WithValue(ctx, global.ContextKeySignInRole, "admin")
		resp, err := spaceService.List(ctx)
		if err != nil {
			fmt.Println(err)
			return
		}
		for _, space := range resp.Spaces {
			fmt.Println(space.Name)
		}
		fmt.Println("spaces called")
	},
}

func init() {
	getCmd.AddCommand(spacesCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// spacesCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// spacesCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
