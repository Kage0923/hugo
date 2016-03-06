// Copyright 2015 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
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
	"mime"
)

var (
	disableLiveReload bool
	renderToDisk      bool
	serverAppend      bool
	serverInterface   string
	serverPort        int
	serverWatch       bool
)

//var serverCmdV *cobra.Command

var serverCmd = &cobra.Command{
	Use:     "server",
	Aliases: []string{"serve"},
	Short:   "A high performance webserver",
	Long: `Hugo provides its own webserver which builds and serves the site.
While hugo server is high performance, it is a webserver with limited options.
Many run it in production, but the standard behavior is for people to use it
in development and use a more full featured server such as Nginx or Caddy.

'hugo server' will avoid writing the rendered and served content to disk,
preferring to store it in memory.

By default hugo will also watch your files for any changes you make and
automatically rebuild the site. It will then live reload any open browser pages
and push the latest content to them. As most Hugo sites are built in a fraction
of a second, you will be able to save and see your changes nearly instantly.`,
	//RunE: server,
}

type filesOnlyFs struct {
	fs http.FileSystem
}

type noDirFile struct {
	http.File
}

func (fs filesOnlyFs) Open(name string) (http.File, error) {
	f, err := fs.fs.Open(name)
	if err != nil {
		return nil, err
	}
	return noDirFile{f}, nil
}

func (f noDirFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

func init() {
	initHugoBuilderFlags(serverCmd)

	serverCmd.Flags().IntVarP(&serverPort, "port", "p", 1313, "port on which the server will listen")
	serverCmd.Flags().StringVarP(&serverInterface, "bind", "", "127.0.0.1", "interface to which the server will bind")
	serverCmd.Flags().BoolVarP(&serverWatch, "watch", "w", true, "watch filesystem for changes and recreate as needed")
	serverCmd.Flags().BoolVarP(&serverAppend, "appendPort", "", true, "append port to baseurl")
	serverCmd.Flags().BoolVar(&disableLiveReload, "disableLiveReload", false, "watch without enabling live browser reload on rebuild")
	serverCmd.Flags().BoolVar(&renderToDisk, "renderToDisk", false, "render to Destination path (default is render to memory & serve from there)")
	serverCmd.Flags().String("memstats", "", "log memory usage to this file")
	serverCmd.Flags().Int("meminterval", 100, "interval to poll memory usage (requires --memstats)")
	serverCmd.RunE = server

	mime.AddExtensionType(".json", "application/json; charset=utf8")
}

func server(cmd *cobra.Command, args []string) error {
	if err := InitializeConfig(serverCmd); err != nil {
		return err
	}

	if flagChanged(cmd.Flags(), "disableLiveReload") {
		viper.Set("DisableLiveReload", disableLiveReload)
	}

	if serverWatch {
		viper.Set("Watch", true)
	}

	if viper.GetBool("watch") {
		serverWatch = true
		watchConfig()
	}

	l, err := net.Listen("tcp", net.JoinHostPort(serverInterface, strconv.Itoa(serverPort)))
	if err == nil {
		l.Close()
	} else {
		if flagChanged(serverCmd.Flags(), "port") {
			// port set explicitly by user -- he/she probably meant it!
			return newSystemErrorF("Port %d already in use", serverPort)
		}
		jww.ERROR.Println("port", serverPort, "already in use, attempting to use an available port")
		sp, err := helpers.FindAvailablePort()
		if err != nil {
			return newSystemError("Unable to find alternative port to use:", err)
		}
		serverPort = sp.Port
	}

	viper.Set("port", serverPort)

	BaseURL, err := fixURL(baseURL)
	if err != nil {
		return err
	}
	viper.Set("BaseURL", BaseURL)

	if err := memStats(); err != nil {
		jww.ERROR.Println("memstats error:", err)
	}

	// If a Destination is provided via flag write to disk
	if destination != "" {
		renderToDisk = true
	}

	// Hugo writes the output to memory instead of the disk
	if !renderToDisk {
		hugofs.DestinationFS = new(afero.MemMapFs)
		// Rendering to memoryFS, publish to Root regardless of publishDir.
		viper.Set("PublishDir", "/")
	}

	if err := build(serverWatch); err != nil {
		return err
	}

	// Watch runs its own server as part of the routine
	if serverWatch {
		watchDirs := getDirList()
		baseWatchDir := viper.GetString("WorkingDir")
		for i, dir := range watchDirs {
			watchDirs[i], _ = helpers.GetRelativePath(dir, baseWatchDir)
		}

		rootWatchDirs := strings.Join(helpers.UniqueStrings(helpers.ExtractRootPaths(watchDirs)), ",")

		jww.FEEDBACK.Printf("Watching for changes in %s%s{%s}\n", baseWatchDir, helpers.FilePathSeparator, rootWatchDirs)
		err := NewWatcher(serverPort)

		if err != nil {
			return err
		}
	}

	serve(serverPort)

	return nil
}

func serve(port int) {
	if renderToDisk {
		jww.FEEDBACK.Println("Serving pages from " + helpers.AbsPathify(viper.GetString("PublishDir")))
	} else {
		jww.FEEDBACK.Println("Serving pages from memory")
	}

	httpFs := afero.NewHttpFs(hugofs.DestinationFS)
	fs := filesOnlyFs{httpFs.Dir(helpers.AbsPathify(viper.GetString("PublishDir")))}
	fileserver := http.FileServer(fs)

	// We're only interested in the path
	u, err := url.Parse(viper.GetString("BaseURL"))
	if err != nil {
		jww.ERROR.Fatalf("Invalid BaseURL: %s", err)
	}
	if u.Path == "" || u.Path == "/" {
		http.Handle("/", fileserver)
	} else {
		http.Handle(u.Path, http.StripPrefix(u.Path, fileserver))
	}

	u.Scheme = "http"
	jww.FEEDBACK.Printf("Web Server is available at %s (bind address %s)\n", u.String(), serverInterface)
	fmt.Println("Press Ctrl+C to stop")

	endpoint := net.JoinHostPort(serverInterface, strconv.Itoa(port))
	err = http.ListenAndServe(endpoint, nil)
	if err != nil {
		jww.ERROR.Printf("Error: %s\n", err.Error())
		os.Exit(1)
	}
}

// fixURL massages the BaseURL into a form needed for serving
// all pages correctly.
func fixURL(s string) (string, error) {
	useLocalhost := false
	if s == "" {
		s = viper.GetString("BaseURL")
		useLocalhost = true
	}
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		s = "http://" + s
	}
	if !strings.HasSuffix(s, "/") {
		s = s + "/"
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", err
	}

	if serverAppend {
		if useLocalhost {
			u.Host = fmt.Sprintf("localhost:%d", serverPort)
			u.Scheme = "http"
			return u.String(), nil
		}
		host := u.Host
		if strings.Contains(host, ":") {
			host, _, err = net.SplitHostPort(u.Host)
			if err != nil {
				return "", fmt.Errorf("Failed to split BaseURL hostpost: %s", err)
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
