// Copyright 2019 The Hugo Authors. All rights reserved.
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
	"strings"
	"testing"

	"github.com/gohugoio/hugo/resources/page"

	"github.com/spf13/afero"

	"github.com/stretchr/testify/require"

	"fmt"

	"github.com/gohugoio/hugo/helpers"
	"github.com/gohugoio/hugo/output"
	"github.com/spf13/viper"
)

func TestSiteWithPageOutputs(t *testing.T) {
	for _, outputs := range [][]string{{"html", "json", "calendar"}, {"json"}} {
		outputs := outputs
		t.Run(fmt.Sprintf("%v", outputs), func(t *testing.T) {
			t.Parallel()
			doTestSiteWithPageOutputs(t, outputs)
		})
	}
}

func doTestSiteWithPageOutputs(t *testing.T, outputs []string) {

	outputsStr := strings.Replace(fmt.Sprintf("%q", outputs), " ", ", ", -1)

	siteConfig := `
baseURL = "http://example.com/blog"

paginate = 1
defaultContentLanguage = "en"

disableKinds = ["section", "taxonomy", "taxonomyTerm", "RSS", "sitemap", "robotsTXT", "404"]

[Taxonomies]
tag = "tags"
category = "categories"

defaultContentLanguage = "en"


[languages]

[languages.en]
title = "Title in English"
languageName = "English"
weight = 1

[languages.nn]
languageName = "Nynorsk"
weight = 2
title = "Tittel på Nynorsk"

`

	pageTemplate := `---
title: "%s"
outputs: %s
---
# Doc

{{< myShort >}}

{{< myOtherShort >}}

`

	b := newTestSitesBuilder(t).WithConfigFile("toml", siteConfig)
	b.WithI18n("en.toml", `
[elbow]
other = "Elbow"
`, "nn.toml", `
[elbow]
other = "Olboge"
`)

	b.WithTemplates(
		// Case issue partials #3333
		"layouts/partials/GoHugo.html", `Go Hugo Partial`,
		"layouts/_default/baseof.json", `START JSON:{{block "main" .}}default content{{ end }}:END JSON`,
		"layouts/_default/baseof.html", `START HTML:{{block "main" .}}default content{{ end }}:END HTML`,
		"layouts/shortcodes/myOtherShort.html", `OtherShort: {{ "<h1>Hi!</h1>" | safeHTML }}`,
		"layouts/shortcodes/myShort.html", `ShortHTML`,
		"layouts/shortcodes/myShort.json", `ShortJSON`,

		"layouts/_default/list.json", `{{ define "main" }}
List JSON|{{ .Title }}|{{ .Content }}|Alt formats: {{ len .AlternativeOutputFormats -}}|
{{- range .AlternativeOutputFormats -}}
Alt Output: {{ .Name -}}|
{{- end -}}|
{{- range .OutputFormats -}}
Output/Rel: {{ .Name -}}/{{ .Rel }}|{{ .MediaType }}
{{- end -}}
 {{ with .OutputFormats.Get "JSON" }}
<atom:link href={{ .Permalink }} rel="self" type="{{ .MediaType }}" />
{{ end }}
{{ .Site.Language.Lang }}: {{ T "elbow" -}}
{{ end }}
`,
		"layouts/_default/list.html", `{{ define "main" }}
List HTML|{{.Title }}|
{{- with .OutputFormats.Get "HTML" -}}
<atom:link href={{ .Permalink }} rel="self" type="{{ .MediaType }}" />
{{- end -}}
{{ .Site.Language.Lang }}: {{ T "elbow" -}}
Partial Hugo 1: {{ partial "GoHugo.html" . }}
Partial Hugo 2: {{ partial "GoHugo" . -}}
Content: {{ .Content }}
Len Pages: {{ .Kind }} {{ len .Site.RegularPages }} Page Number: {{ .Paginator.PageNumber }}
{{ end }}
`,
		"layouts/_default/single.html", `{{ define "main" }}{{ .Content }}{{ end }}`,
	)

	b.WithContent("_index.md", fmt.Sprintf(pageTemplate, "JSON Home", outputsStr))
	b.WithContent("_index.nn.md", fmt.Sprintf(pageTemplate, "JSON Nynorsk Heim", outputsStr))

	for i := 1; i <= 10; i++ {
		b.WithContent(fmt.Sprintf("p%d.md", i), fmt.Sprintf(pageTemplate, fmt.Sprintf("Page %d", i), outputsStr))
	}

	b.Build(BuildCfg{})

	s := b.H.Sites[0]
	require.Equal(t, "en", s.language.Lang)

	home := s.getPage(page.KindHome)

	require.NotNil(t, home)

	lenOut := len(outputs)

	require.Len(t, home.OutputFormats(), lenOut)

	// There is currently always a JSON output to make it simpler ...
	altFormats := lenOut - 1
	hasHTML := helpers.InStringArray(outputs, "html")
	b.AssertFileContent("public/index.json",
		"List JSON",
		fmt.Sprintf("Alt formats: %d", altFormats),
	)

	if hasHTML {
		b.AssertFileContent("public/index.json",
			"Alt Output: HTML",
			"Output/Rel: JSON/alternate|",
			"Output/Rel: HTML/canonical|",
			"en: Elbow",
			"ShortJSON",
			"OtherShort: <h1>Hi!</h1>",
		)

		b.AssertFileContent("public/index.html",
			// The HTML entity is a deliberate part of this test: The HTML templates are
			// parsed with html/template.
			`List HTML|JSON Home|<atom:link href=http://example.com/blog/ rel="self" type="text/html" />`,
			"en: Elbow",
			"ShortHTML",
			"OtherShort: <h1>Hi!</h1>",
			"Len Pages: home 10",
		)
		assert := require.New(t)
		b.AssertFileContent("public/page/2/index.html", "Page Number: 2")
		assert.False(b.CheckExists("public/page/2/index.json"))

		b.AssertFileContent("public/nn/index.html",
			"List HTML|JSON Nynorsk Heim|",
			"nn: Olboge")
	} else {
		b.AssertFileContent("public/index.json",
			"Output/Rel: JSON/canonical|",
			// JSON is plain text, so no need to safeHTML this and that
			`<atom:link href=http://example.com/blog/index.json rel="self" type="application/json" />`,
			"ShortJSON",
			"OtherShort: <h1>Hi!</h1>",
		)
		b.AssertFileContent("public/nn/index.json",
			"List JSON|JSON Nynorsk Heim|",
			"nn: Olboge",
			"ShortJSON",
		)
	}

	of := home.OutputFormats()

	json := of.Get("JSON")
	require.NotNil(t, json)
	require.Equal(t, "/blog/index.json", json.RelPermalink())
	require.Equal(t, "http://example.com/blog/index.json", json.Permalink())

	if helpers.InStringArray(outputs, "cal") {
		cal := of.Get("calendar")
		require.NotNil(t, cal)
		require.Equal(t, "/blog/index.ics", cal.RelPermalink())
		require.Equal(t, "webcal://example.com/blog/index.ics", cal.Permalink())
	}

	require.True(t, home.HasShortcode("myShort"))
	require.False(t, home.HasShortcode("doesNotExist"))

}

