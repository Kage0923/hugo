// Copyright © 2013 Steve Francia <spf@spf13.com>.
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

//Package commands defines and implements command-line commands and flags used by Hugo. Commands and flags are implemented using
//cobra.
package commands

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/fsync"
	"github.com/spf13/hugo/helpers"
	"github.com/spf13/hugo/hugofs"
	"github.com/spf13/hugo/hugolib"
	"github.com/spf13/hugo/livereload"
	"github.com/spf13/hugo/utils"
	"github.com/spf13/hugo/watcher"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/nitro"
	"github.com/spf13/viper"
)

//HugoCmd is Hugo's root command. Every other command attached to HugoCmd is a child command to it.
var HugoCmd = &cobra.Command{
	Use:   "hugo",
	Short: "Hugo is a very fast static site generator",
	Long: `A Fast and Flexible Static Site Generator built with
love by spf13 and friends in Go.

Complete documentation is available at http://gohugo.io`,
	Run: func(cmd *cobra.Command, args []string) {
		InitializeConfig()
		build()
	},
}

var hugoCmdV *cobra.Command

//Flags that are to be added to commands.
var BuildWatch, Draft, Future, UglyUrls, Verbose, Logging, VerboseLog, DisableRSS, DisableSitemap, PluralizeListTitles, NoTimes bool
var Source, Destination, Theme, BaseUrl, CfgFile, LogFile, Editor string

//Execute adds all child commands to the root command HugoCmd and sets flags appropriately.
func Execute() {
	AddCommands()
	utils.StopOnErr(HugoCmd.Execute())
}

//AddCommands adds child commands to the root command HugoCmd.
func AddCommands() {
	HugoCmd.AddCommand(serverCmd)
	HugoCmd.AddCommand(version)
	HugoCmd.AddCommand(check)
	HugoCmd.AddCommand(benchmark)
	HugoCmd.AddCommand(convertCmd)
	HugoCmd.AddCommand(newCmd)
	HugoCmd.AddCommand(listCmd)
}

//Initializes flags
func init() {
	HugoCmd.PersistentFlags().BoolVarP(&Draft, "buildDrafts", "D", false, "include content marked as draft")
	HugoCmd.PersistentFlags().BoolVarP(&Future, "buildFuture", "F", false, "include content with datePublished in the future")
	HugoCmd.PersistentFlags().BoolVar(&DisableRSS, "disableRSS", false, "Do not build RSS files")
	HugoCmd.PersistentFlags().BoolVar(&DisableSitemap, "disableSitemap", false, "Do not build Sitemap file")
	HugoCmd.PersistentFlags().StringVarP(&Source, "source", "s", "", "filesystem path to read files relative from")
	HugoCmd.PersistentFlags().StringVarP(&Destination, "destination", "d", "", "filesystem path to write files to")
	HugoCmd.PersistentFlags().StringVarP(&Theme, "theme", "t", "", "theme to use (located in /themes/THEMENAME/)")
	HugoCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
	HugoCmd.PersistentFlags().BoolVar(&UglyUrls, "uglyUrls", false, "if true, use /filename.html instead of /filename/")
	HugoCmd.PersistentFlags().StringVarP(&BaseUrl, "baseUrl", "b", "", "hostname (and path) to the root eg. http://spf13.com/")
	HugoCmd.PersistentFlags().StringVar(&CfgFile, "config", "", "config file (default is path/config.yaml|json|toml)")
	HugoCmd.PersistentFlags().StringVar(&Editor, "editor", "", "edit new content with this editor, if provided")
	HugoCmd.PersistentFlags().BoolVar(&Logging, "log", false, "Enable Logging")
	HugoCmd.PersistentFlags().StringVar(&LogFile, "logFile", "", "Log File path (if set, logging enabled automatically)")
	HugoCmd.PersistentFlags().BoolVar(&VerboseLog, "verboseLog", false, "verbose logging")
	HugoCmd.PersistentFlags().BoolVar(&nitro.AnalysisOn, "stepAnalysis", false, "display memory and timing of different steps of the program")
	HugoCmd.PersistentFlags().BoolVar(&PluralizeListTitles, "pluralizeListTitles", true, "Pluralize titles in lists using inflect")
	HugoCmd.Flags().BoolVarP(&BuildWatch, "watch", "w", false, "watch filesystem for changes and recreate as needed")
	HugoCmd.Flags().BoolVarP(&NoTimes, "noTimes", "", false, "Don't sync modification time of files")
	hugoCmdV = HugoCmd
}

