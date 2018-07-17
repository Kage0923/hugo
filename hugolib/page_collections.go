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
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gohugoio/hugo/helpers"
)

// PageCollections contains the page collections for a site.
type PageCollections struct {
	// Includes only pages of all types, and only pages in the current language.
	Pages Pages

	// Includes all pages in all languages, including the current one.
	// Includes pages of all types.
	AllPages Pages

	// A convenience cache for the traditional index types, taxonomies, home page etc.
	// This is for the current language only.
	indexPages Pages

	// A convenience cache for the regular pages.
	// This is for the current language only.
	RegularPages Pages

	// A convenience cache for the all the regular pages.
	AllRegularPages Pages

	// Includes absolute all pages (of all types), including drafts etc.
	rawAllPages Pages

	// Includes headless bundles, i.e. bundles that produce no output for its content page.
	headlessPages Pages

	pageIndex
}

type pageIndex struct {
	initSync sync.Once
	index    map[string]*Page
	load     func() map[string]*Page
}

func (pi *pageIndex) init() {
	pi.initSync.Do(func() {
		pi.index = pi.load()
	})
}

// Get initializes the index if not already done so, then
// looks up the given page ref, returns nil if no value found.
func (pi *pageIndex) Get(ref string) *Page {
	pi.init()
	return pi.index[ref]
}

var ambiguityFlag = &Page{Kind: "dummy", title: "ambiguity flag"}

func (c *PageCollections) refreshPageCaches() {
	c.indexPages = c.findPagesByKindNotIn(KindPage, c.Pages)
	c.RegularPages = c.findPagesByKindIn(KindPage, c.Pages)
	c.AllRegularPages = c.findPagesByKindIn(KindPage, c.AllPages)

	var s *Site

	if len(c.Pages) > 0 {
		s = c.Pages[0].s
	}

	indexLoader := func() map[string]*Page {
		index := make(map[string]*Page)

		// Note that we deliberately use the pages from all sites
		// in this index, as we intend to use this in the ref and relref
		// shortcodes. If the user says "sect/doc1.en.md", he/she knows
		// what he/she is looking for.
		for _, pageCollection := range []Pages{c.AllRegularPages, c.headlessPages} {
			for _, p := range pageCollection {

				sourceRef := p.absoluteSourceRef()
				if sourceRef != "" {
					// index the canonical, unambiguous ref
					// e.g. /section/article.md
					indexPage(index, sourceRef, p)

					// also index the legacy canonical lookup (not guaranteed to be unambiguous)
					// e.g. section/article.md
					indexPage(index, sourceRef[1:], p)
				}

				if s != nil && p.s == s {
					// Ref/Relref supports this potentially ambiguous lookup.
					indexPage(index, p.Source.LogicalName(), p)

					translationBaseName := p.Source.TranslationBaseName()
					dir := filepath.ToSlash(strings.TrimSuffix(p.Dir(), helpers.FilePathSeparator))

					if translationBaseName == "index" {
						_, name := path.Split(dir)
						indexPage(index, name, p)
						indexPage(index, dir, p)
					} else {
						// Again, ambiguous
						indexPage(index, translationBaseName, p)
					}

					// We need a way to get to the current language version.
					pathWithNoExtensions := path.Join(dir, translationBaseName)
					indexPage(index, pathWithNoExtensions, p)
				}
			}
		}

		for _, p := range c.indexPages {
			// index the canonical, unambiguous ref for any backing file
			// e.g. /section/_index.md
			sourceRef := p.absoluteSourceRef()
			if sourceRef != "" {
				indexPage(index, sourceRef, p)
			}

			ref := path.Join(p.sections...)

			// index the canonical, unambiguous virtual ref
			// e.g. /section
			// (this may already have been indexed above)
			indexPage(index, "/"+ref, p)

			// index the legacy canonical ref (not guaranteed to be unambiguous)
			// e.g. section
			indexPage(index, ref, p)
		}
		return index
	}

	c.pageIndex = pageIndex{load: indexLoader}
}