// Issue #3447
func TestRedefineRSSOutputFormat(t *testing.T) {
	siteConfig := `
baseURL = "http://example.com/blog"

paginate = 1
defaultContentLanguage = "en"

disableKinds = ["page", "section", "taxonomy", "taxonomyTerm", "sitemap", "robotsTXT", "404"]

[outputFormats]
[outputFormats.RSS]
mediatype = "application/rss"
baseName = "feed"

`

	mf := afero.NewMemMapFs()
	writeToFs(t, mf, "content/foo.html", `foo`)

	th, h := newTestSitesFromConfig(t, mf, siteConfig)

	err := h.Build(BuildCfg{})

	require.NoError(t, err)

	th.assertFileContent("public/feed.xml", "Recent content on")

	s := h.Sites[0]

	//Issue #3450
	require.Equal(t, "http://example.com/blog/feed.xml", s.Info.RSSLink)

}

// Issue #3614
func TestDotLessOutputFormat(t *testing.T) {
	siteConfig := `
baseURL = "http://example.com/blog"

paginate = 1
defaultContentLanguage = "en"

disableKinds = ["page", "section", "taxonomy", "taxonomyTerm", "sitemap", "robotsTXT", "404"]

[mediaTypes]
[mediaTypes."text/nodot"]
delimiter = ""
[mediaTypes."text/defaultdelim"]
suffixes = ["defd"]
[mediaTypes."text/nosuffix"]
[mediaTypes."text/customdelim"]
suffixes = ["del"]
delimiter = "_"

[outputs]
home = [ "DOTLESS", "DEF", "NOS", "CUS" ]

[outputFormats]
[outputFormats.DOTLESS]
mediatype = "text/nodot"
baseName = "_redirects" # This is how Netlify names their redirect files.
[outputFormats.DEF]
mediatype = "text/defaultdelim"
baseName = "defaultdelimbase"
[outputFormats.NOS]
mediatype = "text/nosuffix"
baseName = "nosuffixbase"
[outputFormats.CUS]
mediatype = "text/customdelim"
baseName = "customdelimbase"

`

	mf := afero.NewMemMapFs()
	writeToFs(t, mf, "content/foo.html", `foo`)
	writeToFs(t, mf, "layouts/_default/list.dotless", `a dotless`)
	writeToFs(t, mf, "layouts/_default/list.def.defd", `default delimim`)
	writeToFs(t, mf, "layouts/_default/list.nos", `no suffix`)
	writeToFs(t, mf, "layouts/_default/list.cus.del", `custom delim`)

	th, h := newTestSitesFromConfig(t, mf, siteConfig)

	err := h.Build(BuildCfg{})

	require.NoError(t, err)

	th.assertFileContent("public/_redirects", "a dotless")
	th.assertFileContent("public/defaultdelimbase.defd", "default delimim")
	// This looks weird, but the user has chosen this definition.
	th.assertFileContent("public/nosuffixbase", "no suffix")
	th.assertFileContent("public/customdelimbase_del", "custom delim")

	s := h.Sites[0]
	home := s.getPage(page.KindHome)
	require.NotNil(t, home)

	outputs := home.OutputFormats()

	require.Equal(t, "/blog/_redirects", outputs.Get("DOTLESS").RelPermalink())
	require.Equal(t, "/blog/defaultdelimbase.defd", outputs.Get("DEF").RelPermalink())
	require.Equal(t, "/blog/nosuffixbase", outputs.Get("NOS").RelPermalink())
	require.Equal(t, "/blog/customdelimbase_del", outputs.Get("CUS").RelPermalink())

}

