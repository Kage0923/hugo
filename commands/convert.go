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
	"path"
	"time"

	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/spf13/hugo/hugolib"
	"github.com/spf13/hugo/parser"
	jww "github.com/spf13/jwalterweatherman"
)

var OutputDir string
var Unsafe bool

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert will modify your content to different formats",
	Long:  `Convert will modify your content to different formats`,
	Run:   nil,
}

var toJSONCmd = &cobra.Command{
	Use:   "toJSON",
	Short: "Convert front matter to JSON",
	Long: `toJSON will convert all front matter in the content
	directory to use JSON for the front matter`,
	Run: func(cmd *cobra.Command, args []string) {
		err := convertContents(rune([]byte(parser.JSON_LEAD)[0]))
		if err != nil {
			jww.ERROR.Println(err)
		}
	},
}

var toTOMLCmd = &cobra.Command{
	Use:   "toTOML",
	Short: "Convert front matter to TOML",
	Long: `toTOML will convert all front matter in the content
	directory to use TOML for the front matter`,
	Run: func(cmd *cobra.Command, args []string) {
		err := convertContents(rune([]byte(parser.TOML_LEAD)[0]))
		if err != nil {
			jww.ERROR.Println(err)
		}
	},
}

var toYAMLCmd = &cobra.Command{
	Use:   "toYAML",
	Short: "Convert front matter to YAML",
	Long: `toYAML will convert all front matter in the content
	directory to use YAML for the front matter`,
	Run: func(cmd *cobra.Command, args []string) {
		err := convertContents(rune([]byte(parser.YAML_LEAD)[0]))
		if err != nil {
			jww.ERROR.Println(err)
		}
	},
}

func init() {
	convertCmd.AddCommand(toJSONCmd)
	convertCmd.AddCommand(toTOMLCmd)
	convertCmd.AddCommand(toYAMLCmd)
	convertCmd.PersistentFlags().StringVarP(&OutputDir, "output", "o", "", "filesystem path to write files to")
	convertCmd.PersistentFlags().BoolVar(&Unsafe, "unsafe", false, "enable less safe operations, please backup first")
}

func convertContents(mark rune) (err error) {
	InitializeConfig()
	site := &hugolib.Site{}

	if err := site.Initialise(); err != nil {
		return err
	}

	if site.Source == nil {
		panic(fmt.Sprintf("site.Source not set"))
	}
	if len(site.Source.Files()) < 1 {
		return fmt.Errorf("No source files found")
	}

	jww.FEEDBACK.Println("processing", len(site.Source.Files()), "content files")
	for _, file := range site.Source.Files() {
		jww.INFO.Println("Attempting to convert", file.LogicalName)
		page, err := hugolib.NewPage(file.LogicalName)
		if err != nil {
			return err
		}

		psr, err := parser.ReadFrom(file.Contents)
		if err != nil {
			jww.ERROR.Println("Error processing file:", path.Join(file.Dir, file.LogicalName))
			return err
		}
		metadata, err := psr.Metadata()
		if err != nil {
			jww.ERROR.Println("Error processing file:", path.Join(file.Dir, file.LogicalName))
			return err
		}

		// better handling of dates in formats that don't have support for them
		if mark == parser.FormatToLeadRune("json") || mark == parser.FormatToLeadRune("yaml") {
			newmetadata := cast.ToStringMap(metadata)
			for k, v := range newmetadata {
				switch vv := v.(type) {
				case time.Time:
					newmetadata[k] = vv.Format(time.RFC3339)
				}
			}
			metadata = newmetadata
		}

		page.Dir = file.Dir
		page.SetSourceContent(psr.Content())
		page.SetSourceMetaData(metadata, mark)

		if OutputDir != "" {
			page.SaveSourceAs(path.Join(OutputDir, page.FullFilePath()))
		} else {
			if Unsafe {
				page.SaveSource()
			} else {
				jww.FEEDBACK.Println("Unsafe operation not allowed, use --unsafe or set a different output path")
			}
		}
	}
	return
}