// InitializeConfig initializes a config file with sensible default configuration flags.
func InitializeConfig() {
	viper.SetConfigFile(CfgFile)
	viper.AddConfigPath(Source)
	err := viper.ReadInConfig()
	if err != nil {
		jww.ERROR.Println("Unable to locate Config file. Perhaps you need to create a new site. Run `hugo help new` for details")
	}

	viper.RegisterAlias("taxonomies", "indexes")

	viper.SetDefault("Watch", false)
	viper.SetDefault("MetaDataFormat", "toml")
	viper.SetDefault("DisableRSS", false)
	viper.SetDefault("DisableSitemap", false)
	viper.SetDefault("ContentDir", "content")
	viper.SetDefault("LayoutDir", "layouts")
	viper.SetDefault("StaticDir", "static")
	viper.SetDefault("ArchetypeDir", "archetypes")
	viper.SetDefault("PublishDir", "public")
	viper.SetDefault("DefaultLayout", "post")
	viper.SetDefault("BuildDrafts", false)
	viper.SetDefault("BuildFuture", false)
	viper.SetDefault("UglyUrls", false)
	viper.SetDefault("Verbose", false)
	viper.SetDefault("CanonifyUrls", false)
	viper.SetDefault("Indexes", map[string]string{"tag": "tags", "category": "categories"})
	viper.SetDefault("Permalinks", make(hugolib.PermalinkOverrides, 0))
	viper.SetDefault("Sitemap", hugolib.Sitemap{Priority: -1})
	viper.SetDefault("PygmentsStyle", "monokai")
	viper.SetDefault("DefaultExtension", "html")
	viper.SetDefault("PygmentsUseClasses", false)
	viper.SetDefault("DisableLiveReload", false)
	viper.SetDefault("PluralizeListTitles", true)
	viper.SetDefault("FootnoteAnchorPrefix", "")
	viper.SetDefault("FootnoteReturnLinkContents", "")
	viper.SetDefault("NewContentEditor", "")
	viper.SetDefault("Blackfriday", map[string]bool{"angledQuotes": false})

	if hugoCmdV.PersistentFlags().Lookup("buildDrafts").Changed {
		viper.Set("BuildDrafts", Draft)
	}

	if hugoCmdV.PersistentFlags().Lookup("buildFuture").Changed {
		viper.Set("BuildFuture", Future)
	}

	if hugoCmdV.PersistentFlags().Lookup("uglyUrls").Changed {
		viper.Set("UglyUrls", UglyUrls)
	}

	if hugoCmdV.PersistentFlags().Lookup("disableRSS").Changed {
		viper.Set("DisableRSS", DisableRSS)
	}

	if hugoCmdV.PersistentFlags().Lookup("disableSitemap").Changed {
		viper.Set("DisableSitemap", DisableSitemap)
	}

	if hugoCmdV.PersistentFlags().Lookup("verbose").Changed {
		viper.Set("Verbose", Verbose)
	}

	if hugoCmdV.PersistentFlags().Lookup("pluralizeListTitles").Changed {
		viper.Set("PluralizeListTitles", PluralizeListTitles)
	}

	if hugoCmdV.PersistentFlags().Lookup("editor").Changed {
		viper.Set("NewContentEditor", Editor)
	}

	if hugoCmdV.PersistentFlags().Lookup("logFile").Changed {
		viper.Set("LogFile", LogFile)
	}
	if BaseUrl != "" {
		if !strings.HasSuffix(BaseUrl, "/") {
			BaseUrl = BaseUrl + "/"
		}
		viper.Set("BaseUrl", BaseUrl)
	}

	if Theme != "" {
		viper.Set("theme", Theme)
	}

	if Destination != "" {
		viper.Set("PublishDir", Destination)
	}

	if Source != "" {
		viper.Set("WorkingDir", Source)
	} else {
		dir, _ := os.Getwd()
		viper.Set("WorkingDir", dir)
	}

	if VerboseLog || Logging || (viper.IsSet("LogFile") && viper.GetString("LogFile") != "") {
		if viper.IsSet("LogFile") && viper.GetString("LogFile") != "" {
			jww.SetLogFile(viper.GetString("LogFile"))
		} else {
			jww.UseTempLogFile("hugo")
		}
	} else {
		jww.DiscardLogging()
	}

	if viper.GetBool("verbose") {
		jww.SetStdoutThreshold(jww.LevelInfo)
	}

	if VerboseLog {
		jww.SetLogThreshold(jww.LevelInfo)
	}

	jww.INFO.Println("Using config file:", viper.ConfigFileUsed())
}

