// Copyright 2022 The Hugo Authors. All rights reserved.
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

package i18n_test

import (
	"testing"

	"github.com/gohugoio/hugo/hugolib"
)

func TestI18nFromTheme(t *testing.T) {
	t.Parallel()

	files := `
-- config.toml --
[module]
[[module.imports]]
path = "mytheme"
-- i18n/en.toml --
[l1]
other = 'l1main'
[l2]
other = 'l2main'
-- themes/mytheme/i18n/en.toml --
[l1]
other = 'l1theme'
[l2]
other = 'l2theme'
[l3]
other = 'l3theme'
-- layouts/index.html --
l1: {{ i18n "l1"  }}|l2: {{ i18n "l2"  }}|l3: {{ i18n "l3"  }}

`

	b := hugolib.NewIntegrationTestBuilder(
		hugolib.IntegrationTestConfig{
			T:           t,
			TxtarString: files,
		},
	).Build()

	b.AssertFileContent("public/index.html", `
l1: l1main|l2: l2main|l3: l3theme
	`)
}