func TestCreateSiteOutputFormats(t *testing.T) {
	assert := require.New(t)

	outputsConfig := map[string]interface{}{
		page.KindHome:    []string{"HTML", "JSON"},
		page.KindSection: []string{"JSON"},
	}

	cfg := viper.New()
	cfg.Set("outputs", outputsConfig)

	outputs, err := createSiteOutputFormats(output.DefaultFormats, cfg)
	assert.NoError(err)
	assert.Equal(output.Formats{output.JSONFormat}, outputs[page.KindSection])
	assert.Equal(output.Formats{output.HTMLFormat, output.JSONFormat}, outputs[page.KindHome])

	// Defaults
	assert.Equal(output.Formats{output.HTMLFormat, output.RSSFormat}, outputs[page.KindTaxonomy])
	assert.Equal(output.Formats{output.HTMLFormat, output.RSSFormat}, outputs[page.KindTaxonomyTerm])
	assert.Equal(output.Formats{output.HTMLFormat}, outputs[page.KindPage])

	// These aren't (currently) in use when rendering in Hugo,
	// but the pages needs to be assigned an output format,
	// so these should also be correct/sensible.
	assert.Equal(output.Formats{output.RSSFormat}, outputs[kindRSS])
	assert.Equal(output.Formats{output.SitemapFormat}, outputs[kindSitemap])
	assert.Equal(output.Formats{output.RobotsTxtFormat}, outputs[kindRobotsTXT])
	assert.Equal(output.Formats{output.HTMLFormat}, outputs[kind404])

}

func TestCreateSiteOutputFormatsInvalidConfig(t *testing.T) {
	assert := require.New(t)

	outputsConfig := map[string]interface{}{
		page.KindHome: []string{"FOO", "JSON"},
	}

	cfg := viper.New()
	cfg.Set("outputs", outputsConfig)

	_, err := createSiteOutputFormats(output.DefaultFormats, cfg)
	assert.Error(err)
}

func TestCreateSiteOutputFormatsEmptyConfig(t *testing.T) {
	assert := require.New(t)

	outputsConfig := map[string]interface{}{
		page.KindHome: []string{},
	}

	cfg := viper.New()
	cfg.Set("outputs", outputsConfig)

	outputs, err := createSiteOutputFormats(output.DefaultFormats, cfg)
	assert.NoError(err)
	assert.Equal(output.Formats{output.HTMLFormat, output.RSSFormat}, outputs[page.KindHome])
}

func TestCreateSiteOutputFormatsCustomFormats(t *testing.T) {
	assert := require.New(t)

	outputsConfig := map[string]interface{}{
		page.KindHome: []string{},
	}

	cfg := viper.New()
	cfg.Set("outputs", outputsConfig)

	var (
		customRSS  = output.Format{Name: "RSS", BaseName: "customRSS"}
		customHTML = output.Format{Name: "HTML", BaseName: "customHTML"}
	)

	outputs, err := createSiteOutputFormats(output.Formats{customRSS, customHTML}, cfg)
	assert.NoError(err)
	assert.Equal(output.Formats{customHTML, customRSS}, outputs[page.KindHome])
}

