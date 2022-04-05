// Copyright 2021 The Hugo Authors. All rights reserved.
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

package page_test

import (
	"testing"

	"github.com/gohugoio/hugo/hugolib"
)

func TestGroupByLocalizedDate(t *testing.T) {

	files := `
-- config.toml --
defaultContentLanguage = 'en'
defaultContentLanguageInSubdir = true
[languages]
[languages.en]
title = 'My blog'
weight = 1
[languages.fr]
title = 'Mon blogue'
weight = 2
[languages.nn]
title = 'Bloggen min'
weight = 3
-- content/p1.md --
---
title: "Post 1"
date: "2020-01-01"
---
-- content/p2.md --
---
title: "Post 2"
date: "2020-02-01"
---
-- content/p1.fr.md --
---
title: "Post 1"
date: "2020-01-01"
---
-- content/p2.fr.md --
---
title: "Post 2"
date: "2020-02-01"
---
-- layouts/index.html --
{{ range $k, $v := site.RegularPages.GroupByDate "January, 2006" }}{{ $k }}|{{ $v.Key }}|{{ $v.Pages }}{{ end }}

	`

	b := hugolib.NewIntegrationTestBuilder(
		hugolib.IntegrationTestConfig{
			T:           t,
			TxtarString: files,
			NeedsOsFS:   true,
		}).Build()

	b.AssertFileContent("public/en/index.html", "0|February, 2020|Pages(1)1|January, 2020|Pages(1)")
	b.AssertFileContent("public/fr/index.html", "0|février, 2020|Pages(1)1|janvier, 2020|Pages(1)")
}