func build(watches ...bool) {
	utils.CheckErr(copyStatic(), fmt.Sprintf("Error copying static files to %s", helpers.AbsPathify(viper.GetString("PublishDir"))))
	watch := false
	if len(watches) > 0 && watches[0] {
		watch = true
	}
	utils.StopOnErr(buildSite(BuildWatch || watch))

	if BuildWatch {
		jww.FEEDBACK.Println("Watching for changes in", helpers.AbsPathify(viper.GetString("ContentDir")))
		jww.FEEDBACK.Println("Press ctrl+c to stop")
		utils.CheckErr(NewWatcher(0))
	}
}

func copyStatic() error {
	staticDir := helpers.AbsPathify(viper.GetString("StaticDir")) + "/"
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		jww.ERROR.Println("Unable to find Static Directory:", staticDir)
		return nil
	}

	publishDir := helpers.AbsPathify(viper.GetString("PublishDir")) + "/"

	syncer := fsync.NewSyncer()
	syncer.NoTimes = viper.GetBool("notimes")
	syncer.SrcFs = hugofs.SourceFs
	syncer.DestFs = hugofs.DestinationFS

	if themeSet() {
		themeDir := helpers.AbsPathify("themes/"+viper.GetString("theme")) + "/static/"
		if _, err := os.Stat(themeDir); os.IsNotExist(err) {
			jww.ERROR.Println("Unable to find static directory for theme:", viper.GetString("theme"), "in", themeDir)
			return nil
		}

		// Copy Static to Destination
		jww.INFO.Println("syncing from", themeDir, "to", publishDir)
		utils.CheckErr(syncer.Sync(publishDir, themeDir), fmt.Sprintf("Error copying static files of theme to %s", publishDir))
	}

	// Copy Static to Destination
	jww.INFO.Println("syncing from", staticDir, "to", publishDir)
	return syncer.Sync(publishDir, staticDir)
}

func getDirList() []string {
	var a []string
	walker := func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			jww.ERROR.Println("Walker: ", err)
			return nil
		}

		if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
			jww.ERROR.Printf("Symbolic links not supported, skipping '%s'", path)
			return nil
		}

		if fi.IsDir() {
			a = append(a, path)
		}
		return nil
	}

	filepath.Walk(helpers.AbsPathify(viper.GetString("ContentDir")), walker)
	filepath.Walk(helpers.AbsPathify(viper.GetString("LayoutDir")), walker)
	filepath.Walk(helpers.AbsPathify(viper.GetString("StaticDir")), walker)
	if themeSet() {
		filepath.Walk(helpers.AbsPathify("themes/"+viper.GetString("theme")), walker)
	}

	return a
}

