// Copyright 2017 The Hugo Authors. All rights reserved.
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

package transform_test

import (
	"html/template"
	"strings"
	"testing"

	"github.com/gohugoio/hugo/common/loggers"
	"github.com/gohugoio/hugo/hugolib"
	"github.com/gohugoio/hugo/tpl/transform"
	"github.com/spf13/afero"

	qt "github.com/frankban/quicktest"
	"github.com/gohugoio/hugo/config"
	"github.com/gohugoio/hugo/deps"
	"github.com/gohugoio/hugo/helpers"
	"github.com/gohugoio/hugo/hugofs"
	"github.com/gohugoio/hugo/langs"
)

type tstNoStringer struct{}

func TestEmojify(t *testing.T) {
	t.Parallel()
	b := hugolib.NewIntegrationTestBuilder(
		hugolib.IntegrationTestConfig{T: t},
	).Build()

	ns := transform.New(b.H.Deps)

	for _, test := range []struct {
		s      interface{}
		expect interface{}
	}{
		{":notamoji:", template.HTML(":notamoji:")},
		{"I :heart: Hugo", template.HTML("I ❤️ Hugo")},
		// errors
		{tstNoStringer{}, false},
	} {

		result, err := ns.Emojify(test.s)

		if bb, ok := test.expect.(bool); ok && !bb {
			b.Assert(err, qt.Not(qt.IsNil))
			continue
		}

		b.Assert(err, qt.IsNil)
		b.Assert(result, qt.Equals, test.expect)
	}
}

func TestHighlight(t *testing.T) {
	t.Parallel()
	b := hugolib.NewIntegrationTestBuilder(
		hugolib.IntegrationTestConfig{T: t},
	).Build()

	ns := transform.New(b.H.Deps)

	for _, test := range []struct {
		s      interface{}
		lang   string
		opts   interface{}
		expect interface{}
	}{
		{"func boo() {}", "go", "", "boo"},
		{"func boo() {}", "go", nil, "boo"},
		// Issue #4179
		{`<Foo attr=" &lt; "></Foo>`, "xml", "", `&amp;lt;`},
		{tstNoStringer{}, "go", "", false},
		// Issue #9591
		{strings.Repeat("AAA	\n", 10), "bash", template.HTML("linenos=true,noClasses=false"), "line"},
	} {

		result, err := ns.Highlight(test.s, test.lang, test.opts)

		if bb, ok := test.expect.(bool); ok && !bb {
			b.Assert(err, qt.Not(qt.IsNil))
			continue
		}

		b.Assert(err, qt.IsNil)
		b.Assert(string(result), qt.Contains, test.expect.(string))
	}
}

func TestCanHighlight(t *testing.T) {
	t.Parallel()

	c := qt.New(t)
	ns := &transform.Namespace{}

	c.Assert(ns.CanHighlight("go"), qt.Equals, true)
	c.Assert(ns.CanHighlight("foo"), qt.Equals, false)
}

func TestHTMLEscape(t *testing.T) {
	t.Parallel()
	b := hugolib.NewIntegrationTestBuilder(
		hugolib.IntegrationTestConfig{T: t},
	).Build()

	ns := transform.New(b.H.Deps)

	for _, test := range []struct {
		s      interface{}
		expect interface{}
	}{
		{`"Foo & Bar's Diner" <y@z>`, `&#34;Foo &amp; Bar&#39;s Diner&#34; &lt;y@z&gt;`},
		{"Hugo & Caddy > Wordpress & Apache", "Hugo &amp; Caddy &gt; Wordpress &amp; Apache"},
		// errors
		{tstNoStringer{}, false},
	} {

		result, err := ns.HTMLEscape(test.s)

		if bb, ok := test.expect.(bool); ok && !bb {
			b.Assert(err, qt.Not(qt.IsNil))
			continue
		}

		b.Assert(err, qt.IsNil)
		b.Assert(result, qt.Equals, test.expect)
	}
}

