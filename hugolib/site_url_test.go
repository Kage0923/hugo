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
	"path/filepath"
	"testing"

	"github.com/spf13/hugo/helpers"

	"html/template"

	"github.com/spf13/hugo/hugofs"
	"github.com/spf13/hugo/source"
	"github.com/spf13/viper"
)

const slugDoc1 = "---\ntitle: slug doc 1\nslug: slug-doc-1\naliases:\n - sd1/foo/\n - sd2\n - sd3/\n - sd4.html\n---\nslug doc 1 content\n"

const slugDoc2 = `---
title: slug doc 2
slug: slug-doc-2
---
slug doc 2 content
`

const indexTemplate = "{{ range .Data.Pages }}.{{ end }}"

func must(err error) {
	if err != nil {
		panic(err)
	}
}

var urlFakeSource = []source.ByteSource{
	{Name: filepath.FromSlash("content/blue/doc1.md"), Content: []byte(slugDoc1)},
	{Name: filepath.FromSlash("content/blue/doc2.md"), Content: []byte(slugDoc2)},
}

// Issue #1105
func TestShouldNotAddTrailingSlashToBaseURL(t *testing.T) {
	testCommonResetState()

	for i, this := range []struct {
		in       string
		expected string
	}{
		{"http://base.com/", "http://base.com/"},
		{"http://base.com/sub/", "http://base.com/sub/"},
		{"http://base.com/sub", "http://base.com/sub"},
		{"http://base.com", "http://base.com"}} {

		viper.Set("baseURL", this.in)
		s := newSiteDefaultLang()
		s.initializeSiteInfo()

		if s.Info.BaseURL != template.URL(this.expected) {
			t.Errorf("[%d] got %s expected %s", i, s.Info.BaseURL, this.expected)
		}
	}

}

func TestPageCount(t *testing.T) {
	testCommonResetState()
	hugofs.InitMemFs()

	viper.Set("uglyURLs", false)
	viper.Set("paginate", 10)
	s := &Site{
		Source:   &source.InMemorySource{ByteSource: urlFakeSource},
		Language: helpers.NewDefaultLanguage(),
	}

	if err := buildAndRenderSite(s, "indexes/blue.html", indexTemplate); err != nil {
		t.Fatalf("Failed to build site: %s", err)
	}
	_, err := hugofs.Destination().Open("public/blue")
	if err != nil {
		t.Errorf("No indexed rendered.")
	}

	for _, s := range []string{
		"public/sd1/foo/index.html",
		"public/sd2/index.html",
		"public/sd3/index.html",
		"public/sd4.html",
	} {
		if _, err := hugofs.Destination().Open(filepath.FromSlash(s)); err != nil {
			t.Errorf("No alias rendered: %s", s)
		}
	}
}
