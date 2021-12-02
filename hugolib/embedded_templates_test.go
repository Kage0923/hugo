// Copyright 2018 The Hugo Authors. All rights reserved.
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

package hugolib

import (
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestInternalTemplatesImage(t *testing.T) {
	config := `
baseURL = "https://example.org"

[params]
images=["siteimg1.jpg", "siteimg2.jpg"]

`
	b := newTestSitesBuilder(t).WithConfigFile("toml", config)

	b.WithContent("mybundle/index.md", `---
title: My Bundle
date: 2021-02-26T18:02:00-01:00
lastmod: 2021-05-22T19:25:00-01:00
---
`)

	b.WithContent("mypage.md", `---
title: My Page
images: ["pageimg1.jpg", "pageimg2.jpg"]
date: 2021-02-26T18:02:00+01:00
lastmod: 2021-05-22T19:25:00+01:00
---
`)

	b.WithContent("mysite.md", `---
title: My Site
---
`)

	b.WithTemplatesAdded("_default/single.html", `

{{ template "_internal/twitter_cards.html" . }}
{{ template "_internal/opengraph.html" . }}
{{ template "_internal/schema.html" . }}

`)

	b.WithSunset("content/mybundle/featured-sunset.jpg")
	b.Build(BuildCfg{})

	b.AssertFileContent("public/mybundle/index.html", `
<meta name="twitter:image" content="https://example.org/mybundle/featured-sunset.jpg"/>
<meta name="twitter:title" content="My Bundle"/>
<meta property="og:title" content="My Bundle" />
<meta property="og:url" content="https://example.org/mybundle/" />
<meta property="og:image" content="https://example.org/mybundle/featured-sunset.jpg"/>
<meta property="article:published_time" content="2021-02-26T18:02:00-01:00" />
<meta property="article:modified_time" content="2021-05-22T19:25:00-01:00" />
<meta itemprop="name" content="My Bundle">
<meta itemprop="image" content="https://example.org/mybundle/featured-sunset.jpg">
<meta itemprop="datePublished" content="2021-02-26T18:02:00-01:00" />
<meta itemprop="dateModified" content="2021-05-22T19:25:00-01:00" />

`)
	b.AssertFileContent("public/mypage/index.html", `
<meta name="twitter:image" content="https://example.org/pageimg1.jpg"/>
<meta property="og:image" content="https://example.org/pageimg1.jpg" />
<meta property="og:image" content="https://example.org/pageimg2.jpg" />
<meta property="article:published_time" content="2021-02-26T18:02:00+01:00" />
<meta property="article:modified_time" content="2021-05-22T19:25:00+01:00" />
<meta itemprop="image" content="https://example.org/pageimg1.jpg">
<meta itemprop="image" content="https://example.org/pageimg2.jpg">
<meta itemprop="datePublished" content="2021-02-26T18:02:00+01:00" />
<meta itemprop="dateModified" content="2021-05-22T19:25:00+01:00" />
`)
	b.AssertFileContent("public/mysite/index.html", `
<meta name="twitter:image" content="https://example.org/siteimg1.jpg"/>
<meta property="og:image" content="https://example.org/siteimg1.jpg"/>
<meta itemprop="image" content="https://example.org/siteimg1.jpg"/>
`)
}

// Just some simple test of the embedded templates to avoid
// https://github.com/gohugoio/hugo/issues/4757 and similar.
func TestEmbeddedTemplates(t *testing.T) {
	t.Parallel()

	c := qt.New(t)
	c.Assert(true, qt.Equals, true)

	home := []string{"index.html", `
GA:
{{ template "_internal/google_analytics.html" . }}

GA async:

{{ template "_internal/google_analytics_async.html" . }}

Disqus:

{{ template "_internal/disqus.html" . }}

`}

	b := newTestSitesBuilder(t)
	b.WithSimpleConfigFile().WithTemplatesAdded(home...)

	b.Build(BuildCfg{})

	// Gheck GA regular and async
	b.AssertFileContent("public/index.html",
		"'anonymizeIp', true",
		"'script','https://www.google-analytics.com/analytics.js','ga');\n\tga('create', 'UA-ga_id', 'auto')",
		"<script async src='https://www.google-analytics.com/analytics.js'>")

	// Disqus
	b.AssertFileContent("public/index.html", "\"disqus_shortname\" + '.disqus.com/embed.js';")
}

func TestEmbeddedPaginationTemplate(t *testing.T) {
	t.Parallel()

	test := func(variant string, expectedOutput string) {
		b := newTestSitesBuilder(t)
		b.WithConfigFile("toml", `paginate = 1`)
		b.WithContent(
			"s1/p01.md", "---\ntitle: p01\n---",
			"s1/p02.md", "---\ntitle: p02\n---",
			"s1/p03.md", "---\ntitle: p03\n---",
			"s1/p04.md", "---\ntitle: p04\n---",
			"s1/p05.md", "---\ntitle: p05\n---",
			"s1/p06.md", "---\ntitle: p06\n---",
			"s1/p07.md", "---\ntitle: p07\n---",
			"s1/p08.md", "---\ntitle: p08\n---",
			"s1/p09.md", "---\ntitle: p09\n---",
			"s1/p10.md", "---\ntitle: p10\n---",
		)
		b.WithTemplates("index.html", `{{ .Paginate (where site.RegularPages "Section" "s1") }}`+variant)
		b.Build(BuildCfg{})
		b.AssertFileContent("public/index.html", expectedOutput)
	}

	expectedOutputDefaultFormat := "Pager 1\n    <ul class=\"pagination pagination-default\">\n      <li class=\"page-item disabled\">\n        <a aria-disabled=\"true\" aria-label=\"First\" class=\"page-link\" role=\"button\" tabindex=\"-1\"><span aria-hidden=\"true\">&laquo;&laquo;</span></a>\n      </li>\n      <li class=\"page-item disabled\">\n        <a aria-disabled=\"true\" aria-label=\"Previous\" class=\"page-link\" role=\"button\" tabindex=\"-1\"><span aria-hidden=\"true\">&laquo;</span></a>\n      </li>\n      <li class=\"page-item active\">\n        <a aria-current=\"page\" aria-label=\"Page 1\" class=\"page-link\" role=\"button\">1</a>\n      </li>\n      <li class=\"page-item\">\n        <a href=\"/page/2/\" aria-label=\"Page 2\" class=\"page-link\" role=\"button\">2</a>\n      </li>\n      <li class=\"page-item\">\n        <a href=\"/page/3/\" aria-label=\"Page 3\" class=\"page-link\" role=\"button\">3</a>\n      </li>\n      <li class=\"page-item\">\n        <a href=\"/page/4/\" aria-label=\"Page 4\" class=\"page-link\" role=\"button\">4</a>\n      </li>\n      <li class=\"page-item\">\n        <a href=\"/page/5/\" aria-label=\"Page 5\" class=\"page-link\" role=\"button\">5</a>\n      </li>\n      <li class=\"page-item\">\n        <a href=\"/page/2/\" aria-label=\"Next\" class=\"page-link\" role=\"button\"><span aria-hidden=\"true\">&raquo;</span></a>\n      </li>\n      <li class=\"page-item\">\n        <a href=\"/page/10/\" aria-label=\"Last\" class=\"page-link\" role=\"button\"><span aria-hidden=\"true\">&raquo;&raquo;</span></a>\n      </li>\n    </ul>"
	expectedOutputTerseFormat := "Pager 1\n    <ul class=\"pagination pagination-terse\">\n      <li class=\"page-item active\">\n        <a aria-current=\"page\" aria-label=\"Page 1\" class=\"page-link\" role=\"button\">1</a>\n      </li>\n      <li class=\"page-item\">\n        <a href=\"/page/2/\" aria-label=\"Page 2\" class=\"page-link\" role=\"button\">2</a>\n      </li>\n      <li class=\"page-item\">\n        <a href=\"/page/3/\" aria-label=\"Page 3\" class=\"page-link\" role=\"button\">3</a>\n      </li>\n      <li class=\"page-item\">\n        <a href=\"/page/2/\" aria-label=\"Next\" class=\"page-link\" role=\"button\"><span aria-hidden=\"true\">&raquo;</span></a>\n      </li>\n      <li class=\"page-item\">\n        <a href=\"/page/10/\" aria-label=\"Last\" class=\"page-link\" role=\"button\"><span aria-hidden=\"true\">&raquo;&raquo;</span></a>\n      </li>\n    </ul>"

	variant := `{{ template "_internal/pagination.html" . }}`
	test(variant, expectedOutputDefaultFormat)

	variant = `{{ template "_internal/pagination.html" (dict "page" .) }}`
	test(variant, expectedOutputDefaultFormat)

	variant = `{{ template "_internal/pagination.html" (dict "page" . "format" "default") }}`
	test(variant, expectedOutputDefaultFormat)

	variant = `{{ template "_internal/pagination.html" (dict "page" . "format" "terse") }}`
	test(variant, expectedOutputTerseFormat)
}