func themeSet() bool {
	return viper.GetString("theme") != ""
}

func buildSite(watching ...bool) (err error) {
	startTime := time.Now()
	site := &hugolib.Site{}
	if len(watching) > 0 && watching[0] {
		site.RunMode.Watching = true
	}
	err = site.Build()
	if err != nil {
		return err
	}
	site.Stats()
	jww.FEEDBACK.Printf("in %v ms\n", int(1000*time.Since(startTime).Seconds()))

	return nil
}

//NewWatcher creates a new watcher to watch filesystem events.
func NewWatcher(port int) error {
	if runtime.GOOS == "darwin" {
		tweakLimit()
	}

	watcher, err := watcher.New(1 * time.Second)
	var wg sync.WaitGroup

	if err != nil {
		fmt.Println(err)
		return err
	}

	defer watcher.Close()

	wg.Add(1)

	for _, d := range getDirList() {
		if d != "" {
			_ = watcher.Watch(d)
		}
	}

	go func() {
		for {
			select {
			case evs := <-watcher.Event:
				jww.INFO.Println("File System Event:", evs)

				static_changed := false
				dynamic_changed := false
				static_files_changed := make(map[string]bool)

				for _, ev := range evs {
					ext := filepath.Ext(ev.Name)
					istemp := strings.HasSuffix(ext, "~") || (ext == ".swp") || (ext == ".tmp") || (strings.HasPrefix(ext, ".goutputstream"))
					if istemp {
						continue
					}
					// renames are always followed with Create/Modify
					if ev.IsRename() {
						continue
					}

					isstatic := strings.HasPrefix(ev.Name, helpers.AbsPathify(viper.GetString("StaticDir"))) || strings.HasPrefix(ev.Name, helpers.AbsPathify("themes/"+viper.GetString("theme"))+"/static/")
					static_changed = static_changed || isstatic
					dynamic_changed = dynamic_changed || !isstatic

					if isstatic {
						if staticPath, err := helpers.MakeStaticPathRelative(ev.Name); err == nil {
							static_files_changed[staticPath] = true
						}
					}

					// add new directory to watch list
					if s, err := os.Stat(ev.Name); err == nil && s.Mode().IsDir() {
						if ev.IsCreate() {
							watcher.Watch(ev.Name)
						}
					}
				}

				if static_changed {
					jww.FEEDBACK.Println("Static file changed, syncing\n")
					utils.StopOnErr(copyStatic(), fmt.Sprintf("Error copying static files to %s", helpers.AbsPathify(viper.GetString("PublishDir"))))

					if !BuildWatch && !viper.GetBool("DisableLiveReload") {
						// Will block forever trying to write to a channel that nobody is reading if livereload isn't initalized

						// force refresh when more than one file
						if len(static_files_changed) == 1 {
							for path := range static_files_changed {
								livereload.RefreshPath(path)
							}

						} else {
							livereload.ForceRefresh()
						}
					}
				}

				if dynamic_changed {
					fmt.Print("\nChange detected, rebuilding site\n")
					const layout = "2006-01-02 15:04 -0700"
					fmt.Println(time.Now().Format(layout))
					utils.StopOnErr(buildSite(true))

					if !BuildWatch && !viper.GetBool("DisableLiveReload") {
						// Will block forever trying to write to a channel that nobody is reading if livereload isn't initalized
						livereload.ForceRefresh()
					}
				}
			case err := <-watcher.Error:
				if err != nil {
					fmt.Println("error:", err)
				}
			}
		}
	}()

	if port > 0 {
		if !viper.GetBool("DisableLiveReload") {
			livereload.Initialize()
			http.HandleFunc("/livereload.js", livereload.ServeJS)
			http.HandleFunc("/livereload", livereload.Handler)
		}

		go serve(port)
	}

	wg.Wait()
	return nil
}