func indexPage(index map[string]*Page, ref string, p *Page) {
	existing := index[ref]
	if existing == nil {
		index[ref] = p
	} else if existing != ambiguityFlag && existing != p {
		index[ref] = ambiguityFlag
	}
}

func newPageCollections() *PageCollections {
	return &PageCollections{}
}

func newPageCollectionsFromPages(pages Pages) *PageCollections {
	return &PageCollections{rawAllPages: pages}
}

// context: page used to resolve relative paths
// ref: either unix-style paths (i.e. callers responsible for
// calling filepath.ToSlash as necessary) or shorthand refs.
func (c *PageCollections) getPage(context *Page, ref string) (*Page, error) {

	var result *Page

	if len(ref) > 0 && ref[0:1] == "/" {

		// it's an absolute path
		result = c.pageIndex.Get(ref)

	} else { // either relative path or other supported ref

		// If there's a page context. relative ref interpretation takes precedence.
		if context != nil {
			// For relative refs `filepath.Join` will properly resolve ".." (parent dir)
			// and other elements in the path
			apath := path.Join("/", strings.Join(context.sections, "/"), ref)
			result = c.pageIndex.Get(apath)
		}

		// finally, let's try it as-is for a match against all the alternate refs indexed for each page
		if result == nil {
			result = c.pageIndex.Get(ref)

			if result == ambiguityFlag {
				return nil, fmt.Errorf("The reference \"%s\" in %q resolves to more than one page. Use either an absolute path (begins with \"/\") or relative path to the content directory target.", ref, context.absoluteSourceRef())
			}
		}
	}

	return result, nil
}

func (*PageCollections) findPagesByKindIn(kind string, inPages Pages) Pages {
	var pages Pages
	for _, p := range inPages {
		if p.Kind == kind {
			pages = append(pages, p)
		}
	}
	return pages
}

func (*PageCollections) findFirstPageByKindIn(kind string, inPages Pages) *Page {
	for _, p := range inPages {
		if p.Kind == kind {
			return p
		}
	}
	return nil
}

func (*PageCollections) findPagesByKindNotIn(kind string, inPages Pages) Pages {
	var pages Pages
	for _, p := range inPages {
		if p.Kind != kind {
			pages = append(pages, p)
		}
	}
	return pages
}

func (c *PageCollections) findPagesByKind(kind string) Pages {
	return c.findPagesByKindIn(kind, c.Pages)
}

func (c *PageCollections) addPage(page *Page) {
	c.rawAllPages = append(c.rawAllPages, page)
}

func (c *PageCollections) removePageFilename(filename string) {
	if i := c.rawAllPages.findPagePosByFilename(filename); i >= 0 {
		c.clearResourceCacheForPage(c.rawAllPages[i])
		c.rawAllPages = append(c.rawAllPages[:i], c.rawAllPages[i+1:]...)
	}

}

func (c *PageCollections) removePage(page *Page) {
	if i := c.rawAllPages.findPagePos(page); i >= 0 {
		c.clearResourceCacheForPage(c.rawAllPages[i])
		c.rawAllPages = append(c.rawAllPages[:i], c.rawAllPages[i+1:]...)
	}

}

func (c *PageCollections) findPagesByShortcode(shortcode string) Pages {
	var pages Pages

	for _, p := range c.rawAllPages {
		if p.shortcodeState != nil {
			if _, ok := p.shortcodeState.nameSet[shortcode]; ok {
				pages = append(pages, p)
			}
		}
	}
	return pages
}

func (c *PageCollections) replacePage(page *Page) {
	// will find existing page that matches filepath and remove it
	c.removePage(page)
	c.addPage(page)
}

func (c *PageCollections) clearResourceCacheForPage(page *Page) {
	if len(page.Resources) > 0 {
		first := page.Resources[0]
		dir := path.Dir(first.RelPermalink())
		dir = strings.TrimPrefix(dir, page.LanguagePrefix())
		// This is done to keep the memory usage in check when doing live reloads.
		page.s.ResourceSpec.DeleteCacheByPrefix(dir)
	}
}
