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
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("login called")
		call()
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// loginCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// loginCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func call() {
	// 定义本地代理服务器的地址和端口
	proxyAddr := "127.0.0.1:8001"

	// 创建反向代理
	proxy := NewProxy()

	// 启动代理服务器
	go http.ListenAndServe(proxyAddr, proxy)

	time.Sleep(time.Second)
	resp, err := http.Get("http://" + proxyAddr)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	io.Copy(os.Stdout, resp.Body)
}

func NewProxy() http.Handler {
	// 创建一个反向代理器
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			// 修改请求目标为实际的服务地址
			req.URL.Scheme = "http"
			req.URL.Host = "localhost:8001" // 实际的 order-api 服务地址
			// 可以在这里进行一些其他的请求修改，如头部信息
		},
	}

	// 创建一个处理函数来处理代理请求
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 根据请求路径判断是否需要代理
		if strings.HasPrefix(r.URL.Path, "/api/v1/namespaces/default/services/nginx/proxy/") {
			proxy.ServeHTTP(w, r)
		} else {
			// 处理其他请求，如返回 404
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, "Not Found")
		}
	})
}
