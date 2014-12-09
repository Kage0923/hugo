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

package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bitbucket.org/kardianos/osext"
	"github.com/spf13/cobra"
)

var timeLayout string // the layout for time.Time

var (
	commitHash string
	buildDate  string
)

var version = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Hugo",
	Long:  `All software has versions. This is Hugo's`,
	Run: func(cmd *cobra.Command, args []string) {
		if buildDate == "" {
			setBuildDate() // set the build date from executable's mdate
		} else {
			formatBuildDate() // format the compile time
		}
		if commitHash == "" {
			fmt.Printf("Hugo Static Site Generator v0.13-DEV buildDate: %s\n", buildDate)
		} else {
			fmt.Printf("Hugo Static Site Generator v0.13-DEV-%s buildDate: %s\n", strings.ToUpper(commitHash), buildDate)
		}
	},
}

// setBuildDate checks the ModTime of the Hugo executable and returns it as a
// formatted string.  This assumes that the executable name is Hugo, if it does
// not exist, an empty string will be returned.  This is only called if the
// buildDate wasn't set during compile time.
//
// osext is used for cross-platform.
func setBuildDate() {
	fname, _ := osext.Executable()
	dir, err := filepath.Abs(filepath.Dir(fname))
	if err != nil {
		fmt.Println(err)
		return
	}
	fi, err := os.Lstat(filepath.Join(dir, "hugo"))
	if err != nil {
		fmt.Println(err)
		return
	}
	t := fi.ModTime()
	buildDate = t.Format(time.RFC3339)
}

// formatBuildDate formats the buildDate according to the value in
// .Params.DateFormat, if it's set.
func formatBuildDate() {
	t, _ := time.Parse("2006-01-02T15:04:05-0700", buildDate)
	buildDate = t.Format(time.RFC3339)
}
