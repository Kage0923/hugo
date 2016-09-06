// Copyright 2016 The Hugo Authors. All rights reserved.
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
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bep/inflect"

	"github.com/spf13/hugo/helpers"
	"github.com/spf13/hugo/hugofs"
	"github.com/spf13/hugo/source"
	"github.com/spf13/hugo/target"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

const (
	templateTitle   = "{{ .Title }}"
	pageSimpleTitle = `---
title: simple template
---
content`

	templateMissingFunc = "{{ .Title | funcdoesnotexists }}"
	templateFunc        = "{{ .Title | urlize }}"
	templateContent     = "{{ .Content }}"
	templateDate        = "{{ .Date }}"
	templateWithURLAbs  = "<a href=\"/foobar.jpg\">Going</a>"

	pageWithMd = `---
title: page with md
---
# heading 1
text
## heading 2
more text
`
)

func init() {
	testMode = true
}

// Issue #1797
func TestReadPagesFromSourceWithEmptySource(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	viper.Set("DefaultExtension", "html")
	viper.Set("verbose", true)
	viper.Set("baseurl", "http://auth/bub")

	sources := []source.ByteSource{}

	s := &Site{
		Source:  &source.InMemorySource{ByteSource: sources},
		targets: targetList{page: &target.PagePub{UglyURLs: true}},
	}

	var err error
	d := time.Second * 2
	ticker := time.NewTicker(d)
	select {
	case err = <-s.readPagesFromSource():
		break
	case <-ticker.C:
		err = fmt.Errorf("ReadPagesFromSource() never returns in %s", d.String())
	}
	ticker.Stop()
	if err != nil {
		t.Fatalf("Unable to read source: %s", err)
	}
}

func createAndRenderPages(t *testing.T, s *Site) {
	createPagesAndMeta(t, s)

	if err := s.renderPages(); err != nil {
		t.Fatalf("Unable to render pages. %s", err)
	}
}

func createPagesAndMeta(t *testing.T, s *Site) {
	createPages(t, s)

	s.setupTranslations()
	s.setupPrevNext()

	if err := s.buildSiteMeta(); err != nil {
		t.Fatalf("Unable to build site metadata: %s", err)
	}
}

func createPages(t *testing.T, s *Site) {
	if err := s.createPages(); err != nil {
		t.Fatalf("Unable to create pages: %s", err)
	}
}

func pageMust(p *Page, err error) *Page {
	if err != nil {
		panic(err)
	}
	return p
}

func TestDegenerateRenderThingMissingTemplate(t *testing.T) {
	p, _ := NewPageFrom(strings.NewReader(pageSimpleTitle), "content/a/file.md")
	p.Convert()
	s := new(Site)
	s.prepTemplates()
	err := s.renderThing(p, "foobar", nil)
	if err == nil {
		t.Errorf("Expected err to be returned when missing the template.")
	}
}

