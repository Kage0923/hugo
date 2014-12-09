package hugolib

import (
	"html/template"
	"path/filepath"
	"testing"

	"github.com/spf13/hugo/source"
	"github.com/spf13/viper"
)

func TestPermalink(t *testing.T) {
	tests := []struct {
		file        string
		dir         string
		base        template.URL
		slug        string
		url         string
		uglyurls    bool
		expectedAbs string
		expectedRel string
	}{
		{"x/y/z/boofar.md", "x/y/z", "", "", "", false, "/x/y/z/boofar/", "/x/y/z/boofar/"},
		{"x/y/z/boofar.md", "x/y/z/", "", "", "", false, "/x/y/z/boofar/", "/x/y/z/boofar/"},
		{"x/y/z/boofar.md", "x/y/z/", "", "boofar", "", false, "/x/y/z/boofar/", "/x/y/z/boofar/"},
		{"x/y/z/boofar.md", "x/y/z", "http://barnew/", "", "", false, "http://barnew/x/y/z/boofar/", "/x/y/z/boofar/"},
		{"x/y/z/boofar.md", "x/y/z/", "http://barnew/", "boofar", "", false, "http://barnew/x/y/z/boofar/", "/x/y/z/boofar/"},
		{"x/y/z/boofar.md", "x/y/z", "", "", "", true, "/x/y/z/boofar.html", "/x/y/z/boofar.html"},
		{"x/y/z/boofar.md", "x/y/z/", "", "", "", true, "/x/y/z/boofar.html", "/x/y/z/boofar.html"},
		{"x/y/z/boofar.md", "x/y/z/", "", "boofar", "", true, "/x/y/z/boofar.html", "/x/y/z/boofar.html"},
		{"x/y/z/boofar.md", "x/y/z", "http://barnew/", "", "", true, "http://barnew/x/y/z/boofar.html", "/x/y/z/boofar.html"},
		{"x/y/z/boofar.md", "x/y/z/", "http://barnew/", "boofar", "", true, "http://barnew/x/y/z/boofar.html", "/x/y/z/boofar.html"},

		// test url overrides
		{"x/y/z/boofar.md", "x/y/z", "", "", "/z/y/q/", false, "/z/y/q/", "/z/y/q/"},
	}

	viper.Set("DefaultExtension", "html")

	for i, test := range tests {
		viper.Set("uglyurls", test.uglyurls)
		p := &Page{
			Node: Node{
				UrlPath: UrlPath{
					Section: "z",
					Url:     test.url,
				},
				Site: &SiteInfo{
					BaseUrl: test.base,
				},
			},
			Source: Source{File: *source.NewFile(filepath.FromSlash(test.file))},
		}

		if test.slug != "" {
			p.update(map[string]interface{}{
				"slug": test.slug,
			})
		}

		u, err := p.Permalink()
		if err != nil {
			t.Errorf("Test %d: Unable to process permalink: %s", i, err)
		}

		expected := test.expectedAbs
		if u != expected {
			t.Errorf("Test %d: Expected abs url: %s, got: %s", i, expected, u)
		}

		u, err = p.RelPermalink()
		if err != nil {
			t.Errorf("Test %d: Unable to process permalink: %s", i, err)
		}

		expected = test.expectedRel
		if u != expected {
			t.Errorf("Test %d: Expected abs url: %s, got: %s", i, expected, u)
		}
	}
}
