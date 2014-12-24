// Copyright © 2013-14 Steve Francia <spf@spf13.com>.
//
// Licensed under the Simple Public License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://opensource.org/licenses/Simple-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/hugo/helpers"
	"github.com/spf13/hugo/hugofs"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
)

var serverPort int
var serverWatch bool
var serverAppend bool
var disableLiveReload bool

//var serverCmdV *cobra.Command

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Hugo runs its own webserver to render the files",
	Long: `Hugo is able to run its own high performance web server.
Hugo will render all the files defined in the source directory and
serve them up.`,
	//Run: server,
}

func init() {
	serverCmd.Flags().IntVarP(&serverPort, "port", "p", 1313, "port to run the server on")
	serverCmd.Flags().BoolVarP(&serverWatch, "watch", "w", false, "watch filesystem for changes and recreate as needed")
	serverCmd.Flags().BoolVarP(&serverAppend, "appendPort", "", true, "append port to baseurl")
	serverCmd.Flags().BoolVar(&disableLiveReload, "disableLiveReload", false, "watch without enabling live browser reload on rebuild")
	serverCmd.Flags().String("memstats", "", "log memory usage to this file")
	serverCmd.Flags().Int("meminterval", 100, "interval to poll memory usage (requires --memstats)")
	serverCmd.Run = server
}

func server(cmd *cobra.Command, args []string) {
	InitializeConfig()

	if cmd.Flags().Lookup("disableLiveReload").Changed {
		viper.Set("DisableLiveReload", disableLiveReload)
	}

	if serverWatch {
		viper.Set("Watch", true)
	}

	l, err := net.Listen("tcp", ":"+strconv.Itoa(serverPort))
	if err == nil {
		l.Close()
	} else {
		jww.ERROR.Println("port", serverPort, "already in use, attempting to use an available port")
		sp, err := helpers.FindAvailablePort()
		if err != nil {
			jww.ERROR.Println("Unable to find alternative port to use")
			jww.ERROR.Fatalln(err)
		}
		serverPort = sp.Port
	}

	viper.Set("port", serverPort)

	BaseUrl, err := fixUrl(BaseUrl)
	if err != nil {
		jww.ERROR.Fatal(err)
	}
	viper.Set("BaseUrl", BaseUrl)

	if err := memStats(); err != nil {
		jww.ERROR.Println("memstats error:", err)
	}

	build(serverWatch)

	// Watch runs its own server as part of the routine
	if serverWatch {
		jww.FEEDBACK.Println("Watching for changes in", helpers.AbsPathify(viper.GetString("ContentDir")))
		err := NewWatcher(serverPort)
		if err != nil {
			fmt.Println(err)
		}
	}

	serve(serverPort)
}

func serve(port int) {
	jww.FEEDBACK.Println("Serving pages from " + helpers.AbsPathify(viper.GetString("PublishDir")))

	httpFs := &afero.HttpFs{SourceFs: hugofs.DestinationFS}
	fileserver := http.FileServer(httpFs.Dir(helpers.AbsPathify(viper.GetString("PublishDir"))))

	u, err := url.Parse(viper.GetString("BaseUrl"))
	if err != nil {
		jww.ERROR.Fatalf("Invalid BaseUrl: %s", err)
	}
	if u.Path == "" || u.Path == "/" {
		http.Handle("/", fileserver)
	} else {
		http.Handle(u.Path, http.StripPrefix(u.Path, fileserver))
	}

	u.Scheme = "http"
	jww.FEEDBACK.Printf("Web Server is available at %s\n", u.String())
	fmt.Println("Press ctrl+c to stop")

	err = http.ListenAndServe(":"+strconv.Itoa(port), nil)
	if err != nil {
		jww.ERROR.Printf("Error: %s\n", err.Error())
		os.Exit(1)
	}
}

func fixUrl(s string) (string, error) {
	useLocalhost := false
	if s == "" {
		s = viper.GetString("BaseUrl")
		useLocalhost = true
	}
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		s = "http://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", err
	}

	if serverAppend {
		if useLocalhost {
			u.Host = fmt.Sprintf("localhost:%d", serverPort)
			return u.String(), nil
		}
		host := u.Host
		if strings.Contains(host, ":") {
			host, _, err = net.SplitHostPort(u.Host)
			if err != nil {
				return "", fmt.Errorf("Failed to split BaseUrl hostpost: %s", err)
			}
		}
		u.Host = fmt.Sprintf("%s:%d", host, serverPort)
		return u.String(), nil
	}

	if useLocalhost {
		u.Host = "localhost"
	}
	return u.String(), nil
}

func memStats() error {
	memstats := serverCmd.Flags().Lookup("memstats").Value.String()
	if memstats != "" {
		interval, err := time.ParseDuration(serverCmd.Flags().Lookup("meminterval").Value.String())
		if err != nil {
			interval, _ = time.ParseDuration("100ms")
		}

		fileMemStats, err := os.Create(memstats)
		if err != nil {
			return err
		}

		fileMemStats.WriteString("# Time\tHeapSys\tHeapAlloc\tHeapIdle\tHeapReleased\n")

		go func() {
			var stats runtime.MemStats

			start := time.Now().UnixNano()

			for {
				runtime.ReadMemStats(&stats)
				if fileMemStats != nil {
					fileMemStats.WriteString(fmt.Sprintf("%d\t%d\t%d\t%d\t%d\n",
						(time.Now().UnixNano()-start)/1000000, stats.HeapSys, stats.HeapAlloc, stats.HeapIdle, stats.HeapReleased))
					time.Sleep(interval)
				} else {
					break
				}
			}
		}()
	}
	return nil
}
