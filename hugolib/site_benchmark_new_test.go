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
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

type siteBenchmarkTestcase struct {
	name   string
	create func(t testing.TB) *sitesBuilder
	check  func(s *sitesBuilder)
}

func getBenchmarkSiteNewTestCases() []siteBenchmarkTestcase {
	// TODO(bep) create some common and stable data set

	const pageContent = `---
title: "My Page"
---

My page content.

`

	config := `
baseURL = "https://example.com"

`

	benchmarks := []siteBenchmarkTestcase{
		{"Bundle with image", func(b testing.TB) *sitesBuilder {
			sb := newTestSitesBuilder(b).WithConfigFile("toml", config)
			sb.WithContent("content/blog/mybundle/index.md", pageContent)
			sb.WithSunset("content/blog/mybundle/sunset1.jpg")

			return sb
		},
			func(s *sitesBuilder) {
				s.AssertFileContent("public/blog/mybundle/index.html", "/blog/mybundle/sunset1.jpg")
				s.CheckExists("public/blog/mybundle/sunset1.jpg")

			},
		},
		{"Bundle with JSON file", func(b testing.TB) *sitesBuilder {
			sb := newTestSitesBuilder(b).WithConfigFile("toml", config)
			sb.WithContent("content/blog/mybundle/index.md", pageContent)
			sb.WithContent("content/blog/mybundle/mydata.json", `{ "hello": "world" }`)

			return sb
		},
			func(s *sitesBuilder) {
				s.AssertFileContent("public/blog/mybundle/index.html", "Resources: application/json: /blog/mybundle/mydata.json")
				s.CheckExists("public/blog/mybundle/mydata.json")

			},
		},
		{"Deep content tree", func(b testing.TB) *sitesBuilder {

			sb := newTestSitesBuilder(b).WithConfigFile("toml", `
baseURL = "https://example.com"

[languages]
[languages.en]
weight=1
contentDir="content/en"
[languages.fr]
weight=2
contentDir="content/fr"
[languages.no]
weight=3
contentDir="content/no"
[languages.sv]
weight=4
contentDir="content/sv"
			
`)

			createContent := func(dir, name string) {
				sb.WithContent(filepath.Join("content", dir, name), pageContent)
			}

			createBundledFiles := func(dir string) {
				sb.WithContent(filepath.Join("content", dir, "data.json"), `{ "hello": "world" }`)
				for i := 1; i <= 3; i++ {
					sb.WithContent(filepath.Join("content", dir, fmt.Sprintf("page%d.md", i)), pageContent)
				}
			}

			for _, lang := range []string{"en", "fr", "no", "sv"} {
				for level := 1; level <= 5; level++ {
					sectionDir := path.Join(lang, strings.Repeat("section/", level))
					createContent(sectionDir, "_index.md")
					createBundledFiles(sectionDir)
					for i := 1; i <= 3; i++ {
						leafBundleDir := path.Join(sectionDir, fmt.Sprintf("bundle%d", i))
						createContent(leafBundleDir, "index.md")
						createBundledFiles(path.Join(leafBundleDir, "assets1"))
						createBundledFiles(path.Join(leafBundleDir, "assets1", "assets2"))
					}
				}
			}

			return sb
		},
			func(s *sitesBuilder) {
				s.CheckExists("public/blog/mybundle/index.html")
				s.Assertions.Equal(4, len(s.H.Sites))
				s.Assertions.Equal(len(s.H.Sites[0].RegularPages()), len(s.H.Sites[1].RegularPages()))
				s.Assertions.Equal(30, len(s.H.Sites[0].RegularPages()))

			},
		},
	}

	return benchmarks

}

// Run the benchmarks below as tests. Mostly useful when adding new benchmark
// variants.
func TestBenchmarkSiteNew(b *testing.T) {
	benchmarks := getBenchmarkSiteNewTestCases()
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.T) {
			s := bm.create(b)

			err := s.BuildE(BuildCfg{})
			if err != nil {
				b.Fatal(err)
			}
			bm.check(s)

		})
	}
}

// TODO(bep) eventually remove the old (too complicated setup).
func BenchmarkSiteNew(b *testing.B) {
	benchmarks := getBenchmarkSiteNewTestCases()

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			sites := make([]*sitesBuilder, b.N)
			for i := 0; i < b.N; i++ {
				sites[i] = bm.create(b)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				s := sites[i]
				err := s.BuildE(BuildCfg{})
				if err != nil {
					b.Fatal(err)
				}
				bm.check(s)
			}
		})
	}
}
