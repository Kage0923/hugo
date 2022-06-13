// Copyright 2015 The Hugo Authors. All rights reserved.
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

package helpers

import (
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/gohugoio/hugo/common/paths"

	"github.com/PuerkitoBio/purell"
)

func sanitizeURLWithFlags(in string, f purell.NormalizationFlags) string {
	s, err := purell.NormalizeURLString(in, f)
	if err != nil {
		return in
	}

	// Temporary workaround for the bug fix and resulting
	// behavioral change in purell.NormalizeURLString():
	// a leading '/' was inadvertently added to relative links,
	// but no longer, see #878.
	//
	// I think the real solution is to allow Hugo to
	// make relative URL with relative path,
	// e.g. "../../post/hello-again/", as wished by users
	// in issues #157, #622, etc., without forcing
	// relative URLs to begin with '/'.
	// Once the fixes are in, let's remove this kludge
	// and restore SanitizeURL() to the way it was.
	//                         -- @anthonyfok, 2015-02-16
	//
	// Begin temporary kludge
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	if len(u.Path) > 0 && !strings.HasPrefix(u.Path, "/") {
		u.Path = "/" + u.Path
	}
	return u.String()
	// End temporary kludge

	// return s

}

// SanitizeURL sanitizes the input URL string.
func SanitizeURL(in string) string {
	return sanitizeURLWithFlags(in, purell.FlagsSafe|purell.FlagRemoveTrailingSlash|purell.FlagRemoveDotSegments|purell.FlagRemoveDuplicateSlashes|purell.FlagRemoveUnnecessaryHostDots|purell.FlagRemoveEmptyPortSeparator)
}

// SanitizeURLKeepTrailingSlash is the same as SanitizeURL, but will keep any trailing slash.
func SanitizeURLKeepTrailingSlash(in string) string {
	return sanitizeURLWithFlags(in, purell.FlagsSafe|purell.FlagRemoveDotSegments|purell.FlagRemoveDuplicateSlashes|purell.FlagRemoveUnnecessaryHostDots|purell.FlagRemoveEmptyPortSeparator)
}

// URLize is similar to MakePath, but with Unicode handling
// Example:
//     uri: Vim (text editor)
//     urlize: vim-text-editor
func (p *PathSpec) URLize(uri string) string {
	return p.URLEscape(p.MakePathSanitized(uri))
}

// URLizeFilename creates an URL from a filename by escaping unicode letters
// and turn any filepath separator into forward slashes.
func (p *PathSpec) URLizeFilename(filename string) string {
	return p.URLEscape(filepath.ToSlash(filename))
}

// URLEscape escapes unicode letters.
func (p *PathSpec) URLEscape(uri string) string {
	// escape unicode letters
	parsedURI, err := url.Parse(uri)
	if err != nil {
		// if net/url can not parse URL it means Sanitize works incorrectly
		panic(err)
	}
	x := parsedURI.String()
	return x
}

// AbsURL creates an absolute URL from the relative path given and the BaseURL set in config.
func (p *PathSpec) AbsURL(in string, addLanguage bool) string {
	url, err := url.Parse(in)
	if err != nil {
		return in
	}

	if url.IsAbs() || strings.HasPrefix(in, "//") {
		// It  is already  absolute, return it as is.
		return in
	}

	baseURL := p.getBaseURLRoot(in)

	if addLanguage {
		prefix := p.GetLanguagePrefix()
		if prefix != "" {
			hasPrefix := false
			// avoid adding language prefix if already present
			in2 := in
			if strings.HasPrefix(in, "/") {
				in2 = in[1:]
			}
			if in2 == prefix {
				hasPrefix = true
			} else {
				hasPrefix = strings.HasPrefix(in2, prefix+"/")
			}

			if !hasPrefix {
				addSlash := in == "" || strings.HasSuffix(in, "/")
				in = path.Join(prefix, in)

				if addSlash {
					in += "/"
				}
			}
		}
	}

	return paths.MakePermalink(baseURL, in).String()
}

func (p *PathSpec) getBaseURLRoot(path string) string {
	if strings.HasPrefix(path, "/") {
		// Treat it as relative to the server root.
		return p.BaseURLNoPathString
	} else {
		// Treat it as relative to the baseURL.
		return p.BaseURLString
	}
}

func (p *PathSpec) RelURL(in string, addLanguage bool) string {
	baseURL := p.getBaseURLRoot(in)
	canonifyURLs := p.CanonifyURLs
	if (!strings.HasPrefix(in, baseURL) && strings.HasPrefix(in, "http")) || strings.HasPrefix(in, "//") {
		return in
	}

	u := in

	if strings.HasPrefix(in, baseURL) {
		u = strings.TrimPrefix(u, baseURL)
	}

	if addLanguage {
		prefix := p.GetLanguagePrefix()
		if prefix != "" {
			hasPrefix := false
			// avoid adding language prefix if already present
			in2 := in
			if strings.HasPrefix(in, "/") {
				in2 = in[1:]
			}
			if in2 == prefix {
				hasPrefix = true
			} else {
				hasPrefix = strings.HasPrefix(in2, prefix+"/")
			}

			if !hasPrefix {
				hadSlash := strings.HasSuffix(u, "/")

				u = path.Join(prefix, u)

				if hadSlash {
					u += "/"
				}
			}
		}
	}

	if !canonifyURLs {
		u = paths.AddContextRoot(baseURL, u)
	}

	if in == "" && !strings.HasSuffix(u, "/") && strings.HasSuffix(baseURL, "/") {
		u += "/"
	}

	if !strings.HasPrefix(u, "/") {
		u = "/" + u
	}

	return u
}

// PrependBasePath prepends any baseURL sub-folder to the given resource
func (p *PathSpec) PrependBasePath(rel string, isAbs bool) string {
	basePath := p.GetBasePath(!isAbs)
	if basePath != "" {
		rel = filepath.ToSlash(rel)
		// Need to prepend any path from the baseURL
		hadSlash := strings.HasSuffix(rel, "/")
		rel = path.Join(basePath, rel)
		if hadSlash {
			rel += "/"
		}
	}
	return rel
}

// URLizeAndPrep applies misc sanitation to the given URL to get it in line
// with the Hugo standard.
func (p *PathSpec) URLizeAndPrep(in string) string {
	return p.URLPrep(p.URLize(in))
}

// URLPrep applies misc sanitation to the given URL.
func (p *PathSpec) URLPrep(in string) string {
	if p.UglyURLs {
		return paths.Uglify(SanitizeURL(in))
	}
	pretty := paths.PrettifyURL(SanitizeURL(in))
	if path.Ext(pretty) == ".xml" {
		return pretty
	}
	url, err := purell.NormalizeURLString(pretty, purell.FlagAddTrailingSlash)
	if err != nil {
		return pretty
	}
	return url
}