func TestAddInvalidTemplate(t *testing.T) {
	s := new(Site)
	err := s.prepTemplates("missing", templateMissingFunc)
	if err == nil {
		t.Fatalf("Expecting the template to return an error")
	}
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

func NopCloser(w io.Writer) io.WriteCloser {
	return nopCloser{w}
}

func TestRenderThing(t *testing.T) {
	tests := []struct {
		content  string
		template string
		expected string
	}{
		{pageSimpleTitle, templateTitle, "simple template"},
		{pageSimpleTitle, templateFunc, "simple-template"},
		{pageWithMd, templateContent, "\n\n<h1 id=\"heading-1\">heading 1</h1>\n\n<p>text</p>\n\n<h2 id=\"heading-2\">heading 2</h2>\n\n<p>more text</p>\n"},
		{simplePageRFC3339Date, templateDate, "2013-05-17 16:59:30 &#43;0000 UTC"},
	}

	for i, test := range tests {

		s := new(Site)

		p, err := NewPageFrom(strings.NewReader(test.content), "content/a/file.md")
		p.Convert()
		if err != nil {
			t.Fatalf("Error parsing buffer: %s", err)
		}
		templateName := fmt.Sprintf("foobar%d", i)

		s.prepTemplates(templateName, test.template)

		if err != nil {
			t.Fatalf("Unable to add template: %s", err)
		}

		p.Content = template.HTML(p.Content)
		html := new(bytes.Buffer)
		err = s.renderThing(p, templateName, NopCloser(html))
		if err != nil {
			t.Errorf("Unable to render html: %s", err)
		}

		if string(html.Bytes()) != test.expected {
			t.Errorf("Content does not match.\nExpected\n\t'%q'\ngot\n\t'%q'", test.expected, html)
		}
	}
}

func HTML(in string) string {
	return in
}

func TestRenderThingOrDefault(t *testing.T) {
	tests := []struct {
		missing  bool
		template string
		expected string
	}{
		{true, templateTitle, HTML("simple template")},
		{true, templateFunc, HTML("simple-template")},
		{false, templateTitle, HTML("simple template")},
		{false, templateFunc, HTML("simple-template")},
	}

	hugofs.InitMemFs()

	for i, test := range tests {

		s := newSiteDefaultLang()

		p, err := NewPageFrom(strings.NewReader(pageSimpleTitle), "content/a/file.md")
		if err != nil {
			t.Fatalf("Error parsing buffer: %s", err)
		}
		templateName := fmt.Sprintf("default%d", i)

		s.prepTemplates(templateName, test.template)

		var err2 error

		if test.missing {
			err2 = s.renderAndWritePage("name", "out", p, "missing", templateName)
		} else {
			err2 = s.renderAndWritePage("name", "out", p, templateName, "missing_default")
		}

		if err2 != nil {
			t.Errorf("Unable to render html: %s", err)
		}

		file, err := hugofs.Destination().Open(filepath.FromSlash("out/index.html"))
		if err != nil {
			t.Errorf("Unable to open html: %s", err)
		}
		if helpers.ReaderToString(file) != test.expected {
			t.Errorf("Content does not match. Expected '%s', got '%s'", test.expected, helpers.ReaderToString(file))
		}
	}
}

func TestDraftAndFutureRender(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	hugofs.InitMemFs()
	sources := []source.ByteSource{
		{Name: filepath.FromSlash("sect/doc1.md"), Content: []byte("---\ntitle: doc1\ndraft: true\npublishdate: \"2414-05-29\"\n---\n# doc1\n*some content*")},
		{Name: filepath.FromSlash("sect/doc2.md"), Content: []byte("---\ntitle: doc2\ndraft: true\npublishdate: \"2012-05-29\"\n---\n# doc2\n*some content*")},
		{Name: filepath.FromSlash("sect/doc3.md"), Content: []byte("---\ntitle: doc3\ndraft: false\npublishdate: \"2414-05-29\"\n---\n# doc3\n*some content*")},
		{Name: filepath.FromSlash("sect/doc4.md"), Content: []byte("---\ntitle: doc4\ndraft: false\npublishdate: \"2012-05-29\"\n---\n# doc4\n*some content*")},
	}

	siteSetup := func() *Site {
		s := &Site{
			Source: &source.InMemorySource{ByteSource: sources},
			Lang:   newDefaultLanguage(),
		}

		s.initializeSiteInfo()

		createPages(t, s)

		return s
	}

	viper.Set("baseurl", "http://auth/bub")

	// Testing Defaults.. Only draft:true and publishDate in the past should be rendered
	s := siteSetup()
	if len(s.AllPages) != 1 {
		t.Fatal("Draft or Future dated content published unexpectedly")
	}

	// only publishDate in the past should be rendered
	viper.Set("BuildDrafts", true)
	s = siteSetup()
	if len(s.AllPages) != 2 {
		t.Fatal("Future Dated Posts published unexpectedly")
	}

	//  drafts should not be rendered, but all dates should
	viper.Set("BuildDrafts", false)
	viper.Set("BuildFuture", true)
	s = siteSetup()
	if len(s.AllPages) != 2 {
		t.Fatal("Draft posts published unexpectedly")
	}

	// all 4 should be included
	viper.Set("BuildDrafts", true)
	viper.Set("BuildFuture", true)
	s = siteSetup()
	if len(s.AllPages) != 4 {
		t.Fatal("Drafts or Future posts not included as expected")
	}

	//setting defaults back
	viper.Set("BuildDrafts", false)
	viper.Set("BuildFuture", false)
}

func TestFutureExpirationRender(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	hugofs.InitMemFs()
	sources := []source.ByteSource{
		{Name: filepath.FromSlash("sect/doc3.md"), Content: []byte("---\ntitle: doc1\nexpirydate: \"2400-05-29\"\n---\n# doc1\n*some content*")},
		{Name: filepath.FromSlash("sect/doc4.md"), Content: []byte("---\ntitle: doc2\nexpirydate: \"2000-05-29\"\n---\n# doc2\n*some content*")},
	}

	siteSetup := func() *Site {
		s := &Site{
			Source: &source.InMemorySource{ByteSource: sources},
			Lang:   newDefaultLanguage(),
		}

		s.initializeSiteInfo()

		createPages(t, s)

		return s
	}

	viper.Set("baseurl", "http://auth/bub")

	s := siteSetup()

	if len(s.AllPages) != 1 {
		if len(s.AllPages) > 1 {
			t.Fatal("Expired content published unexpectedly")
		}

		if len(s.AllPages) < 1 {
			t.Fatal("Valid content expired unexpectedly")
		}
	}

	if s.AllPages[0].Title == "doc2" {
		t.Fatal("Expired content published unexpectedly")
	}
}

// Issue #957
func TestCrossrefs(t *testing.T) {
	hugofs.InitMemFs()
	for _, uglyURLs := range []bool{true, false} {
		for _, relative := range []bool{true, false} {
			doTestCrossrefs(t, relative, uglyURLs)
		}
	}
}

func doTestCrossrefs(t *testing.T, relative, uglyURLs bool) {
	viper.Reset()
	defer viper.Reset()

	baseURL := "http://foo/bar"
	viper.Set("DefaultExtension", "html")
	viper.Set("baseurl", baseURL)
	viper.Set("UglyURLs", uglyURLs)
	viper.Set("verbose", true)

	var refShortcode string
	var expectedBase string
	var expectedURLSuffix string
	var expectedPathSuffix string

	if relative {
		refShortcode = "relref"
		expectedBase = "/bar"
	} else {
		refShortcode = "ref"
		expectedBase = baseURL
	}

	if uglyURLs {
		expectedURLSuffix = ".html"
		expectedPathSuffix = ".html"
	} else {
		expectedURLSuffix = "/"
		expectedPathSuffix = "/index.html"
	}

	sources := []source.ByteSource{
		{
			Name:    filepath.FromSlash("sect/doc1.md"),
			Content: []byte(fmt.Sprintf(`Ref 2: {{< %s "sect/doc2.md" >}}`, refShortcode)),
		},
		// Issue #1148: Make sure that no P-tags is added around shortcodes.
		{
			Name: filepath.FromSlash("sect/doc2.md"),
			Content: []byte(fmt.Sprintf(`**Ref 1:** 

{{< %s "sect/doc1.md" >}}

THE END.`, refShortcode)),
		},
		// Issue #1753: Should not add a trailing newline after shortcode.
		{
			Name:    filepath.FromSlash("sect/doc3.md"),
			Content: []byte(fmt.Sprintf(`**Ref 1:**{{< %s "sect/doc3.md" >}}.`, refShortcode)),
		},
	}

	s := &Site{
		Source:  &source.InMemorySource{ByteSource: sources},
		targets: targetList{page: &target.PagePub{UglyURLs: uglyURLs}},
		Lang:    newDefaultLanguage(),
	}

	s.initializeSiteInfo()

	s.prepTemplates("_default/single.html", "{{.Content}}")

	createAndRenderPages(t, s)

	tests := []struct {
		doc      string
		expected string
	}{
		{filepath.FromSlash(fmt.Sprintf("sect/doc1%s", expectedPathSuffix)), fmt.Sprintf("<p>Ref 2: %s/sect/doc2%s</p>\n", expectedBase, expectedURLSuffix)},
		{filepath.FromSlash(fmt.Sprintf("sect/doc2%s", expectedPathSuffix)), fmt.Sprintf("<p><strong>Ref 1:</strong></p>\n\n%s/sect/doc1%s\n\n<p>THE END.</p>\n", expectedBase, expectedURLSuffix)},
		{filepath.FromSlash(fmt.Sprintf("sect/doc3%s", expectedPathSuffix)), fmt.Sprintf("<p><strong>Ref 1:</strong>%s/sect/doc3%s.</p>\n", expectedBase, expectedURLSuffix)},
	}

	for _, test := range tests {
		file, err := hugofs.Destination().Open(test.doc)

		if err != nil {
			t.Fatalf("Did not find %s in target: %s", test.doc, err)
		}

		content := helpers.ReaderToString(file)

		if content != test.expected {
			t.Errorf("%s content expected:\n%q\ngot:\n%q", test.doc, test.expected, content)
		}
	}

}

// Issue #939
// Issue #1923
func TestShouldAlwaysHaveUglyURLs(t *testing.T) {
	hugofs.InitMemFs()
	for _, uglyURLs := range []bool{true, false} {
		doTestShouldAlwaysHaveUglyURLs(t, uglyURLs)
	}
}

func doTestShouldAlwaysHaveUglyURLs(t *testing.T, uglyURLs bool) {
	viper.Reset()
	defer viper.Reset()

	viper.Set("DefaultExtension", "html")
	viper.Set("verbose", true)
	viper.Set("baseurl", "http://auth/bub")
	viper.Set("DisableSitemap", false)
	viper.Set("DisableRSS", false)
	viper.Set("RSSUri", "index.xml")
	viper.Set("blackfriday",
		map[string]interface{}{
			"plainIDAnchors": true})

	viper.Set("UglyURLs", uglyURLs)

	sources := []source.ByteSource{
		{Name: filepath.FromSlash("sect/doc1.md"), Content: []byte("---\nmarkup: markdown\n---\n# title\nsome *content*")},
		{Name: filepath.FromSlash("sect/doc2.md"), Content: []byte("---\nurl: /ugly.html\nmarkup: markdown\n---\n# title\ndoc2 *content*")},
	}

	s := &Site{
		Source:  &source.InMemorySource{ByteSource: sources},
		targets: targetList{page: &target.PagePub{UglyURLs: uglyURLs}},
		Lang:    newDefaultLanguage(),
	}

	s.initializeSiteInfo()

	s.prepTemplates(
		"index.html", "Home Sweet {{ if.IsHome  }}Home{{ end }}.",
		"_default/single.html", "{{.Content}}{{ if.IsHome  }}This is not home!{{ end }}",
		"404.html", "Page Not Found.{{ if.IsHome  }}This is not home!{{ end }}",
		"rss.xml", "<root>RSS</root>",
		"sitemap.xml", "<root>SITEMAP</root>")

	createAndRenderPages(t, s)
	s.renderHomePage()
	s.renderSitemap()

	var expectedPagePath string
	if uglyURLs {
		expectedPagePath = "sect/doc1.html"
	} else {
		expectedPagePath = "sect/doc1/index.html"
	}

	tests := []struct {
		doc      string
		expected string
	}{
		{filepath.FromSlash("index.html"), "Home Sweet Home."},
		{filepath.FromSlash(expectedPagePath), "\n\n<h1 id=\"title\">title</h1>\n\n<p>some <em>content</em></p>\n"},
		{filepath.FromSlash("404.html"), "Page Not Found."},
		{filepath.FromSlash("index.xml"), "<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\" ?>\n<root>RSS</root>"},
		{filepath.FromSlash("sitemap.xml"), "<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\" ?>\n<root>SITEMAP</root>"},
		// Issue #1923
		{filepath.FromSlash("ugly.html"), "\n\n<h1 id=\"title\">title</h1>\n\n<p>doc2 <em>content</em></p>\n"},
	}

	for _, p := range s.Pages {
		assert.False(t, p.IsHome)
	}

	for _, test := range tests {
		file, err := hugofs.Destination().Open(test.doc)
		if err != nil {
			t.Fatalf("Did not find %s in target: %s", test.doc, err)
		}

		content := helpers.ReaderToString(file)

		if content != test.expected {
			t.Errorf("%s content expected:\n%q\ngot:\n%q", test.doc, test.expected, content)
		}
	}

}

// Issue #1176
func TestSectionNaming(t *testing.T) {

	for _, canonify := range []bool{true, false} {
		for _, uglify := range []bool{true, false} {
			for _, pluralize := range []bool{true, false} {
				doTestSectionNaming(t, canonify, uglify, pluralize)
			}
		}
	}
}

func doTestSectionNaming(t *testing.T, canonify, uglify, pluralize bool) {
	hugofs.InitMemFs()
	viper.Reset()
	defer viper.Reset()
	viper.Set("baseurl", "http://auth/sub/")
	viper.Set("DefaultExtension", "html")
	viper.Set("UglyURLs", uglify)
	viper.Set("PluralizeListTitles", pluralize)
	viper.Set("CanonifyURLs", canonify)

	var expectedPathSuffix string

	if uglify {
		expectedPathSuffix = ".html"
	} else {
		expectedPathSuffix = "/index.html"
	}

	sources := []source.ByteSource{
		{Name: filepath.FromSlash("sect/doc1.html"), Content: []byte("doc1")},
		{Name: filepath.FromSlash("Fish and Chips/doc2.html"), Content: []byte("doc2")},
		{Name: filepath.FromSlash("ラーメン/doc3.html"), Content: []byte("doc3")},
	}

	s := &Site{
		Source:  &source.InMemorySource{ByteSource: sources},
		targets: targetList{page: &target.PagePub{UglyURLs: uglify}},
		Lang:    newDefaultLanguage(),
	}

	s.initializeSiteInfo()
	s.prepTemplates(
		"_default/single.html", "{{.Content}}",
		"_default/list.html", "{{ .Title }}")

	createAndRenderPages(t, s)
	s.renderSectionLists()

	tests := []struct {
		doc         string
		pluralAware bool
		expected    string
	}{
		{filepath.FromSlash(fmt.Sprintf("sect/doc1%s", expectedPathSuffix)), false, "doc1"},
		{filepath.FromSlash(fmt.Sprintf("sect%s", expectedPathSuffix)), true, "Sect"},
		{filepath.FromSlash(fmt.Sprintf("fish-and-chips/doc2%s", expectedPathSuffix)), false, "doc2"},
		{filepath.FromSlash(fmt.Sprintf("fish-and-chips%s", expectedPathSuffix)), true, "Fish and Chips"},
		{filepath.FromSlash(fmt.Sprintf("ラーメン/doc3%s", expectedPathSuffix)), false, "doc3"},
		{filepath.FromSlash(fmt.Sprintf("ラーメン%s", expectedPathSuffix)), true, "ラーメン"},
	}

	for _, test := range tests {
		file, err := hugofs.Destination().Open(test.doc)
		if err != nil {
			t.Fatalf("Did not find %s in target: %s", test.doc, err)
		}

		content := helpers.ReaderToString(file)

		if test.pluralAware && pluralize {
			test.expected = inflect.Pluralize(test.expected)
		}

		if content != test.expected {
			t.Errorf("%s content expected:\n%q\ngot:\n%q", test.doc, test.expected, content)
		}
	}

}
func TestSkipRender(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	hugofs.InitMemFs()
	sources := []source.ByteSource{
		{Name: filepath.FromSlash("sect/doc1.html"), Content: []byte("---\nmarkup: markdown\n---\n# title\nsome *content*")},
		{Name: filepath.FromSlash("sect/doc2.html"), Content: []byte("<!doctype html><html><body>more content</body></html>")},
		{Name: filepath.FromSlash("sect/doc3.md"), Content: []byte("# doc3\n*some* content")},
		{Name: filepath.FromSlash("sect/doc4.md"), Content: []byte("---\ntitle: doc4\n---\n# doc4\n*some content*")},
		{Name: filepath.FromSlash("sect/doc5.html"), Content: []byte("<!doctype html><html>{{ template \"head\" }}<body>body5</body></html>")},
		{Name: filepath.FromSlash("sect/doc6.html"), Content: []byte("<!doctype html><html>{{ template \"head_abs\" }}<body>body5</body></html>")},
		{Name: filepath.FromSlash("doc7.html"), Content: []byte("<html><body>doc7 content</body></html>")},
		{Name: filepath.FromSlash("sect/doc8.html"), Content: []byte("---\nmarkup: md\n---\n# title\nsome *content*")},
	}

	viper.Set("DefaultExtension", "html")
	viper.Set("verbose", true)
	viper.Set("CanonifyURLs", true)
	viper.Set("baseurl", "http://auth/bub")
	s := &Site{
		Source:  &source.InMemorySource{ByteSource: sources},
		targets: targetList{page: &target.PagePub{UglyURLs: true}},
		Lang:    newDefaultLanguage(),
	}

	s.initializeSiteInfo()

	s.prepTemplates(
		"_default/single.html", "{{.Content}}",
		"head", "<head><script src=\"script.js\"></script></head>",
		"head_abs", "<head><script src=\"/script.js\"></script></head>")

	createAndRenderPages(t, s)

	tests := []struct {
		doc      string
		expected string
	}{
		{filepath.FromSlash("sect/doc1.html"), "\n\n<h1 id=\"title\">title</h1>\n\n<p>some <em>content</em></p>\n"},
		{filepath.FromSlash("sect/doc2.html"), "<!doctype html><html><body>more content</body></html>"},
		{filepath.FromSlash("sect/doc3.html"), "\n\n<h1 id=\"doc3\">doc3</h1>\n\n<p><em>some</em> content</p>\n"},
		{filepath.FromSlash("sect/doc4.html"), "\n\n<h1 id=\"doc4\">doc4</h1>\n\n<p><em>some content</em></p>\n"},
		{filepath.FromSlash("sect/doc5.html"), "<!doctype html><html><head><script src=\"script.js\"></script></head><body>body5</body></html>"},
		{filepath.FromSlash("sect/doc6.html"), "<!doctype html><html><head><script src=\"http://auth/bub/script.js\"></script></head><body>body5</body></html>"},
		{filepath.FromSlash("doc7.html"), "<html><body>doc7 content</body></html>"},
		{filepath.FromSlash("sect/doc8.html"), "\n\n<h1 id=\"title\">title</h1>\n\n<p>some <em>content</em></p>\n"},
	}

	for _, test := range tests {
		file, err := hugofs.Destination().Open(test.doc)
		if err != nil {
			t.Fatalf("Did not find %s in target.", test.doc)
		}

		content := helpers.ReaderToString(file)

		if content != test.expected {
			t.Errorf("%s content expected:\n%q\ngot:\n%q", test.doc, test.expected, content)
		}
	}
}

func TestAbsURLify(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	viper.Set("DefaultExtension", "html")

	hugofs.InitMemFs()
	sources := []source.ByteSource{
		{Name: filepath.FromSlash("sect/doc1.html"), Content: []byte("<!doctype html><html><head></head><body><a href=\"#frag1\">link</a></body></html>")},
		{Name: filepath.FromSlash("content/blue/doc2.html"), Content: []byte("---\nf: t\n---\n<!doctype html><html><body>more content</body></html>")},
	}
	for _, baseURL := range []string{"http://auth/bub", "http://base", "//base"} {
		for _, canonify := range []bool{true, false} {
			viper.Set("CanonifyURLs", canonify)
			viper.Set("BaseURL", baseURL)
			s := &Site{
				Source:  &source.InMemorySource{ByteSource: sources},
				targets: targetList{page: &target.PagePub{UglyURLs: true}},
				Lang:    newDefaultLanguage(),
			}
			t.Logf("Rendering with BaseURL %q and CanonifyURLs set %v", viper.GetString("baseURL"), canonify)
			s.initializeSiteInfo()

			s.prepTemplates("blue/single.html", templateWithURLAbs)

			createAndRenderPages(t, s)

			tests := []struct {
				file, expected string
			}{
				{"content/blue/doc2.html", "<a href=\"%s/foobar.jpg\">Going</a>"},
				{"sect/doc1.html", "<!doctype html><html><head></head><body><a href=\"#frag1\">link</a></body></html>"},
			}

			for _, test := range tests {

				file, err := hugofs.Destination().Open(filepath.FromSlash(test.file))
				if err != nil {
					t.Fatalf("Unable to locate rendered content: %s", test.file)
				}

				content := helpers.ReaderToString(file)

				expected := test.expected

				if strings.Contains(expected, "%s") {
					expected = fmt.Sprintf(expected, baseURL)
				}

				if !canonify {
					expected = strings.Replace(expected, baseURL, "", -1)
				}

				if content != expected {
					t.Errorf("AbsURLify with baseURL %q content expected:\n%q\ngot\n%q", baseURL, expected, content)
				}
			}
		}
	}
}

var weightedPage1 = []byte(`+++
weight = "2"
title = "One"
my_param = "foo"
my_date = 1979-05-27T07:32:00Z
+++
Front Matter with Ordered Pages`)

var weightedPage2 = []byte(`+++
weight = "6"
title = "Two"
publishdate = "2012-03-05"
my_param = "foo"
+++
Front Matter with Ordered Pages 2`)

var weightedPage3 = []byte(`+++
weight = "4"
title = "Three"
date = "2012-04-06"
publishdate = "2012-04-06"
my_param = "bar"
only_one = "yes"
my_date = 2010-05-27T07:32:00Z
+++
Front Matter with Ordered Pages 3`)

var weightedPage4 = []byte(`+++
weight = "4"
title = "Four"
date = "2012-01-01"
publishdate = "2012-01-01"
my_param = "baz"
my_date = 2010-05-27T07:32:00Z
+++
Front Matter with Ordered Pages 4. This is longer content`)

var weightedSources = []source.ByteSource{
	{Name: filepath.FromSlash("sect/doc1.md"), Content: weightedPage1},
	{Name: filepath.FromSlash("sect/doc2.md"), Content: weightedPage2},
	{Name: filepath.FromSlash("sect/doc3.md"), Content: weightedPage3},
	{Name: filepath.FromSlash("sect/doc4.md"), Content: weightedPage4},
}

func TestOrderedPages(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	hugofs.InitMemFs()

	viper.Set("baseurl", "http://auth/bub")
	s := &Site{
		Source: &source.InMemorySource{ByteSource: weightedSources},
		Lang:   newDefaultLanguage(),
	}
	s.initializeSiteInfo()

	createPagesAndMeta(t, s)

	if s.Sections["sect"][0].Weight != 2 || s.Sections["sect"][3].Weight != 6 {
		t.Errorf("Pages in unexpected order. First should be '%d', got '%d'", 2, s.Sections["sect"][0].Weight)
	}

	if s.Sections["sect"][1].Page.Title != "Three" || s.Sections["sect"][2].Page.Title != "Four" {
		t.Errorf("Pages in unexpected order. Second should be '%s', got '%s'", "Three", s.Sections["sect"][1].Page.Title)
	}

	bydate := s.Pages.ByDate()

	if bydate[0].Title != "One" {
		t.Errorf("Pages in unexpected order. First should be '%s', got '%s'", "One", bydate[0].Title)
	}

	rev := bydate.Reverse()
	if rev[0].Title != "Three" {
		t.Errorf("Pages in unexpected order. First should be '%s', got '%s'", "Three", rev[0].Title)
	}

	bypubdate := s.Pages.ByPublishDate()

	if bypubdate[0].Title != "One" {
		t.Errorf("Pages in unexpected order. First should be '%s', got '%s'", "One", bypubdate[0].Title)
	}

	rbypubdate := bypubdate.Reverse()
	if rbypubdate[0].Title != "Three" {
		t.Errorf("Pages in unexpected order. First should be '%s', got '%s'", "Three", rbypubdate[0].Title)
	}

	bylength := s.Pages.ByLength()
	if bylength[0].Title != "One" {
		t.Errorf("Pages in unexpected order. First should be '%s', got '%s'", "One", bylength[0].Title)
	}

	rbylength := bylength.Reverse()
	if rbylength[0].Title != "Four" {
		t.Errorf("Pages in unexpected order. First should be '%s', got '%s'", "Four", rbylength[0].Title)
	}
}

var groupedSources = []source.ByteSource{
	{Name: filepath.FromSlash("sect1/doc1.md"), Content: weightedPage1},
	{Name: filepath.FromSlash("sect1/doc2.md"), Content: weightedPage2},
	{Name: filepath.FromSlash("sect2/doc3.md"), Content: weightedPage3},
	{Name: filepath.FromSlash("sect3/doc4.md"), Content: weightedPage4},
}

func TestGroupedPages(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()

	hugofs.InitMemFs()

	viper.Set("baseurl", "http://auth/bub")
	s := &Site{
		Source: &source.InMemorySource{ByteSource: groupedSources},
	}
	s.initializeSiteInfo()

	createPagesAndMeta(t, s)

	rbysection, err := s.Pages.GroupBy("Section", "desc")
	if err != nil {
		t.Fatalf("Unable to make PageGroup array: %s", err)
	}
	if rbysection[0].Key != "sect3" {
		t.Errorf("PageGroup array in unexpected order. First group key should be '%s', got '%s'", "sect3", rbysection[0].Key)
	}
	if rbysection[1].Key != "sect2" {
		t.Errorf("PageGroup array in unexpected order. Second group key should be '%s', got '%s'", "sect2", rbysection[1].Key)
	}
	if rbysection[2].Key != "sect1" {
		t.Errorf("PageGroup array in unexpected order. Third group key should be '%s', got '%s'", "sect1", rbysection[2].Key)
	}
	if rbysection[0].Pages[0].Title != "Four" {
		t.Errorf("PageGroup has an unexpected page. First group's pages should have '%s', got '%s'", "Four", rbysection[0].Pages[0].Title)
	}
	if len(rbysection[2].Pages) != 2 {
		t.Errorf("PageGroup has unexpected number of pages. Third group should have '%d' pages, got '%d' pages", 2, len(rbysection[2].Pages))
	}

	bytype, err := s.Pages.GroupBy("Type", "asc")
	if err != nil {
		t.Fatalf("Unable to make PageGroup array: %s", err)
	}
	if bytype[0].Key != "sect1" {
		t.Errorf("PageGroup array in unexpected order. First group key should be '%s', got '%s'", "sect1", bytype[0].Key)
	}
	if bytype[1].Key != "sect2" {
		t.Errorf("PageGroup array in unexpected order. Second group key should be '%s', got '%s'", "sect2", bytype[1].Key)
	}
	if bytype[2].Key != "sect3" {
		t.Errorf("PageGroup array in unexpected order. Third group key should be '%s', got '%s'", "sect3", bytype[2].Key)
	}
	if bytype[2].Pages[0].Title != "Four" {
		t.Errorf("PageGroup has an unexpected page. Third group's data should have '%s', got '%s'", "Four", bytype[0].Pages[0].Title)
	}
	if len(bytype[0].Pages) != 2 {
		t.Errorf("PageGroup has unexpected number of pages. First group should have '%d' pages, got '%d' pages", 2, len(bytype[2].Pages))
	}

	bydate, err := s.Pages.GroupByDate("2006-01", "asc")
	if err != nil {
		t.Fatalf("Unable to make PageGroup array: %s", err)
	}
	if bydate[0].Key != "0001-01" {
		t.Errorf("PageGroup array in unexpected order. First group key should be '%s', got '%s'", "0001-01", bydate[0].Key)
	}
	if bydate[1].Key != "2012-01" {
		t.Errorf("PageGroup array in unexpected order. Second group key should be '%s', got '%s'", "2012-01", bydate[1].Key)
	}
	if bydate[2].Key != "2012-04" {
		t.Errorf("PageGroup array in unexpected order. Third group key should be '%s', got '%s'", "2012-04", bydate[2].Key)
	}
	if bydate[2].Pages[0].Title != "Three" {
		t.Errorf("PageGroup has an unexpected page. Third group's pages should have '%s', got '%s'", "Three", bydate[2].Pages[0].Title)
	}
	if len(bydate[0].Pages) != 2 {
		t.Errorf("PageGroup has unexpected number of pages. First group should have '%d' pages, got '%d' pages", 2, len(bydate[2].Pages))
	}

	bypubdate, err := s.Pages.GroupByPublishDate("2006")
	if err != nil {
		t.Fatalf("Unable to make PageGroup array: %s", err)
	}
	if bypubdate[0].Key != "2012" {
		t.Errorf("PageGroup array in unexpected order. First group key should be '%s', got '%s'", "2012", bypubdate[0].Key)
	}
	if bypubdate[1].Key != "0001" {
		t.Errorf("PageGroup array in unexpected order. Second group key should be '%s', got '%s'", "0001", bypubdate[1].Key)
	}
	if bypubdate[0].Pages[0].Title != "Three" {
		t.Errorf("PageGroup has an unexpected page. Third group's pages should have '%s', got '%s'", "Three", bypubdate[0].Pages[0].Title)
	}
	if len(bypubdate[0].Pages) != 3 {
		t.Errorf("PageGroup has unexpected number of pages. First group should have '%d' pages, got '%d' pages", 3, len(bypubdate[0].Pages))
	}

	byparam, err := s.Pages.GroupByParam("my_param", "desc")
	if err != nil {
		t.Fatalf("Unable to make PageGroup array: %s", err)
	}
	if byparam[0].Key != "foo" {
		t.Errorf("PageGroup array in unexpected order. First group key should be '%s', got '%s'", "foo", byparam[0].Key)
	}
	if byparam[1].Key != "baz" {
		t.Errorf("PageGroup array in unexpected order. Second group key should be '%s', got '%s'", "baz", byparam[1].Key)
	}
	if byparam[2].Key != "bar" {
		t.Errorf("PageGroup array in unexpected order. Third group key should be '%s', got '%s'", "bar", byparam[2].Key)
	}
	if byparam[2].Pages[0].Title != "Three" {
		t.Errorf("PageGroup has an unexpected page. Third group's pages should have '%s', got '%s'", "Three", byparam[2].Pages[0].Title)
	}
	if len(byparam[0].Pages) != 2 {
		t.Errorf("PageGroup has unexpected number of pages. First group should have '%d' pages, got '%d' pages", 2, len(byparam[0].Pages))
	}

	_, err = s.Pages.GroupByParam("not_exist")
	if err == nil {
		t.Errorf("GroupByParam didn't return an expected error")
	}

	byOnlyOneParam, err := s.Pages.GroupByParam("only_one")
	if err != nil {
		t.Fatalf("Unable to make PageGroup array: %s", err)
	}
	if len(byOnlyOneParam) != 1 {
		t.Errorf("PageGroup array has unexpected elements. Group length should be '%d', got '%d'", 1, len(byOnlyOneParam))
	}
	if byOnlyOneParam[0].Key != "yes" {
		t.Errorf("PageGroup array in unexpected order. First group key should be '%s', got '%s'", "yes", byOnlyOneParam[0].Key)
	}

	byParamDate, err := s.Pages.GroupByParamDate("my_date", "2006-01")
	if err != nil {
		t.Fatalf("Unable to make PageGroup array: %s", err)
	}
	if byParamDate[0].Key != "2010-05" {
		t.Errorf("PageGroup array in unexpected order. First group key should be '%s', got '%s'", "2010-05", byParamDate[0].Key)
	}
	if byParamDate[1].Key != "1979-05" {
		t.Errorf("PageGroup array in unexpected order. Second group key should be '%s', got '%s'", "1979-05", byParamDate[1].Key)
	}
	if byParamDate[1].Pages[0].Title != "One" {
		t.Errorf("PageGroup has an unexpected page. Second group's pages should have '%s', got '%s'", "One", byParamDate[1].Pages[0].Title)
	}
	if len(byParamDate[0].Pages) != 2 {
		t.Errorf("PageGroup has unexpected number of pages. First group should have '%d' pages, got '%d' pages", 2, len(byParamDate[2].Pages))
	}
}

var pageWithWeightedTaxonomies1 = []byte(`+++
tags = [ "a", "b", "c" ]
tags_weight = 22
categories = ["d"]
title = "foo"
categories_weight = 44
+++
Front Matter with weighted tags and categories`)

var pageWithWeightedTaxonomies2 = []byte(`+++
tags = "a"
tags_weight = 33
title = "bar"
categories = [ "d", "e" ]
categories_weight = 11
alias = "spf13"
date = 1979-05-27T07:32:00Z
+++
Front Matter with weighted tags and categories`)

var pageWithWeightedTaxonomies3 = []byte(`+++
title = "bza"
categories = [ "e" ]
categories_weight = 11
alias = "spf13"
date = 2010-05-27T07:32:00Z
+++
Front Matter with weighted tags and categories`)

func TestWeightedTaxonomies(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	hugofs.InitMemFs()
	sources := []source.ByteSource{
		{Name: filepath.FromSlash("sect/doc1.md"), Content: pageWithWeightedTaxonomies2},
		{Name: filepath.FromSlash("sect/doc2.md"), Content: pageWithWeightedTaxonomies1},
		{Name: filepath.FromSlash("sect/doc3.md"), Content: pageWithWeightedTaxonomies3},
	}
	taxonomies := make(map[string]string)

	taxonomies["tag"] = "tags"
	taxonomies["category"] = "categories"

	viper.Set("baseurl", "http://auth/bub")
	viper.Set("taxonomies", taxonomies)
	s := &Site{
		Source: &source.InMemorySource{ByteSource: sources},
		Lang:   newDefaultLanguage(),
	}
	s.initializeSiteInfo()

	createPagesAndMeta(t, s)

	if s.Taxonomies["tags"]["a"][0].Page.Title != "foo" {
		t.Errorf("Pages in unexpected order, 'foo' expected first, got '%v'", s.Taxonomies["tags"]["a"][0].Page.Title)
	}

	if s.Taxonomies["categories"]["d"][0].Page.Title != "bar" {
		t.Errorf("Pages in unexpected order, 'bar' expected first, got '%v'", s.Taxonomies["categories"]["d"][0].Page.Title)
	}

	if s.Taxonomies["categories"]["e"][0].Page.Title != "bza" {
		t.Errorf("Pages in unexpected order, 'bza' expected first, got '%v'", s.Taxonomies["categories"]["e"][0].Page.Title)
	}
}

func findPage(site *Site, f string) *Page {
	currentPath := source.NewFile(filepath.FromSlash(f))
	//t.Logf("looking for currentPath: %s", currentPath.Path())

	for _, page := range site.Pages {
		//t.Logf("page: %s", page.Source.Path())
		if page.Source.Path() == currentPath.Path() {
			return page
		}
	}
	return nil
}

func setupLinkingMockSite(t *testing.T) *Site {
	hugofs.InitMemFs()
	sources := []source.ByteSource{
		{Name: filepath.FromSlash("index.md"), Content: []byte("")},
		{Name: filepath.FromSlash("rootfile.md"), Content: []byte("")},
		{Name: filepath.FromSlash("root-image.png"), Content: []byte("")},

		{Name: filepath.FromSlash("level2/2-root.md"), Content: []byte("")},
		{Name: filepath.FromSlash("level2/index.md"), Content: []byte("")},
		{Name: filepath.FromSlash("level2/common.md"), Content: []byte("")},

		//		{Name: filepath.FromSlash("level2b/2b-root.md"), Content: []byte("")},
		//		{Name: filepath.FromSlash("level2b/index.md"), Content: []byte("")},
		//		{Name: filepath.FromSlash("level2b/common.md"), Content: []byte("")},

		{Name: filepath.FromSlash("level2/2-image.png"), Content: []byte("")},
		{Name: filepath.FromSlash("level2/common.png"), Content: []byte("")},

		{Name: filepath.FromSlash("level2/level3/3-root.md"), Content: []byte("")},
		{Name: filepath.FromSlash("level2/level3/index.md"), Content: []byte("")},
		{Name: filepath.FromSlash("level2/level3/common.md"), Content: []byte("")},
		{Name: filepath.FromSlash("level2/level3/3-image.png"), Content: []byte("")},
		{Name: filepath.FromSlash("level2/level3/common.png"), Content: []byte("")},
	}

	viper.Set("baseurl", "http://auth/")
	viper.Set("DefaultExtension", "html")
	viper.Set("UglyURLs", false)
	viper.Set("PluralizeListTitles", false)
	viper.Set("CanonifyURLs", false)
	viper.Set("blackfriday",
		map[string]interface{}{
			"sourceRelativeLinksProjectFolder": "/docs"})

	site := &Site{
		Source: &source.InMemorySource{ByteSource: sources},
		Lang:   newDefaultLanguage(),
	}

	site.initializeSiteInfo()

	createPagesAndMeta(t, site)

	return site
}

func TestRefLinking(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	site := setupLinkingMockSite(t)

	currentPage := findPage(site, "level2/level3/index.md")
	if currentPage == nil {
		t.Fatalf("failed to find current page in site")
	}

	// refLink doesn't use the location of the current page to work out reflinks
	okresults := map[string]string{
		"index.md":  "/",
		"common.md": "/level2/common/",
		"3-root.md": "/level2/level3/3-root/",
	}
	for link, url := range okresults {
		if out, err := site.Info.refLink(link, currentPage, true); err != nil || out != url {
			t.Errorf("Expected %s to resolve to (%s), got (%s) - error: %s", link, url, out, err)
		}
	}
	// TODO: and then the failure cases.
}

func TestSourceRelativeLinksing(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	site := setupLinkingMockSite(t)

	type resultMap map[string]string

	okresults := map[string]resultMap{
		"index.md": map[string]string{
			"/docs/rootfile.md":             "/rootfile/",
			"/docs/index.md":                "/",
			"rootfile.md":                   "/rootfile/",
			"index.md":                      "/",
			"level2/2-root.md":              "/level2/2-root/",
			"level2/index.md":               "/level2/",
			"/docs/level2/2-root.md":        "/level2/2-root/",
			"/docs/level2/index.md":         "/level2/",
			"level2/level3/3-root.md":       "/level2/level3/3-root/",
			"level2/level3/index.md":        "/level2/level3/",
			"/docs/level2/level3/3-root.md": "/level2/level3/3-root/",
			"/docs/level2/level3/index.md":  "/level2/level3/",
			"/docs/level2/2-root/":          "/level2/2-root/",
			"/docs/level2/":                 "/level2/",
			"/docs/level2/2-root":           "/level2/2-root/",
			"/docs/level2":                  "/level2/",
			"/level2/2-root/":               "/level2/2-root/",
			"/level2/":                      "/level2/",
			"/level2/2-root":                "/level2/2-root/",
			"/level2":                       "/level2/",
		}, "rootfile.md": map[string]string{
			"/docs/rootfile.md":             "/rootfile/",
			"/docs/index.md":                "/",
			"rootfile.md":                   "/rootfile/",
			"index.md":                      "/",
			"level2/2-root.md":              "/level2/2-root/",
			"level2/index.md":               "/level2/",
			"/docs/level2/2-root.md":        "/level2/2-root/",
			"/docs/level2/index.md":         "/level2/",
			"level2/level3/3-root.md":       "/level2/level3/3-root/",
			"level2/level3/index.md":        "/level2/level3/",
			"/docs/level2/level3/3-root.md": "/level2/level3/3-root/",
			"/docs/level2/level3/index.md":  "/level2/level3/",
		}, "level2/2-root.md": map[string]string{
			"../rootfile.md":                "/rootfile/",
			"../index.md":                   "/",
			"/docs/rootfile.md":             "/rootfile/",
			"/docs/index.md":                "/",
			"2-root.md":                     "/level2/2-root/",
			"index.md":                      "/level2/",
			"../level2/2-root.md":           "/level2/2-root/",
			"../level2/index.md":            "/level2/",
			"./2-root.md":                   "/level2/2-root/",
			"./index.md":                    "/level2/",
			"/docs/level2/index.md":         "/level2/",
			"/docs/level2/2-root.md":        "/level2/2-root/",
			"level3/3-root.md":              "/level2/level3/3-root/",
			"level3/index.md":               "/level2/level3/",
			"../level2/level3/index.md":     "/level2/level3/",
			"../level2/level3/3-root.md":    "/level2/level3/3-root/",
			"/docs/level2/level3/index.md":  "/level2/level3/",
			"/docs/level2/level3/3-root.md": "/level2/level3/3-root/",
		}, "level2/index.md": map[string]string{
			"../rootfile.md":                "/rootfile/",
			"../index.md":                   "/",
			"/docs/rootfile.md":             "/rootfile/",
			"/docs/index.md":                "/",
			"2-root.md":                     "/level2/2-root/",
			"index.md":                      "/level2/",
			"../level2/2-root.md":           "/level2/2-root/",
			"../level2/index.md":            "/level2/",
			"./2-root.md":                   "/level2/2-root/",
			"./index.md":                    "/level2/",
			"/docs/level2/index.md":         "/level2/",
			"/docs/level2/2-root.md":        "/level2/2-root/",
			"level3/3-root.md":              "/level2/level3/3-root/",
			"level3/index.md":               "/level2/level3/",
			"../level2/level3/index.md":     "/level2/level3/",
			"../level2/level3/3-root.md":    "/level2/level3/3-root/",
			"/docs/level2/level3/index.md":  "/level2/level3/",
			"/docs/level2/level3/3-root.md": "/level2/level3/3-root/",
		}, "level2/level3/3-root.md": map[string]string{
			"../../rootfile.md":      "/rootfile/",
			"../../index.md":         "/",
			"/docs/rootfile.md":      "/rootfile/",
			"/docs/index.md":         "/",
			"../2-root.md":           "/level2/2-root/",
			"../index.md":            "/level2/",
			"/docs/level2/2-root.md": "/level2/2-root/",
			"/docs/level2/index.md":  "/level2/",
			"3-root.md":              "/level2/level3/3-root/",
			"index.md":               "/level2/level3/",
			"./3-root.md":            "/level2/level3/3-root/",
			"./index.md":             "/level2/level3/",
			//			"../level2/level3/3-root.md":    "/level2/level3/3-root/",
			//			"../level2/level3/index.md":     "/level2/level3/",
			"/docs/level2/level3/3-root.md": "/level2/level3/3-root/",
			"/docs/level2/level3/index.md":  "/level2/level3/",
		}, "level2/level3/index.md": map[string]string{
			"../../rootfile.md":      "/rootfile/",
			"../../index.md":         "/",
			"/docs/rootfile.md":      "/rootfile/",
			"/docs/index.md":         "/",
			"../2-root.md":           "/level2/2-root/",
			"../index.md":            "/level2/",
			"/docs/level2/2-root.md": "/level2/2-root/",
			"/docs/level2/index.md":  "/level2/",
			"3-root.md":              "/level2/level3/3-root/",
			"index.md":               "/level2/level3/",
			"./3-root.md":            "/level2/level3/3-root/",
			"./index.md":             "/level2/level3/",
			//			"../level2/level3/3-root.md":    "/level2/level3/3-root/",
			//			"../level2/level3/index.md":     "/level2/level3/",
			"/docs/level2/level3/3-root.md": "/level2/level3/3-root/",
			"/docs/level2/level3/index.md":  "/level2/level3/",
		},
	}

	for currentFile, results := range okresults {
		currentPage := findPage(site, currentFile)
		if currentPage == nil {
			t.Fatalf("failed to find current page in site")
		}
		for link, url := range results {
			if out, err := site.Info.SourceRelativeLink(link, currentPage); err != nil || out != url {
				t.Errorf("Expected %s to resolve to (%s), got (%s) - error: %s", link, url, out, err)
			} else {
				//t.Logf("tested ok %s maps to %s", link, out)
			}
		}
	}
	// TODO: and then the failure cases.
	// 			"https://docker.com":           "",
	// site_test.go:1094: Expected https://docker.com to resolve to (), got () - error: Not a plain filepath link (https://docker.com)

}

func TestSourceRelativeLinkFileing(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	site := setupLinkingMockSite(t)

	type resultMap map[string]string

	okresults := map[string]resultMap{
		"index.md": map[string]string{
			"/root-image.png": "/root-image.png",
			"root-image.png":  "/root-image.png",
		}, "rootfile.md": map[string]string{
			"/root-image.png": "/root-image.png",
		}, "level2/2-root.md": map[string]string{
			"/root-image.png": "/root-image.png",
			"common.png":      "/level2/common.png",
		}, "level2/index.md": map[string]string{
			"/root-image.png": "/root-image.png",
			"common.png":      "/level2/common.png",
			"./common.png":    "/level2/common.png",
		}, "level2/level3/3-root.md": map[string]string{
			"/root-image.png": "/root-image.png",
			"common.png":      "/level2/level3/common.png",
			"../common.png":   "/level2/common.png",
		}, "level2/level3/index.md": map[string]string{
			"/root-image.png": "/root-image.png",
			"common.png":      "/level2/level3/common.png",
			"../common.png":   "/level2/common.png",
		},
	}

	for currentFile, results := range okresults {
		currentPage := findPage(site, currentFile)
		if currentPage == nil {
			t.Fatalf("failed to find current page in site")
		}
		for link, url := range results {
			if out, err := site.Info.SourceRelativeLinkFile(link, currentPage); err != nil || out != url {
				t.Errorf("Expected %s to resolve to (%s), got (%s) - error: %s", link, url, out, err)
			} else {
				//t.Logf("tested ok %s maps to %s", link, out)
			}
		}
	}
}

func TestMultilingualSwitch(t *testing.T) {
	// General settings
	viper.Set("DefaultExtension", "html")
	viper.Set("baseurl", "http://example.com/blog")
	viper.Set("DisableSitemap", false)
	viper.Set("DisableRSS", false)
	viper.Set("RSSUri", "index.xml")
	viper.Set("Taxonomies", map[string]string{"tag": "tags"})
	viper.Set("Permalinks", map[string]string{"other": "/somewhere/else/:filename"})

	// Sources
	sources := []source.ByteSource{
		{filepath.FromSlash("sect/doc1.en.md"), []byte(`---
title: doc1
slug: doc1-slug
tags:
 - tag1
publishdate: "2000-01-01"
---
# doc1
*some content*
NOTE: slug should be used as URL
`)},
		{filepath.FromSlash("sect/doc1.fr.md"), []byte(`---
title: doc1
tags:
 - tag1
 - tag2
publishdate: "2000-01-04"
---
# doc1
*quelque contenu*
NOTE: should be in the 'en' Page's 'Translations' field.
NOTE: date is after "doc3"
`)},
		{filepath.FromSlash("sect/doc2.en.md"), []byte(`---
title: doc2
publishdate: "2000-01-02"
---
# doc2
*some content*
NOTE: without slug, "doc2" should be used, without ".en" as URL
`)},
		{filepath.FromSlash("sect/doc3.en.md"), []byte(`---
title: doc3
publishdate: "2000-01-03"
tags:
 - tag2
url: /superbob
---
# doc3
*some content*
NOTE: third 'en' doc, should trigger pagination on home page.
`)},
		{filepath.FromSlash("sect/doc4.md"), []byte(`---
title: doc4
tags:
 - tag1
publishdate: "2000-01-05"
---
# doc4
*du contenu francophone*
NOTE: should use the DefaultContentLanguage and mark this doc as 'fr'.
NOTE: doesn't have any corresponding translation in 'en'
`)},
		{filepath.FromSlash("other/doc5.fr.md"), []byte(`---
title: doc5
publishdate: "2000-01-06"
---
# doc5
*autre contenu francophone*
NOTE: should use the "permalinks" configuration with :filename
`)},
	}

	hugofs.InitMemFs()

	// Multilingual settings
	viper.Set("Multilingual", true)
	en := NewLanguage("en")
	viper.Set("DefaultContentLanguage", "fr")
	viper.Set("paginate", "2")

	languages := NewLanguages(en, NewLanguage("fr"))
	s := &Site{
		Source: &source.InMemorySource{ByteSource: sources},
		Lang:   en,
		Multilingual: &Multilingual{
			Languages: languages,
		},
	}

	s.prepTemplates()
	s.initializeSiteInfo()

	createPagesAndMeta(t, s)

	assert.Len(t, s.Source.Files(), 6, "should have 6 source files")
	assert.Len(t, s.Pages, 3, "should have 3 pages")
	assert.Len(t, s.AllPages, 6, "should have 6 total pages (including translations)")

	doc1en := s.Pages[0]
	permalink, err := doc1en.Permalink()
	assert.NoError(t, err, "permalink call failed")
	assert.Equal(t, "http://example.com/blog/en/sect/doc1-slug", permalink, "invalid doc1.en permalink")
	assert.Len(t, doc1en.Translations(), 1, "doc1-en should have one translation, excluding itself")

	doc2 := s.Pages[1]
	permalink, err = doc2.Permalink()
	assert.NoError(t, err, "permalink call failed")
	assert.Equal(t, "http://example.com/blog/en/sect/doc2", permalink, "invalid doc2 permalink")

	doc3 := s.Pages[2]
	permalink, err = doc3.Permalink()
	assert.NoError(t, err, "permalink call failed")
	assert.Equal(t, "http://example.com/blog/superbob", permalink, "invalid doc3 permalink")
	assert.Equal(t, "/superbob", doc3.URL, "invalid url, was specified on doc3")

	assert.Equal(t, doc2.Next, doc3, "doc3 should follow doc2, in .Next")

	doc1fr := doc1en.Translations()[0]
	permalink, err = doc1fr.Permalink()
	assert.NoError(t, err, "permalink call failed")
	assert.Equal(t, "http://example.com/blog/fr/sect/doc1", permalink, "invalid doc1fr permalink")

	assert.Equal(t, doc1en.Translations()[0], doc1fr, "doc1-en should have doc1-fr as translation")
	assert.Equal(t, doc1fr.Translations()[0], doc1en, "doc1-fr should have doc1-en as translation")
	assert.Equal(t, "fr", doc1fr.Language().Lang)

	doc4 := s.AllPages[4]
	permalink, err = doc4.Permalink()
	assert.NoError(t, err, "permalink call failed")
	assert.Equal(t, "http://example.com/blog/fr/sect/doc4", permalink, "invalid doc4 permalink")
	assert.Len(t, doc4.Translations(), 0, "found translations for doc4")

	doc5 := s.AllPages[5]
	permalink, err = doc5.Permalink()
	assert.NoError(t, err, "permalink call failed")
	assert.Equal(t, "http://example.com/blog/fr/somewhere/else/doc5", permalink, "invalid doc5 permalink")

	// Taxonomies and their URLs
	assert.Len(t, s.Taxonomies, 1, "should have 1 taxonomy")
	tags := s.Taxonomies["tags"]
	assert.Len(t, tags, 2, "should have 2 different tags")
	assert.Equal(t, tags["tag1"][0].Page, doc1en, "first tag1 page should be doc1")

	// Expect the tags locations to be in certain places, with the /en/ prefixes, etc..
}

func assertFileContent(t *testing.T, path string, content string) {
	fl, err := hugofs.Destination().Open(path)
	assert.NoError(t, err, "file content not found when asserting on content of %s", path)

	cnt, err := ioutil.ReadAll(fl)
	assert.NoError(t, err, "cannot read file content when asserting on content of %s", path)

	assert.Equal(t, content, string(cnt))
}
