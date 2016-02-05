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
	"github.com/spf13/cobra"
	"github.com/spf13/hugo/hugolib"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check content in the source directory",
	Long: `Hugo will perform some basic analysis on the content provided
and will give feedback.`,
}

func init() {
	initHugoBuilderFlags(checkCmd)
	checkCmd.RunE = check
}

func check(cmd *cobra.Command, args []string) error {
	if err := InitializeConfig(checkCmd); err != nil {
		return err
	}
	site := hugolib.Site{}

	return site.Analyze()
}
