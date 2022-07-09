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

package pageparser

import (
	"fmt"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
)

type lexerTest struct {
	name  string
	input string
	items []typeText
}

type typeText struct {
	typ  ItemType
	text string
}

func nti(tp ItemType, val string) typeText {
	return typeText{typ: tp, text: val}
}

var (
	tstJSON                = `{ "a": { "b": "\"Hugo\"}" } }`
	tstFrontMatterTOML     = nti(TypeFrontMatterTOML, "foo = \"bar\"\n")
	tstFrontMatterYAML     = nti(TypeFrontMatterYAML, "foo: \"bar\"\n")
	tstFrontMatterYAMLCRLF = nti(TypeFrontMatterYAML, "foo: \"bar\"\r\n")
	tstFrontMatterJSON     = nti(TypeFrontMatterJSON, tstJSON+"\r\n")
	tstSomeText            = nti(tText, "\nSome text.\n")
	tstSummaryDivider      = nti(TypeLeadSummaryDivider, "<!--more-->\n")
	tstNewline             = nti(tText, "\n")

	tstORG = `
#+TITLE: T1
#+AUTHOR: A1
#+DESCRIPTION: D1
`
	tstFrontMatterORG = nti(TypeFrontMatterORG, tstORG)
)

var crLfReplacer = strings.NewReplacer("\r", "#", "\n", "$")

// TODO(bep) a way to toggle ORG mode vs the rest.
var frontMatterTests = []lexerTest{
	{"empty", "", []typeText{tstEOF}},
	{"Byte order mark", "\ufeff\nSome text.\n", []typeText{nti(TypeIgnore, "\ufeff"), tstSomeText, tstEOF}},
	{"HTML Document", `  <html>  `, []typeText{nti(tError, "plain HTML documents not supported")}},
	{"HTML Document with shortcode", `<html>{{< sc1 >}}</html>`, []typeText{nti(tError, "plain HTML documents not supported")}},
	{"No front matter", "\nSome text.\n", []typeText{tstSomeText, tstEOF}},
	{"YAML front matter", "---\nfoo: \"bar\"\n---\n\nSome text.\n", []typeText{tstFrontMatterYAML, tstSomeText, tstEOF}},
	{"YAML empty front matter", "---\n---\n\nSome text.\n", []typeText{nti(TypeFrontMatterYAML, ""), tstSomeText, tstEOF}},
	{"YAML commented out front matter", "<!--\n---\nfoo: \"bar\"\n---\n-->\nSome text.\n", []typeText{nti(TypeIgnore, "<!--\n"), tstFrontMatterYAML, nti(TypeIgnore, "-->"), tstSomeText, tstEOF}},
	{"YAML commented out front matter, no end", "<!--\n---\nfoo: \"bar\"\n---\nSome text.\n", []typeText{nti(TypeIgnore, "<!--\n"), tstFrontMatterYAML, nti(tError, "starting HTML comment with no end")}},
	// Note that we keep all bytes as they are, but we need to handle CRLF
	{"YAML front matter CRLF", "---\r\nfoo: \"bar\"\r\n---\n\nSome text.\n", []typeText{tstFrontMatterYAMLCRLF, tstSomeText, tstEOF}},
	{"TOML front matter", "+++\nfoo = \"bar\"\n+++\n\nSome text.\n", []typeText{tstFrontMatterTOML, tstSomeText, tstEOF}},
	{"JSON front matter", tstJSON + "\r\n\nSome text.\n", []typeText{tstFrontMatterJSON, tstSomeText, tstEOF}},
	{"ORG front matter", tstORG + "\nSome text.\n", []typeText{tstFrontMatterORG, tstSomeText, tstEOF}},
	{"Summary divider ORG", tstORG + "\nSome text.\n# more\nSome text.\n", []typeText{tstFrontMatterORG, tstSomeText, nti(TypeLeadSummaryDivider, "# more\n"), nti(tText, "Some text.\n"), tstEOF}},
	{"Summary divider", "+++\nfoo = \"bar\"\n+++\n\nSome text.\n<!--more-->\nSome text.\n", []typeText{tstFrontMatterTOML, tstSomeText, tstSummaryDivider, nti(tText, "Some text.\n"), tstEOF}},
	{"Summary divider same line", "+++\nfoo = \"bar\"\n+++\n\nSome text.<!--more-->Some text.\n", []typeText{tstFrontMatterTOML, nti(tText, "\nSome text."), nti(TypeLeadSummaryDivider, "<!--more-->"), nti(tText, "Some text.\n"), tstEOF}},
	// https://github.com/gohugoio/hugo/issues/5402
	{"Summary and shortcode, no space", "+++\nfoo = \"bar\"\n+++\n\nSome text.\n<!--more-->{{< sc1 >}}\nSome text.\n", []typeText{tstFrontMatterTOML, tstSomeText, nti(TypeLeadSummaryDivider, "<!--more-->"), tstLeftNoMD, tstSC1, tstRightNoMD, tstSomeText, tstEOF}},
	// https://github.com/gohugoio/hugo/issues/5464
	{"Summary and shortcode only", "+++\nfoo = \"bar\"\n+++\n{{< sc1 >}}\n<!--more-->\n{{< sc2 >}}", []typeText{tstFrontMatterTOML, tstLeftNoMD, tstSC1, tstRightNoMD, tstNewline, tstSummaryDivider, tstLeftNoMD, tstSC2, tstRightNoMD, tstEOF}},
}