// https://github.com/gohugoio/hugo/issues/5849
func TestOutputFormatPermalinkable(t *testing.T) {

	config := `
baseURL = "https://example.com"



# DAMP is similar to AMP, but not permalinkable.
[outputFormats]
[outputFormats.damp]
mediaType = "text/html"
path = "damp"
[outputFormats.ramp]
mediaType = "text/html"
path = "ramp"
permalinkable = true
[outputFormats.base]
mediaType = "text/html"
isHTML = true
baseName = "that"
permalinkable = true
[outputFormats.nobase]
mediaType = "application/json"
permalinkable = true

`

	b := newTestSitesBuilder(t).WithConfigFile("toml", config)
	b.WithContent("_index.md", `
---
Title: Home Sweet Home
outputs: [ "html", "amp", "damp", "base" ]
---

`)

	b.WithContent("blog/html-amp.md", `
---
Title: AMP and HTML
outputs: [ "html", "amp" ]
---

`)

	b.WithContent("blog/html-damp.md", `
---
Title: DAMP and HTML
outputs: [ "html", "damp" ]
---

`)

	b.WithContent("blog/html-ramp.md", `
---
Title: RAMP and HTML
outputs: [ "html", "ramp" ]
---

`)

	b.WithContent("blog/html.md", `
---
Title: HTML only
outputs: [ "html" ]
---

`)

	b.WithContent("blog/amp.md", `
---
Title: AMP only
outputs: [ "amp" ]
---

`)

	b.WithContent("blog/html-base-nobase.md", `
---
Title: HTML, Base and Nobase
outputs: [ "html", "base", "nobase" ]
---

`)

	const commonTemplate = `
This RelPermalink: {{ .RelPermalink }}
Output Formats: {{ len .OutputFormats }};{{ range .OutputFormats }}{{ .Name }};{{ .RelPermalink }}|{{ end }}

`

	b.WithTemplatesAdded("index.html", commonTemplate)
	b.WithTemplatesAdded("_default/single.html", commonTemplate)
	b.WithTemplatesAdded("_default/single.json", commonTemplate)

	b.Build(BuildCfg{})

	b.AssertFileContent("public/index.html",
		"This RelPermalink: /",
		"Output Formats: 4;HTML;/|AMP;/amp/|damp;/damp/|base;/that.html|",
	)

	b.AssertFileContent("public/amp/index.html",
		"This RelPermalink: /amp/",
		"Output Formats: 4;HTML;/|AMP;/amp/|damp;/damp/|base;/that.html|",
	)

	b.AssertFileContent("public/blog/html-amp/index.html",
		"Output Formats: 2;HTML;/blog/html-amp/|AMP;/amp/blog/html-amp/|",
		"This RelPermalink: /blog/html-amp/")

	b.AssertFileContent("public/amp/blog/html-amp/index.html",
		"Output Formats: 2;HTML;/blog/html-amp/|AMP;/amp/blog/html-amp/|",
		"This RelPermalink: /amp/blog/html-amp/")

	// Damp is not permalinkable
	b.AssertFileContent("public/damp/blog/html-damp/index.html",
		"This RelPermalink: /blog/html-damp/",
		"Output Formats: 2;HTML;/blog/html-damp/|damp;/damp/blog/html-damp/|")

	b.AssertFileContent("public/blog/html-ramp/index.html",
		"This RelPermalink: /blog/html-ramp/",
		"Output Formats: 2;HTML;/blog/html-ramp/|ramp;/ramp/blog/html-ramp/|")

	b.AssertFileContent("public/ramp/blog/html-ramp/index.html",
		"This RelPermalink: /ramp/blog/html-ramp/",
		"Output Formats: 2;HTML;/blog/html-ramp/|ramp;/ramp/blog/html-ramp/|")

	// https://github.com/gohugoio/hugo/issues/5877
	outputFormats := "Output Formats: 3;HTML;/blog/html-base-nobase/|base;/blog/html-base-nobase/that.html|nobase;/blog/html-base-nobase/index.json|"

	b.AssertFileContent("public/blog/html-base-nobase/index.json",
		"This RelPermalink: /blog/html-base-nobase/index.json",
		outputFormats,
	)

	b.AssertFileContent("public/blog/html-base-nobase/that.html",
		"This RelPermalink: /blog/html-base-nobase/that.html",
		outputFormats,
	)

	b.AssertFileContent("public/blog/html-base-nobase/index.html",
		"This RelPermalink: /blog/html-base-nobase/",
		outputFormats,
	)

}
