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
	"sync"

	"github.com/pkg/errors"

	"github.com/gohugoio/hugo/cache"
	"github.com/gohugoio/hugo/helpers"
	"github.com/gohugoio/hugo/resources/page"
)

// Used in the page cache to mark more than one hit for a given key.
var ambiguityFlag = &pageState{}

// PageCollections contains the page collections for a site.
type PageCollections struct {

	// Includes absolute all pages (of all types), including drafts etc.
	rawAllPages pageStatePages

	// rawAllPages plus additional pages created during the build process.
	workAllPages pageStatePages

	// Includes headless bundles, i.e. bundles that produce no output for its content page.
	headlessPages pageStatePages

	// Lazy initialized page collections
	pages           *lazyPagesFactory
	regularPages    *lazyPagesFactory
	allPages        *lazyPagesFactory
	allRegularPages *lazyPagesFactory

	// The index for .Site.GetPage etc.
	pageIndex *cache.Lazy
}

// Pages returns all pages.
// This is for the current language only.
func (c *PageCollections) Pages() page.Pages {
	return c.pages.get()
}

// RegularPages returns all the regular pages.
// This is for the current language only.
func (c *PageCollections) RegularPages() page.Pages {
	return c.regularPages.get()
}

// AllPages returns all pages for all languages.
func (c *PageCollections) AllPages() page.Pages {
	return c.allPages.get()
}

// AllPages returns all regular pages for all languages.
func (c *PageCollections) AllRegularPages() page.Pages {
	return c.allRegularPages.get()
}

// Get initializes the index if not already done so, then
// looks up the given page ref, returns nil if no value found.
func (c *PageCollections) getFromCache(ref string) (page.Page, error) {
	v, found, err := c.pageIndex.Get(ref)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	p := v.(page.Page)

	if p != ambiguityFlag {
		return p, nil
	}
	return nil, fmt.Errorf("page reference %q is ambiguous", ref)
}

type lazyPagesFactory struct {
	pages page.Pages

	init    sync.Once
	factory page.PagesFactory
}

func (l *lazyPagesFactory) get() page.Pages {
	l.init.Do(func() {
		l.pages = l.factory()
	})
	return l.pages
}

func newLazyPagesFactory(factory page.PagesFactory) *lazyPagesFactory {
	return &lazyPagesFactory{factory: factory}
}

func newPageCollections() *PageCollections {
	return newPageCollectionsFromPages(nil)
}

func newPageCollectionsFromPages(pages pageStatePages) *PageCollections {

	c := &PageCollections{rawAllPages: pages}

	c.pages = newLazyPagesFactory(func() page.Pages {
		pages := make(page.Pages, len(c.workAllPages))
		for i, p := range c.workAllPages {
			pages[i] = p
		}
		return pages
	})

	c.regularPages = newLazyPagesFactory(func() page.Pages {
		return c.findPagesByKindInWorkPages(page.KindPage, c.workAllPages)
	})

	c.pageIndex = cache.NewLazy(func() (map[string]interface{}, error) {
		index := make(map[string]interface{})

		add := func(ref string, p page.Page) {
			ref = strings.ToLower(ref)
			existing := index[ref]
			if existing == nil {
				index[ref] = p
			} else if existing != ambiguityFlag && existing != p {
				index[ref] = ambiguityFlag
			}
		}

		for _, pageCollection := range []pageStatePages{c.workAllPages, c.headlessPages} {
			for _, p := range pageCollection {
				if p.IsPage() {
					sourceRef := p.sourceRef()
					if sourceRef != "" {
						// index the canonical ref
						// e.g. /section/article.md
						add(sourceRef, p)
					}

					// Ref/Relref supports this potentially ambiguous lookup.
					add(p.File().LogicalName(), p)

					translationBaseName := p.File().TranslationBaseName()

					dir, _ := path.Split(sourceRef)
					dir = strings.TrimSuffix(dir, "/")

					if translationBaseName == "index" {
						add(dir, p)
						add(path.Base(dir), p)
					} else {
						add(translationBaseName, p)
					}

					// We need a way to get to the current language version.
					pathWithNoExtensions := path.Join(dir, translationBaseName)
					add(pathWithNoExtensions, p)
				} else {
					// index the canonical, unambiguous ref for any backing file
					// e.g. /section/_index.md
					sourceRef := p.sourceRef()
					if sourceRef != "" {
						add(sourceRef, p)
					}

					ref := p.SectionsPath()

					// index the canonical, unambiguous virtual ref
					// e.g. /section
					// (this may already have been indexed above)
					add("/"+ref, p)
				}
			}
		}

		return index, nil
	})

	return c
}