func TestFrontMatter(t *testing.T) {
	t.Parallel()
	c := qt.New(t)
	for i, test := range frontMatterTests {
		items := collect([]byte(test.input), false, lexIntroSection)
		if !equal(test.input, items, test.items) {
			got := itemsToString(items, []byte(test.input))
			expected := testItemsToString(test.items)
			c.Assert(got, qt.Equals, expected, qt.Commentf("Test %d: %s", i, test.name))
		}
	}
}

func itemsToString(items []Item, source []byte) string {
	var sb strings.Builder
	for i, item := range items {
		var s string
		if item.Err != nil {
			s = item.Err.Error()
		} else {
			s = string(item.Val(source))
		}
		sb.WriteString(fmt.Sprintf("%s: %s\n", item.Type, s))

		if i < len(items)-1 {
			sb.WriteString("\n")
		}
	}
	return crLfReplacer.Replace(sb.String())
}

func testItemsToString(items []typeText) string {
	var sb strings.Builder
	for i, item := range items {
		sb.WriteString(fmt.Sprintf("%s: %s\n", item.typ, item.text))

		if i < len(items)-1 {
			sb.WriteString("\n")
		}
	}
	return crLfReplacer.Replace(sb.String())
}

func collectWithConfig(input []byte, skipFrontMatter bool, stateStart stateFunc, cfg Config) (items []Item) {
	l := newPageLexer(input, stateStart, cfg)
	l.run()
	iter := NewIterator(l.items)

	for {
		item := iter.Next()
		items = append(items, item)
		if item.Type == tEOF || item.Type == tError {
			break
		}
	}
	return
}

func collect(input []byte, skipFrontMatter bool, stateStart stateFunc) (items []Item) {
	var cfg Config

	return collectWithConfig(input, skipFrontMatter, stateStart, cfg)
}

func collectStringMain(input string) []Item {
	return collect([]byte(input), true, lexMainSection)
}

// no positional checking, for now ...
func equal(source string, got []Item, expect []typeText) bool {
	if len(got) != len(expect) {
		return false
	}
	sourceb := []byte(source)
	for k := range got {
		g := got[k]
		e := expect[k]
		if g.Type != e.typ {
			return false
		}

		var s string
		if g.Err != nil {
			s = g.Err.Error()
		} else {
			s = string(g.Val(sourceb))
		}

		if s != e.text {
			return false
		}

	}
	return true
}
