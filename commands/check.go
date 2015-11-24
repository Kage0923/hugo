// Copyright © 2013 Steve Francia <spf@spf13.com>.
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
	"github.com/spf13/cobra"
	"github.com/spf13/hugo/hugolib"
)

var check = &cobra.Command{
	Use:   "check",
	Short: "Check content in the source directory",
	Long: `Hugo will perform some basic analysis on the content provided
and will give feedback.`,
	Run: func(cmd *cobra.Command, args []string) {
		InitializeConfig()
		site := hugolib.Site{}
		site.Analyze()
	},
}