// This is an adapter func for the old API with Kind as first argument.
// This is invoked when you do .Site.GetPage. We drop the Kind and fails
// if there are more than 2 arguments, which would be ambigous.
func (c *PageCollections) getPageOldVersion(ref ...string) (page.Page, error) {
	var refs []string
	for _, r := range ref {
		// A common construct in the wild is
		// .Site.GetPage "home" "" or
		// .Site.GetPage "home" "/"
		if r != "" && r != "/" {
			refs = append(refs, r)
		}
	}

	var key string

	if len(refs) > 2 {
		// This was allowed in Hugo <= 0.44, but we cannot support this with the
		// new API. This should be the most unusual case.
		return nil, fmt.Errorf(`too many arguments to .Site.GetPage: %v. Use lookups on the form {{ .Site.GetPage "/posts/mypage-md" }}`, ref)
	}

	if len(refs) == 0 || refs[0] == page.KindHome {
		key = "/"
	} else if len(refs) == 1 {
		if len(ref) == 2 && refs[0] == page.KindSection {
			// This is an old style reference to the "Home Page section".
			// Typically fetched via {{ .Site.GetPage "section" .Section }}
			// See https://github.com/gohugoio/hugo/issues/4989
			key = "/"
		} else {
			key = refs[0]
		}
	} else {
		key = refs[1]
	}

	key = filepath.ToSlash(key)
	if !strings.HasPrefix(key, "/") {
		key = "/" + key
	}

	return c.getPageNew(nil, key)
}

// 	Only used in tests.
func (c *PageCollections) getPage(typ string, sections ...string) page.Page {
	refs := append([]string{typ}, path.Join(sections...))
	p, _ := c.getPageOldVersion(refs...)
	return p
}

// Case insensitive page lookup.
func (c *PageCollections) getPageNew(context page.Page, ref string) (page.Page, error) {
	var anError error

	ref = strings.ToLower(ref)

	// Absolute (content root relative) reference.
	if strings.HasPrefix(ref, "/") {
		p, err := c.getFromCache(ref)
		if err == nil && p != nil {
			return p, nil
		}
		if err != nil {
			anError = err
		}

	} else if context != nil {
		// Try the page-relative path.
		ppath := path.Join("/", strings.ToLower(context.SectionsPath()), ref)
		p, err := c.getFromCache(ppath)
		if err == nil && p != nil {
			return p, nil
		}
		if err != nil {
			anError = err
		}
	}

	if !strings.HasPrefix(ref, "/") {
		// Many people will have "post/foo.md" in their content files.
		p, err := c.getFromCache("/" + ref)
		if err == nil && p != nil {
			if context != nil {
				// TODO(bep) remove this case and the message below when the storm has passed
				err := wrapErr(errors.Errorf(`make non-relative ref/relref page reference(s) in page %q absolute, e.g. {{< ref "/blog/my-post.md" >}}`, context.Path()), context)
				helpers.DistinctWarnLog.Println(err)
			}
			return p, nil
		}
		if err != nil {
			anError = err
		}
	}

	// Last try.
	ref = strings.TrimPrefix(ref, "/")
	p, err := c.getFromCache(ref)
	if err != nil {
		anError = err
	}

	if p == nil && anError != nil {
		return nil, wrapErr(errors.Wrap(anError, "failed to resolve ref"), context)
	}

	return p, nil
}

func (*PageCollections) findPagesByKindIn(kind string, inPages page.Pages) page.Pages {
	var pages page.Pages
	for _, p := range inPages {
		if p.Kind() == kind {
			pages = append(pages, p)
		}
	}
	return pages
}

func (c *PageCollections) findPagesByKind(kind string) page.Pages {
	return c.findPagesByKindIn(kind, c.Pages())
}

func (c *PageCollections) findWorkPagesByKind(kind string) pageStatePages {
	var pages pageStatePages
	for _, p := range c.workAllPages {
		if p.Kind() == kind {
			pages = append(pages, p)
		}
	}
	return pages
}

func (*PageCollections) findPagesByKindInWorkPages(kind string, inPages pageStatePages) page.Pages {
	var pages page.Pages
	for _, p := range inPages {
		if p.Kind() == kind {
			pages = append(pages, p)
		}
	}
	return pages
}

func (c *PageCollections) findFirstWorkPageByKindIn(kind string) *pageState {
	for _, p := range c.workAllPages {
		if p.Kind() == kind {
			return p
		}
	}
	return nil
}

func (c *PageCollections) addPage(page *pageState) {
	c.rawAllPages = append(c.rawAllPages, page)
}

func (c *PageCollections) removePageFilename(filename string) {
	if i := c.rawAllPages.findPagePosByFilename(filename); i >= 0 {
		c.clearResourceCacheForPage(c.rawAllPages[i])
		c.rawAllPages = append(c.rawAllPages[:i], c.rawAllPages[i+1:]...)
	}

}

func (c *PageCollections) removePage(page *pageState) {
	if i := c.rawAllPages.findPagePos(page); i >= 0 {
		c.clearResourceCacheForPage(c.rawAllPages[i])
		c.rawAllPages = append(c.rawAllPages[:i], c.rawAllPages[i+1:]...)
	}
}

func (c *PageCollections) findPagesByShortcode(shortcode string) page.Pages {
	var pages page.Pages
	for _, p := range c.rawAllPages {
		if p.HasShortcode(shortcode) {
			pages = append(pages, p)
		}
	}
	return pages
}

func (c *PageCollections) replacePage(page *pageState) {
	// will find existing page that matches filepath and remove it
	c.removePage(page)
	c.addPage(page)
}

func (c *PageCollections) clearResourceCacheForPage(page *pageState) {
	if len(page.resources) > 0 {
		page.s.ResourceSpec.DeleteCacheByPrefix(page.targetPaths().SubResourceBaseTarget)
	}
}