func TestHTMLUnescape(t *testing.T) {
	t.Parallel()
	b := hugolib.NewIntegrationTestBuilder(
		hugolib.IntegrationTestConfig{T: t},
	).Build()

	ns := transform.New(b.H.Deps)

	for _, test := range []struct {
		s      interface{}
		expect interface{}
	}{
		{`&quot;Foo &amp; Bar&#39;s Diner&quot; &lt;y@z&gt;`, `"Foo & Bar's Diner" <y@z>`},
		{"Hugo &amp; Caddy &gt; Wordpress &amp; Apache", "Hugo & Caddy > Wordpress & Apache"},
		// errors
		{tstNoStringer{}, false},
	} {

		result, err := ns.HTMLUnescape(test.s)

		if bb, ok := test.expect.(bool); ok && !bb {
			b.Assert(err, qt.Not(qt.IsNil))
			continue
		}

		b.Assert(err, qt.IsNil)
		b.Assert(result, qt.Equals, test.expect)
	}
}

func TestMarkdownify(t *testing.T) {
	t.Parallel()
	b := hugolib.NewIntegrationTestBuilder(
		hugolib.IntegrationTestConfig{T: t},
	).Build()

	ns := transform.New(b.H.Deps)

	for _, test := range []struct {
		s      interface{}
		expect interface{}
	}{
		{"Hello **World!**", template.HTML("Hello <strong>World!</strong>")},
		{[]byte("Hello Bytes **World!**"), template.HTML("Hello Bytes <strong>World!</strong>")},
		{tstNoStringer{}, false},
	} {

		result, err := ns.Markdownify(test.s)

		if bb, ok := test.expect.(bool); ok && !bb {
			b.Assert(err, qt.Not(qt.IsNil))
			continue
		}

		b.Assert(err, qt.IsNil)
		b.Assert(result, qt.Equals, test.expect)
	}
}

// Issue #3040
func TestMarkdownifyBlocksOfText(t *testing.T) {
	t.Parallel()
	b := hugolib.NewIntegrationTestBuilder(
		hugolib.IntegrationTestConfig{T: t},
	).Build()

	ns := transform.New(b.H.Deps)

	text := `
#First 

This is some *bold* text.

## Second

This is some more text.

And then some.
`

	result, err := ns.Markdownify(text)
	b.Assert(err, qt.IsNil)
	b.Assert(result, qt.Equals, template.HTML(
		"<p>#First</p>\n<p>This is some <em>bold</em> text.</p>\n<h2 id=\"second\">Second</h2>\n<p>This is some more text.</p>\n<p>And then some.</p>\n"))
}

func TestPlainify(t *testing.T) {
	t.Parallel()
	b := hugolib.NewIntegrationTestBuilder(
		hugolib.IntegrationTestConfig{T: t},
	).Build()

	ns := transform.New(b.H.Deps)

	for _, test := range []struct {
		s      interface{}
		expect interface{}
	}{
		{"<em>Note:</em> blah <b>blah</b>", "Note: blah blah"},
		// errors
		{tstNoStringer{}, false},
	} {

		result, err := ns.Plainify(test.s)

		if bb, ok := test.expect.(bool); ok && !bb {
			b.Assert(err, qt.Not(qt.IsNil))
			continue
		}

		b.Assert(err, qt.IsNil)
		b.Assert(result, qt.Equals, test.expect)
	}
}

func newDeps(cfg config.Provider) *deps.Deps {
	cfg.Set("contentDir", "content")
	cfg.Set("i18nDir", "i18n")

	l := langs.NewLanguage("en", cfg)

	cs, err := helpers.NewContentSpec(l, loggers.NewErrorLogger(), afero.NewMemMapFs(), nil)
	if err != nil {
		panic(err)
	}

	return &deps.Deps{
		Cfg:         cfg,
		Fs:          hugofs.NewMem(l),
		ContentSpec: cs,
	}
}
