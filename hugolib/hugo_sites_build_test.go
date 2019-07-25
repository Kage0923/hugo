package hugolib

import (
	"fmt"
	"strings"
	"testing"

	"path/filepath"
	"time"

	"github.com/gohugoio/hugo/resources/page"

	"github.com/fortytw2/leaktest"
	"github.com/fsnotify/fsnotify"
	"github.com/gohugoio/hugo/helpers"
	"github.com/gohugoio/hugo/hugofs"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestMultiSitesMainLangInRoot(t *testing.T) {
	t.Parallel()
	for _, b := range []bool{false} {
		doTestMultiSitesMainLangInRoot(t, b)
	}
}

func doTestMultiSitesMainLangInRoot(t *testing.T, defaultInSubDir bool) {
	assert := require.New(t)

	siteConfig := map[string]interface{}{
		"DefaultContentLanguage":         "fr",
		"DefaultContentLanguageInSubdir": defaultInSubDir,
	}

	b := newMultiSiteTestBuilder(t, "toml", multiSiteTOMLConfigTemplate, siteConfig)

	pathMod := func(s string) string {
		return s
	}

	if !defaultInSubDir {
		pathMod = func(s string) string {
			return strings.Replace(s, "/fr/", "/", -1)
		}
	}

	b.CreateSites()
	b.Build(BuildCfg{})

	sites := b.H.Sites

	require.Len(t, sites, 4)

	enSite := sites[0]
	frSite := sites[1]

	assert.Equal("/en", enSite.Info.LanguagePrefix)

	if defaultInSubDir {
		assert.Equal("/fr", frSite.Info.LanguagePrefix)
	} else {
		assert.Equal("", frSite.Info.LanguagePrefix)
	}

	assert.Equal("/blog/en/foo", enSite.PathSpec.RelURL("foo", true))

	doc1en := enSite.RegularPages()[0]
	doc1fr := frSite.RegularPages()[0]

	enPerm := doc1en.Permalink()
	enRelPerm := doc1en.RelPermalink()
	assert.Equal("http://example.com/blog/en/sect/doc1-slug/", enPerm)
	assert.Equal("/blog/en/sect/doc1-slug/", enRelPerm)

	frPerm := doc1fr.Permalink()
	frRelPerm := doc1fr.RelPermalink()

	b.AssertFileContent(pathMod("public/fr/sect/doc1/index.html"), "Single", "Bonjour")
	b.AssertFileContent("public/en/sect/doc1-slug/index.html", "Single", "Hello")

	if defaultInSubDir {
		assert.Equal("http://example.com/blog/fr/sect/doc1/", frPerm)
		assert.Equal("/blog/fr/sect/doc1/", frRelPerm)

		// should have a redirect on top level.
		b.AssertFileContent("public/index.html", `<meta http-equiv="refresh" content="0; url=http://example.com/blog/fr" />`)
	} else {
		// Main language in root
		assert.Equal("http://example.com/blog/sect/doc1/", frPerm)
		assert.Equal("/blog/sect/doc1/", frRelPerm)

		// should have redirect back to root
		b.AssertFileContent("public/fr/index.html", `<meta http-equiv="refresh" content="0; url=http://example.com/blog" />`)
	}
	b.AssertFileContent(pathMod("public/fr/index.html"), "Home", "Bonjour")
	b.AssertFileContent("public/en/index.html", "Home", "Hello")

	// Check list pages
	b.AssertFileContent(pathMod("public/fr/sect/index.html"), "List", "Bonjour")
	b.AssertFileContent("public/en/sect/index.html", "List", "Hello")
	b.AssertFileContent(pathMod("public/fr/plaques/FRtag1/index.html"), "Taxonomy List", "Bonjour")
	b.AssertFileContent("public/en/tags/tag1/index.html", "Taxonomy List", "Hello")

	// Check sitemaps
	// Sitemaps behaves different: In a multilanguage setup there will always be a index file and
	// one sitemap in each lang folder.
	b.AssertFileContent("public/sitemap.xml",
		"<loc>http://example.com/blog/en/sitemap.xml</loc>",
		"<loc>http://example.com/blog/fr/sitemap.xml</loc>")

	if defaultInSubDir {
		b.AssertFileContent("public/fr/sitemap.xml", "<loc>http://example.com/blog/fr/</loc>")
	} else {
		b.AssertFileContent("public/fr/sitemap.xml", "<loc>http://example.com/blog/</loc>")
	}
	b.AssertFileContent("public/en/sitemap.xml", "<loc>http://example.com/blog/en/</loc>")

	// Check rss
	b.AssertFileContent(pathMod("public/fr/index.xml"), pathMod(`<atom:link href="http://example.com/blog/fr/index.xml"`),
		`rel="self" type="application/rss+xml"`)
	b.AssertFileContent("public/en/index.xml", `<atom:link href="http://example.com/blog/en/index.xml"`)
	b.AssertFileContent(
		pathMod("public/fr/sect/index.xml"),
		pathMod(`<atom:link href="http://example.com/blog/fr/sect/index.xml"`))
	b.AssertFileContent("public/en/sect/index.xml", `<atom:link href="http://example.com/blog/en/sect/index.xml"`)
	b.AssertFileContent(
		pathMod("public/fr/plaques/FRtag1/index.xml"),
		pathMod(`<atom:link href="http://example.com/blog/fr/plaques/FRtag1/index.xml"`))
	b.AssertFileContent("public/en/tags/tag1/index.xml", `<atom:link href="http://example.com/blog/en/tags/tag1/index.xml"`)

	// Check paginators
	b.AssertFileContent(pathMod("public/fr/page/1/index.html"), pathMod(`refresh" content="0; url=http://example.com/blog/fr/"`))
	b.AssertFileContent("public/en/page/1/index.html", `refresh" content="0; url=http://example.com/blog/en/"`)
	b.AssertFileContent(pathMod("public/fr/page/2/index.html"), "Home Page 2", "Bonjour", pathMod("http://example.com/blog/fr/"))
	b.AssertFileContent("public/en/page/2/index.html", "Home Page 2", "Hello", "http://example.com/blog/en/")
	b.AssertFileContent(pathMod("public/fr/sect/page/1/index.html"), pathMod(`refresh" content="0; url=http://example.com/blog/fr/sect/"`))
	b.AssertFileContent("public/en/sect/page/1/index.html", `refresh" content="0; url=http://example.com/blog/en/sect/"`)
	b.AssertFileContent(pathMod("public/fr/sect/page/2/index.html"), "List Page 2", "Bonjour", pathMod("http://example.com/blog/fr/sect/"))
	b.AssertFileContent("public/en/sect/page/2/index.html", "List Page 2", "Hello", "http://example.com/blog/en/sect/")
	b.AssertFileContent(
		pathMod("public/fr/plaques/FRtag1/page/1/index.html"),
		pathMod(`refresh" content="0; url=http://example.com/blog/fr/plaques/FRtag1/"`))
	b.AssertFileContent("public/en/tags/tag1/page/1/index.html", `refresh" content="0; url=http://example.com/blog/en/tags/tag1/"`)
	b.AssertFileContent(
		pathMod("public/fr/plaques/FRtag1/page/2/index.html"), "List Page 2", "Bonjour",
		pathMod("http://example.com/blog/fr/plaques/FRtag1/"))
	b.AssertFileContent("public/en/tags/tag1/page/2/index.html", "List Page 2", "Hello", "http://example.com/blog/en/tags/tag1/")
	// nn (Nynorsk) and nb (Bokmål) have custom pagePath: side ("page" in Norwegian)
	b.AssertFileContent("public/nn/side/1/index.html", `refresh" content="0; url=http://example.com/blog/nn/"`)
	b.AssertFileContent("public/nb/side/1/index.html", `refresh" content="0; url=http://example.com/blog/nb/"`)
}

func TestMultiSitesWithTwoLanguages(t *testing.T) {
	t.Parallel()

	assert := require.New(t)
	b := newTestSitesBuilder(t).WithConfigFile("toml", `

defaultContentLanguage = "nn"

[languages]
[languages.nn]
languageName = "Nynorsk"
weight = 1
title = "Tittel på Nynorsk"
[languages.nn.params]
p1 = "p1nn"

[languages.en]
title = "Title in English"
languageName = "English"
weight = 2
[languages.en.params]
p1 = "p1en"
`)

	b.CreateSites()
	b.Build(BuildCfg{SkipRender: true})
	sites := b.H.Sites

	assert.Len(sites, 2)

	nnSite := sites[0]
	nnHome := nnSite.getPage(page.KindHome)
	assert.Len(nnHome.AllTranslations(), 2)
	assert.Len(nnHome.Translations(), 1)
	assert.True(nnHome.IsTranslated())

	enHome := sites[1].getPage(page.KindHome)

	p1, err := enHome.Param("p1")
	assert.NoError(err)
	assert.Equal("p1en", p1)

	p1, err = nnHome.Param("p1")
	assert.NoError(err)
	assert.Equal("p1nn", p1)
}

func TestMultiSitesBuild(t *testing.T) {

	for _, config := range []struct {
		content string
		suffix  string
	}{
		{multiSiteTOMLConfigTemplate, "toml"},
		{multiSiteYAMLConfigTemplate, "yml"},
		{multiSiteJSONConfigTemplate, "json"},
	} {

		t.Run(config.suffix, func(t *testing.T) {
			t.Parallel()
			doTestMultiSitesBuild(t, config.content, config.suffix)
		})
	}
}

func doTestMultiSitesBuild(t *testing.T, configTemplate, configSuffix string) {
	assert := require.New(t)

	b := newMultiSiteTestBuilder(t, configSuffix, configTemplate, nil)
	b.CreateSites()

	sites := b.H.Sites
	assert.Equal(4, len(sites))

	b.Build(BuildCfg{})

	// Check site config
	for _, s := range sites {
		require.True(t, s.Info.defaultContentLanguageInSubdir, s.Info.title)
		require.NotNil(t, s.disabledKinds)
	}

	gp1 := b.H.GetContentPage(filepath.FromSlash("content/sect/doc1.en.md"))
	require.NotNil(t, gp1)
	require.Equal(t, "doc1", gp1.Title())
	gp2 := b.H.GetContentPage(filepath.FromSlash("content/dummysect/notfound.md"))
	require.Nil(t, gp2)

	enSite := sites[0]
	enSiteHome := enSite.getPage(page.KindHome)
	require.True(t, enSiteHome.IsTranslated())

	require.Equal(t, "en", enSite.language.Lang)

	//dumpPages(enSite.RegularPages()...)

	assert.Equal(5, len(enSite.RegularPages()))
	assert.Equal(32, len(enSite.AllPages()))

	// Check 404s
	b.AssertFileContent("public/en/404.html", "404|en|404 Page not found")
	b.AssertFileContent("public/fr/404.html", "404|fr|404 Page not found")

	// Check robots.txt
	b.AssertFileContent("public/en/robots.txt", "robots|en|")
	b.AssertFileContent("public/nn/robots.txt", "robots|nn|")

	b.AssertFileContent("public/en/sect/doc1-slug/index.html", "Permalink: http://example.com/blog/en/sect/doc1-slug/")
	b.AssertFileContent("public/en/sect/doc2/index.html", "Permalink: http://example.com/blog/en/sect/doc2/")
	b.AssertFileContent("public/superbob/index.html", "Permalink: http://example.com/blog/superbob/")

	doc2 := enSite.RegularPages()[1]
	doc3 := enSite.RegularPages()[2]
	require.Equal(t, doc2.Prev(), doc3, "doc3 should follow doc2, in .PrevPage")
	doc1en := enSite.RegularPages()[0]
	doc1fr := doc1en.Translations()[0]
	b.AssertFileContent("public/fr/sect/doc1/index.html", "Permalink: http://example.com/blog/fr/sect/doc1/")

	require.Equal(t, doc1en.Translations()[0], doc1fr, "doc1-en should have doc1-fr as translation")
	require.Equal(t, doc1fr.Translations()[0], doc1en, "doc1-fr should have doc1-en as translation")
	require.Equal(t, "fr", doc1fr.Language().Lang)

	doc4 := enSite.AllPages()[4]
	require.Len(t, doc4.Translations(), 0, "found translations for doc4")

	// Taxonomies and their URLs
	require.Len(t, enSite.Taxonomies, 1, "should have 1 taxonomy")
	tags := enSite.Taxonomies["tags"]
	require.Len(t, tags, 2, "should have 2 different tags")
	require.Equal(t, tags["tag1"][0].Page, doc1en, "first tag1 page should be doc1")

	frSite := sites[1]

	require.Equal(t, "fr", frSite.language.Lang)
	require.Len(t, frSite.RegularPages(), 4, "should have 3 pages")
	require.Len(t, frSite.AllPages(), 32, "should have 32 total pages (including translations and nodes)")

	for _, frenchPage := range frSite.RegularPages() {
		p := frenchPage
		require.Equal(t, "fr", p.Language().Lang)
	}

	// See https://github.com/gohugoio/hugo/issues/4285
	// Before Hugo 0.33 you had to be explicit with the content path to get the correct Page, which
	// isn't ideal in a multilingual setup. You want a way to get the current language version if available.
	// Now you can do lookups with translation base name to get that behaviour.
	// Let us test all the regular page variants:
	getPageDoc1En := enSite.getPage(page.KindPage, filepath.ToSlash(doc1en.File().Path()))
	getPageDoc1EnBase := enSite.getPage(page.KindPage, "sect/doc1")
	getPageDoc1Fr := frSite.getPage(page.KindPage, filepath.ToSlash(doc1fr.File().Path()))
	getPageDoc1FrBase := frSite.getPage(page.KindPage, "sect/doc1")
	require.Equal(t, doc1en, getPageDoc1En)
	require.Equal(t, doc1fr, getPageDoc1Fr)
	require.Equal(t, doc1en, getPageDoc1EnBase)
	require.Equal(t, doc1fr, getPageDoc1FrBase)

	// Check redirect to main language, French
	b.AssertFileContent("public/index.html", "0; url=http://example.com/blog/fr")

	// check home page content (including data files rendering)
	b.AssertFileContent("public/en/index.html", "Default Home Page 1", "Hello", "Hugo Rocks!")
	b.AssertFileContent("public/fr/index.html", "French Home Page 1", "Bonjour", "Hugo Rocks!")

	// check single page content
	b.AssertFileContent("public/fr/sect/doc1/index.html", "Single", "Shortcode: Bonjour", "LingoFrench")
	b.AssertFileContent("public/en/sect/doc1-slug/index.html", "Single", "Shortcode: Hello", "LingoDefault")

	// Check node translations
	homeEn := enSite.getPage(page.KindHome)
	require.NotNil(t, homeEn)
	require.Len(t, homeEn.Translations(), 3)
	require.Equal(t, "fr", homeEn.Translations()[0].Language().Lang)
	require.Equal(t, "nn", homeEn.Translations()[1].Language().Lang)
	require.Equal(t, "På nynorsk", homeEn.Translations()[1].Title())
	require.Equal(t, "nb", homeEn.Translations()[2].Language().Lang)
	require.Equal(t, "På bokmål", homeEn.Translations()[2].Title(), configSuffix)
	require.Equal(t, "Bokmål", homeEn.Translations()[2].Language().LanguageName, configSuffix)

	sectFr := frSite.getPage(page.KindSection, "sect")
	require.NotNil(t, sectFr)

	require.Equal(t, "fr", sectFr.Language().Lang)
	require.Len(t, sectFr.Translations(), 1)
	require.Equal(t, "en", sectFr.Translations()[0].Language().Lang)
	require.Equal(t, "Sects", sectFr.Translations()[0].Title())

	nnSite := sites[2]
	require.Equal(t, "nn", nnSite.language.Lang)
	taxNn := nnSite.getPage(page.KindTaxonomyTerm, "lag")
	require.NotNil(t, taxNn)
	require.Len(t, taxNn.Translations(), 1)
	require.Equal(t, "nb", taxNn.Translations()[0].Language().Lang)

	taxTermNn := nnSite.getPage(page.KindTaxonomy, "lag", "sogndal")
	require.NotNil(t, taxTermNn)
	require.Equal(t, taxTermNn, nnSite.getPage(page.KindTaxonomy, "LAG", "SOGNDAL"))
	require.Len(t, taxTermNn.Translations(), 1)
	require.Equal(t, "nb", taxTermNn.Translations()[0].Language().Lang)

	// Check sitemap(s)
	b.AssertFileContent("public/sitemap.xml",
		"<loc>http://example.com/blog/en/sitemap.xml</loc>",
		"<loc>http://example.com/blog/fr/sitemap.xml</loc>")
	b.AssertFileContent("public/en/sitemap.xml", "http://example.com/blog/en/sect/doc2/")
	b.AssertFileContent("public/fr/sitemap.xml", "http://example.com/blog/fr/sect/doc1/")

	// Check taxonomies
	enTags := enSite.Taxonomies["tags"]
	frTags := frSite.Taxonomies["plaques"]
	require.Len(t, enTags, 2, fmt.Sprintf("Tags in en: %v", enTags))
	require.Len(t, frTags, 2, fmt.Sprintf("Tags in fr: %v", frTags))
	require.NotNil(t, enTags["tag1"])
	require.NotNil(t, frTags["FRtag1"])
	b.AssertFileContent("public/fr/plaques/FRtag1/index.html", "FRtag1|Bonjour|http://example.com/blog/fr/plaques/FRtag1/")
	b.AssertFileContent("public/en/tags/tag1/index.html", "tag1|Hello|http://example.com/blog/en/tags/tag1/")

	// Check Blackfriday config
	require.True(t, strings.Contains(content(doc1fr), "&laquo;"), content(doc1fr))
	require.False(t, strings.Contains(content(doc1en), "&laquo;"), content(doc1en))
	require.True(t, strings.Contains(content(doc1en), "&ldquo;"), content(doc1en))

	// en and nn have custom site menus
	require.Len(t, frSite.Menus(), 0, "fr: "+configSuffix)
	require.Len(t, enSite.Menus(), 1, "en: "+configSuffix)
	require.Len(t, nnSite.Menus(), 1, "nn: "+configSuffix)

	require.Equal(t, "Home", enSite.Menus()["main"].ByName()[0].Name)
	require.Equal(t, "Heim", nnSite.Menus()["main"].ByName()[0].Name)

	// Issue #3108
	prevPage := enSite.RegularPages()[0].Prev()
	require.NotNil(t, prevPage)
	require.Equal(t, page.KindPage, prevPage.Kind())

	for {
		if prevPage == nil {
			break
		}
		require.Equal(t, page.KindPage, prevPage.Kind())
		prevPage = prevPage.Prev()
	}

	// Check bundles
	b.AssertFileContent("public/fr/bundles/b1/index.html", "RelPermalink: /blog/fr/bundles/b1/|")
	bundleFr := frSite.getPage(page.KindPage, "bundles/b1/index.md")
	require.NotNil(t, bundleFr)
	require.Equal(t, 1, len(bundleFr.Resources()))
	logoFr := bundleFr.Resources().GetMatch("logo*")
	require.NotNil(t, logoFr)
	b.AssertFileContent("public/fr/bundles/b1/index.html", "Resources: image/png: /blog/fr/bundles/b1/logo.png")
	b.AssertFileContent("public/fr/bundles/b1/logo.png", "PNG Data")

	bundleEn := enSite.getPage(page.KindPage, "bundles/b1/index.en.md")
	require.NotNil(t, bundleEn)
	b.AssertFileContent("public/en/bundles/b1/index.html", "RelPermalink: /blog/en/bundles/b1/|")
	require.Equal(t, 1, len(bundleEn.Resources()))
	logoEn := bundleEn.Resources().GetMatch("logo*")
	require.NotNil(t, logoEn)
	b.AssertFileContent("public/en/bundles/b1/index.html", "Resources: image/png: /blog/en/bundles/b1/logo.png")
	b.AssertFileContent("public/en/bundles/b1/logo.png", "PNG Data")

}

func TestMultiSitesRebuild(t *testing.T) {
	// t.Parallel() not supported, see https://github.com/fortytw2/leaktest/issues/4
	// This leaktest seems to be a little bit shaky on Travis.
	if !isCI() {
		defer leaktest.CheckTimeout(t, 10*time.Second)()
	}

	assert := require.New(t)

	b := newMultiSiteTestDefaultBuilder(t).Running().CreateSites().Build(BuildCfg{})

	sites := b.H.Sites
	fs := b.Fs

	b.AssertFileContent("public/en/sect/doc2/index.html", "Single: doc2|Hello|en|", "\n\n<h1 id=\"doc2\">doc2</h1>\n\n<p><em>some content</em>")

	enSite := sites[0]
	frSite := sites[1]

	assert.Len(enSite.RegularPages(), 5)
	assert.Len(frSite.RegularPages(), 4)

	// Verify translations
	b.AssertFileContent("public/en/sect/doc1-slug/index.html", "Hello")
	b.AssertFileContent("public/fr/sect/doc1/index.html", "Bonjour")

	// check single page content
	b.AssertFileContent("public/fr/sect/doc1/index.html", "Single", "Shortcode: Bonjour")
	b.AssertFileContent("public/en/sect/doc1-slug/index.html", "Single", "Shortcode: Hello")

	homeEn := enSite.getPage(page.KindHome)
	require.NotNil(t, homeEn)
	assert.Len(homeEn.Translations(), 3)

	contentFs := b.H.Fs.Source

	for i, this := range []struct {
		preFunc    func(t *testing.T)
		events     []fsnotify.Event
		assertFunc func(t *testing.T)
	}{
		// * Remove doc
		// * Add docs existing languages
		// (Add doc new language: TODO(bep) we should load config.toml as part of these so we can add languages).
		// * Rename file
		// * Change doc
		// * Change a template
		// * Change language file
		{
			func(t *testing.T) {
				fs.Source.Remove("content/sect/doc2.en.md")
			},
			[]fsnotify.Event{{Name: filepath.FromSlash("content/sect/doc2.en.md"), Op: fsnotify.Remove}},
			func(t *testing.T) {
				assert.Len(enSite.RegularPages(), 4, "1 en removed")

				// Check build stats
				require.Equal(t, 1, enSite.buildStats.draftCount, "Draft")
				require.Equal(t, 1, enSite.buildStats.futureCount, "Future")
				require.Equal(t, 1, enSite.buildStats.expiredCount, "Expired")
				require.Equal(t, 0, frSite.buildStats.draftCount, "Draft")
				require.Equal(t, 1, frSite.buildStats.futureCount, "Future")
				require.Equal(t, 1, frSite.buildStats.expiredCount, "Expired")
			},
		},
		{
			func(t *testing.T) {
				writeNewContentFile(t, contentFs, "new_en_1", "2016-07-31", "content/new1.en.md", -5)
				writeNewContentFile(t, contentFs, "new_en_2", "1989-07-30", "content/new2.en.md", -10)
				writeNewContentFile(t, contentFs, "new_fr_1", "2016-07-30", "content/new1.fr.md", 10)
			},
			[]fsnotify.Event{
				{Name: filepath.FromSlash("content/new1.en.md"), Op: fsnotify.Create},
				{Name: filepath.FromSlash("content/new2.en.md"), Op: fsnotify.Create},
				{Name: filepath.FromSlash("content/new1.fr.md"), Op: fsnotify.Create},
			},
			func(t *testing.T) {
				assert.Len(enSite.RegularPages(), 6)
				assert.Len(enSite.AllPages(), 34)
				assert.Len(frSite.RegularPages(), 5)
				require.Equal(t, "new_fr_1", frSite.RegularPages()[3].Title())
				require.Equal(t, "new_en_2", enSite.RegularPages()[0].Title())
				require.Equal(t, "new_en_1", enSite.RegularPages()[1].Title())

				rendered := readDestination(t, fs, "public/en/new1/index.html")
				require.True(t, strings.Contains(rendered, "new_en_1"), rendered)
			},
		},
		{
			func(t *testing.T) {
				p := "content/sect/doc1.en.md"
				doc1 := readFileFromFs(t, contentFs, p)
				doc1 += "CHANGED"
				writeToFs(t, contentFs, p, doc1)
			},
			[]fsnotify.Event{{Name: filepath.FromSlash("content/sect/doc1.en.md"), Op: fsnotify.Write}},
			func(t *testing.T) {
				assert.Len(enSite.RegularPages(), 6)
				doc1 := readDestination(t, fs, "public/en/sect/doc1-slug/index.html")
				require.True(t, strings.Contains(doc1, "CHANGED"), doc1)

			},
		},
		// Rename a file
		{
			func(t *testing.T) {
				if err := contentFs.Rename("content/new1.en.md", "content/new1renamed.en.md"); err != nil {
					t.Fatalf("Rename failed: %s", err)
				}
			},
			[]fsnotify.Event{
				{Name: filepath.FromSlash("content/new1renamed.en.md"), Op: fsnotify.Rename},
				{Name: filepath.FromSlash("content/new1.en.md"), Op: fsnotify.Rename},
			},
			func(t *testing.T) {
				assert.Len(enSite.RegularPages(), 6, "Rename")
				require.Equal(t, "new_en_1", enSite.RegularPages()[1].Title())
				rendered := readDestination(t, fs, "public/en/new1renamed/index.html")
				require.True(t, strings.Contains(rendered, "new_en_1"), rendered)
			}},
		{
			// Change a template
			func(t *testing.T) {
				template := "layouts/_default/single.html"
				templateContent := readSource(t, fs, template)
				templateContent += "{{ print \"Template Changed\"}}"
				writeSource(t, fs, template, templateContent)
			},
			[]fsnotify.Event{{Name: filepath.FromSlash("layouts/_default/single.html"), Op: fsnotify.Write}},
			func(t *testing.T) {
				assert.Len(enSite.RegularPages(), 6)
				assert.Len(enSite.AllPages(), 34)
				assert.Len(frSite.RegularPages(), 5)
				doc1 := readDestination(t, fs, "public/en/sect/doc1-slug/index.html")
				require.True(t, strings.Contains(doc1, "Template Changed"), doc1)
			},
		},
		{
			// Change a language file
			func(t *testing.T) {
				languageFile := "i18n/fr.yaml"
				langContent := readSource(t, fs, languageFile)
				langContent = strings.Replace(langContent, "Bonjour", "Salut", 1)
				writeSource(t, fs, languageFile, langContent)
			},
			[]fsnotify.Event{{Name: filepath.FromSlash("i18n/fr.yaml"), Op: fsnotify.Write}},
			func(t *testing.T) {
				assert.Len(enSite.RegularPages(), 6)
				assert.Len(enSite.AllPages(), 34)
				assert.Len(frSite.RegularPages(), 5)
				docEn := readDestination(t, fs, "public/en/sect/doc1-slug/index.html")
				require.True(t, strings.Contains(docEn, "Hello"), "No Hello")
				docFr := readDestination(t, fs, "public/fr/sect/doc1/index.html")
				require.True(t, strings.Contains(docFr, "Salut"), "No Salut")

				homeEn := enSite.getPage(page.KindHome)
				require.NotNil(t, homeEn)
				assert.Len(homeEn.Translations(), 3)
				require.Equal(t, "fr", homeEn.Translations()[0].Language().Lang)

			},
		},
		// Change a shortcode
		{
			func(t *testing.T) {
				writeSource(t, fs, "layouts/shortcodes/shortcode.html", "Modified Shortcode: {{ i18n \"hello\" }}")
			},
			[]fsnotify.Event{
				{Name: filepath.FromSlash("layouts/shortcodes/shortcode.html"), Op: fsnotify.Write},
			},
			func(t *testing.T) {
				assert.Len(enSite.RegularPages(), 6)
				assert.Len(enSite.AllPages(), 34)
				assert.Len(frSite.RegularPages(), 5)
				b.AssertFileContent("public/fr/sect/doc1/index.html", "Single", "Modified Shortcode: Salut")
				b.AssertFileContent("public/en/sect/doc1-slug/index.html", "Single", "Modified Shortcode: Hello")
			},
		},
	} {

		if this.preFunc != nil {
			this.preFunc(t)
		}

		err := b.H.Build(BuildCfg{}, this.events...)

		if err != nil {
			t.Fatalf("[%d] Failed to rebuild sites: %s", i, err)
		}

		this.assertFunc(t)
	}

}

func TestAddNewLanguage(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	b := newMultiSiteTestDefaultBuilder(t)
	b.CreateSites().Build(BuildCfg{})

	fs := b.Fs

	newConfig := multiSiteTOMLConfigTemplate + `

[Languages.sv]
weight = 15
title = "Svenska"
`

	writeNewContentFile(t, fs.Source, "Swedish Contentfile", "2016-01-01", "content/sect/doc1.sv.md", 10)
	// replace the config
	b.WithNewConfig(newConfig)

	sites := b.H

	assert.NoError(b.LoadConfig())
	err := b.H.Build(BuildCfg{NewConfig: b.Cfg})

	if err != nil {
		t.Fatalf("Failed to rebuild sites: %s", err)
	}

	require.Len(t, sites.Sites, 5, fmt.Sprintf("Len %d", len(sites.Sites)))

	// The Swedish site should be put in the middle (language weight=15)
	enSite := sites.Sites[0]
	svSite := sites.Sites[1]
	frSite := sites.Sites[2]
	require.True(t, enSite.language.Lang == "en", enSite.language.Lang)
	require.True(t, svSite.language.Lang == "sv", svSite.language.Lang)
	require.True(t, frSite.language.Lang == "fr", frSite.language.Lang)

	homeEn := enSite.getPage(page.KindHome)
	require.NotNil(t, homeEn)
	require.Len(t, homeEn.Translations(), 4)

	require.Equal(t, "sv", homeEn.Translations()[0].Language().Lang)

	require.Len(t, enSite.RegularPages(), 5)
	require.Len(t, frSite.RegularPages(), 4)

	// Veriy Swedish site
	require.Len(t, svSite.RegularPages(), 1)
	svPage := svSite.RegularPages()[0]

	require.Equal(t, "Swedish Contentfile", svPage.Title())
	require.Equal(t, "sv", svPage.Language().Lang)
	require.Len(t, svPage.Translations(), 2)
	require.Len(t, svPage.AllTranslations(), 3)
	require.Equal(t, "en", svPage.Translations()[0].Language().Lang)

	// Regular pages have no children
	require.Len(t, svPage.Pages(), 0)
	require.Len(t, svPage.Data().(page.Data).Pages(), 0)

}

// https://github.com/gohugoio/hugo/issues/4706
func TestContentStressTest(t *testing.T) {
	b := newTestSitesBuilder(t)

	numPages := 500

	contentTempl := `
---
%s
title: %q
weight: %d
multioutput: %t
---

# Header

CONTENT

The End.
`

	contentTempl = strings.Replace(contentTempl, "CONTENT", strings.Repeat(`
	
## Another header

Some text. Some more text.

`, 100), -1)

	var content []string
	defaultOutputs := `outputs: ["html", "json", "rss" ]`

	for i := 1; i <= numPages; i++ {
		outputs := defaultOutputs
		multioutput := true
		if i%3 == 0 {
			outputs = `outputs: ["json"]`
			multioutput = false
		}
		section := "s1"
		if i%10 == 0 {
			section = "s2"
		}
		content = append(content, []string{fmt.Sprintf("%s/page%d.md", section, i), fmt.Sprintf(contentTempl, outputs, fmt.Sprintf("Title %d", i), i, multioutput)}...)
	}

	content = append(content, []string{"_index.md", fmt.Sprintf(contentTempl, defaultOutputs, fmt.Sprintf("Home %d", 0), 0, true)}...)
	content = append(content, []string{"s1/_index.md", fmt.Sprintf(contentTempl, defaultOutputs, fmt.Sprintf("S %d", 1), 1, true)}...)
	content = append(content, []string{"s2/_index.md", fmt.Sprintf(contentTempl, defaultOutputs, fmt.Sprintf("S %d", 2), 2, true)}...)

	b.WithSimpleConfigFile()
	b.WithTemplates("layouts/_default/single.html", `Single: {{ .Content }}|RelPermalink: {{ .RelPermalink }}|Permalink: {{ .Permalink }}`)
	b.WithTemplates("layouts/_default/myview.html", `View: {{ len .Content }}`)
	b.WithTemplates("layouts/_default/single.json", `Single JSON: {{ .Content }}|RelPermalink: {{ .RelPermalink }}|Permalink: {{ .Permalink }}`)
	b.WithTemplates("layouts/_default/list.html", `
Page: {{ .Paginator.PageNumber }}
P: {{ with .File }}{{ path.Join .Path }}{{ end }}
List: {{ len .Paginator.Pages }}|List Content: {{ len .Content }}
{{ $shuffled :=  where .Site.RegularPages "Params.multioutput" true | shuffle }}
{{ $first5 := $shuffled | first 5 }}
L1: {{ len .Site.RegularPages }} L2: {{ len $first5 }}
{{ range $i, $e := $first5 }}
Render {{ $i }}: {{ .Render "myview" }}
{{ end }}
END
`)

	b.WithContent(content...)

	b.CreateSites().Build(BuildCfg{})

	contentMatchers := []string{"<h2 id=\"another-header\">Another header</h2>", "<h2 id=\"another-header-99\">Another header</h2>", "<p>The End.</p>"}

	for i := 1; i <= numPages; i++ {
		if i%3 != 0 {
			section := "s1"
			if i%10 == 0 {
				section = "s2"
			}
			checkContent(b, fmt.Sprintf("public/%s/page%d/index.html", section, i), contentMatchers...)
		}
	}

	for i := 1; i <= numPages; i++ {
		section := "s1"
		if i%10 == 0 {
			section = "s2"
		}
		checkContent(b, fmt.Sprintf("public/%s/page%d/index.json", section, i), contentMatchers...)
	}

	checkContent(b, "public/s1/index.html", "P: s1/_index.md\nList: 10|List Content: 8335\n\n\nL1: 500 L2: 5\n\nRender 0: View: 8335\n\nRender 1: View: 8335\n\nRender 2: View: 8335\n\nRender 3: View: 8335\n\nRender 4: View: 8335\n\nEND\n")
	checkContent(b, "public/s2/index.html", "P: s2/_index.md\nList: 10|List Content: 8335", "Render 4: View: 8335\n\nEND")
	checkContent(b, "public/index.html", "P: _index.md\nList: 10|List Content: 8335", "4: View: 8335\n\nEND")

	// Check paginated pages
	for i := 2; i <= 9; i++ {
		checkContent(b, fmt.Sprintf("public/page/%d/index.html", i), fmt.Sprintf("Page: %d", i), "Content: 8335\n\n\nL1: 500 L2: 5\n\nRender 0: View: 8335", "Render 4: View: 8335\n\nEND")
	}
}

func checkContent(s *sitesBuilder, filename string, matches ...string) {
	content := readDestination(s.T, s.Fs, filename)
	for _, match := range matches {
		if !strings.Contains(content, match) {
			s.Fatalf("No match for %q in content for %s\n%q", match, filename, content)
		}
	}

}

func TestTranslationsFromContentToNonContent(t *testing.T) {
	b := newTestSitesBuilder(t)
	b.WithConfigFile("toml", `

baseURL = "http://example.com/"

defaultContentLanguage = "en"

[languages]
[languages.en]
weight = 10
contentDir = "content/en"
[languages.nn]
weight = 20
contentDir = "content/nn"


`)

	b.WithContent("en/mysection/_index.md", `
---
Title: My Section
---

`)

	b.WithContent("en/_index.md", `
---
Title: My Home
---

`)

	b.WithContent("en/categories/mycat/_index.md", `
---
Title: My MyCat
---

`)

	b.WithContent("en/categories/_index.md", `
---
Title: My categories
---

`)

	for _, lang := range []string{"en", "nn"} {

		b.WithContent(lang+"/mysection/page.md", `
---
Title: My Page
categories: ["mycat"]
---

`)

	}

	b.Build(BuildCfg{})

	for _, path := range []string{
		"/",
		"/mysection",
		"/categories",
		"/categories/mycat",
	} {

		t.Run(path, func(t *testing.T) {
			assert := require.New(t)

			s1, _ := b.H.Sites[0].getPageNew(nil, path)
			s2, _ := b.H.Sites[1].getPageNew(nil, path)

			assert.NotNil(s1)
			assert.NotNil(s2)

			assert.Equal(1, len(s1.Translations()))
			assert.Equal(1, len(s2.Translations()))
			assert.Equal(s2, s1.Translations()[0])
			assert.Equal(s1, s2.Translations()[0])

			m1 := s1.Translations().MergeByLanguage(s2.Translations())
			m2 := s2.Translations().MergeByLanguage(s1.Translations())

			assert.Equal(1, len(m1))
			assert.Equal(1, len(m2))
		})

	}
}

// https://github.com/gohugoio/hugo/issues/5777
func TestTableOfContentsInShortcodes(t *testing.T) {
	t.Parallel()

	b := newMultiSiteTestDefaultBuilder(t)

	b.WithTemplatesAdded("layouts/shortcodes/toc.html", tocShortcode)
	b.WithTemplatesAdded("layouts/shortcodes/wrapper.html", "{{ .Inner }}")
	b.WithContent("post/simple.en.md", tocPageSimple)
	b.WithContent("post/variants1.en.md", tocPageVariants1)
	b.WithContent("post/variants2.en.md", tocPageVariants2)

	b.WithContent("post/withSCInHeading.en.md", tocPageWithShortcodesInHeadings)

	b.CreateSites().Build(BuildCfg{})

	b.AssertFileContent("public/en/post/simple/index.html",
		tocPageSimpleExpected,
		// Make sure it is inserted twice
		`TOC1: <nav id="TableOfContents">`,
		`TOC2: <nav id="TableOfContents">`,
	)

	b.AssertFileContentFn("public/en/post/variants1/index.html", func(s string) bool {
		return strings.Count(s, "TableOfContents") == 4
	})
	b.AssertFileContentFn("public/en/post/variants2/index.html", func(s string) bool {
		return strings.Count(s, "TableOfContents") == 6
	})

	b.AssertFileContent("public/en/post/withSCInHeading/index.html", tocPageWithShortcodesInHeadingsExpected)
}

var tocShortcode = `
TOC1: {{ .Page.TableOfContents }}

TOC2: {{ .Page.TableOfContents }}
`

func TestSelfReferencedContentInShortcode(t *testing.T) {
	t.Parallel()

	b := newMultiSiteTestDefaultBuilder(t)

	var (
		shortcode = `{{- .Page.Content -}}{{- .Page.Summary -}}{{- .Page.Plain -}}{{- .Page.PlainWords -}}{{- .Page.WordCount -}}{{- .Page.ReadingTime -}}`

		page = `---
title: sctest
---
Empty:{{< mycontent >}}:
`
	)

	b.WithTemplatesAdded("layouts/shortcodes/mycontent.html", shortcode)
	b.WithContent("post/simple.en.md", page)

	b.CreateSites().Build(BuildCfg{})

	b.AssertFileContent("public/en/post/simple/index.html", "Empty:[]00:")
}

var tocPageSimple = `---
title: tocTest
publishdate: "2000-01-01"
---
{{< toc >}}
# Heading 1 {#1}
Some text.
## Subheading 1.1 {#1-1}
Some more text.
# Heading 2 {#2}
Even more text.
## Subheading 2.1 {#2-1}
Lorem ipsum...
`

var tocPageVariants1 = `---
title: tocTest
publishdate: "2000-01-01"
---
Variant 1:
{{% wrapper %}}
{{< toc >}}
{{% /wrapper %}}
# Heading 1

Variant 3:
{{% toc %}}

`

var tocPageVariants2 = `---
title: tocTest
publishdate: "2000-01-01"
---
Variant 1:
{{% wrapper %}}
{{< toc >}}
{{% /wrapper %}}
# Heading 1

Variant 2:
{{< wrapper >}}
{{< toc >}}
{{< /wrapper >}}

Variant 3:
{{% toc %}}

`

var tocPageSimpleExpected = `<nav id="TableOfContents">
<ul>
<li><a href="#1">Heading 1</a>
<ul>
<li><a href="#1-1">Subheading 1.1</a></li>
</ul></li>
<li><a href="#2">Heading 2</a>
<ul>
<li><a href="#2-1">Subheading 2.1</a></li>
</ul></li>
</ul>
</nav>`

var tocPageWithShortcodesInHeadings = `---
title: tocTest
publishdate: "2000-01-01"
---

{{< toc >}}

# Heading 1 {#1}

Some text.

## Subheading 1.1 {{< shortcode >}} {#1-1}

Some more text.

# Heading 2 {{% shortcode %}} {#2}

Even more text.

## Subheading 2.1 {#2-1}

Lorem ipsum...
`

var tocPageWithShortcodesInHeadingsExpected = `<nav id="TableOfContents">
<ul>
<li><a href="#1">Heading 1</a>
<ul>
<li><a href="#1-1">Subheading 1.1 Shortcode: Hello</a></li>
</ul></li>
<li><a href="#2">Heading 2 Shortcode: Hello</a>
<ul>
<li><a href="#2-1">Subheading 2.1</a></li>
</ul></li>
</ul>
</nav>`

var multiSiteTOMLConfigTemplate = `
baseURL = "http://example.com/blog"

paginate = 1
disablePathToLower = true
defaultContentLanguage = "{{ .DefaultContentLanguage }}"
defaultContentLanguageInSubdir = {{ .DefaultContentLanguageInSubdir }}
enableRobotsTXT = true

[permalinks]
other = "/somewhere/else/:filename"

[blackfriday]
angledQuotes = true

[Taxonomies]
tag = "tags"

[Languages]
[Languages.en]
weight = 10
title = "In English"
languageName = "English"
[Languages.en.blackfriday]
angledQuotes = false
[[Languages.en.menu.main]]
url    = "/"
name   = "Home"
weight = 0

[Languages.fr]
weight = 20
title = "Le Français"
languageName = "Français"
[Languages.fr.Taxonomies]
plaque = "plaques"

[Languages.nn]
weight = 30
title = "På nynorsk"
languageName = "Nynorsk"
paginatePath = "side"
[Languages.nn.Taxonomies]
lag = "lag"
[[Languages.nn.menu.main]]
url    = "/"
name   = "Heim"
weight = 1

[Languages.nb]
weight = 40
title = "På bokmål"
languageName = "Bokmål"
paginatePath = "side"
[Languages.nb.Taxonomies]
lag = "lag"
`

var multiSiteYAMLConfigTemplate = `
baseURL: "http://example.com/blog"

disablePathToLower: true
paginate: 1
defaultContentLanguage: "{{ .DefaultContentLanguage }}"
defaultContentLanguageInSubdir: {{ .DefaultContentLanguageInSubdir }}
enableRobotsTXT: true

permalinks:
    other: "/somewhere/else/:filename"

blackfriday:
    angledQuotes: true

Taxonomies:
    tag: "tags"

Languages:
    en:
        weight: 10
        title: "In English"
        languageName: "English"
        blackfriday:
            angledQuotes: false
        menu:
            main:
                - url: "/"
                  name: "Home"
                  weight: 0
    fr:
        weight: 20
        title: "Le Français"
        languageName: "Français"
        Taxonomies:
            plaque: "plaques"
    nn:
        weight: 30
        title: "På nynorsk"
        languageName: "Nynorsk"
        paginatePath: "side"
        Taxonomies:
            lag: "lag"
        menu:
            main:
                - url: "/"
                  name: "Heim"
                  weight: 1
    nb:
        weight: 40
        title: "På bokmål"
        languageName: "Bokmål"
        paginatePath: "side"
        Taxonomies:
            lag: "lag"

`

// TODO(bep) clean move
var multiSiteJSONConfigTemplate = `
{
  "baseURL": "http://example.com/blog",
  "paginate": 1,
  "disablePathToLower": true,
  "defaultContentLanguage": "{{ .DefaultContentLanguage }}",
  "defaultContentLanguageInSubdir": true,
  "enableRobotsTXT": true,
  "permalinks": {
    "other": "/somewhere/else/:filename"
  },
  "blackfriday": {
    "angledQuotes": true
  },
  "Taxonomies": {
    "tag": "tags"
  },
  "Languages": {
    "en": {
      "weight": 10,
      "title": "In English",
      "languageName": "English",
      "blackfriday": {
        "angledQuotes": false
      },
	  "menu": {
        "main": [
			{
			"url": "/",
			"name": "Home",
			"weight": 0
			}
		]
      }
    },
    "fr": {
      "weight": 20,
      "title": "Le Français",
      "languageName": "Français",
      "Taxonomies": {
        "plaque": "plaques"
      }
    },
    "nn": {
      "weight": 30,
      "title": "På nynorsk",
      "paginatePath": "side",
      "languageName": "Nynorsk",
      "Taxonomies": {
        "lag": "lag"
      },
	  "menu": {
        "main": [
			{
        	"url": "/",
			"name": "Heim",
			"weight": 1
			}
      	]
      }
    },
    "nb": {
      "weight": 40,
      "title": "På bokmål",
      "paginatePath": "side",
      "languageName": "Bokmål",
      "Taxonomies": {
        "lag": "lag"
      }
    }
  }
}
`

func writeSource(t testing.TB, fs *hugofs.Fs, filename, content string) {
	t.Helper()
	writeToFs(t, fs.Source, filename, content)
}

func writeToFs(t testing.TB, fs afero.Fs, filename, content string) {
	t.Helper()
	if err := afero.WriteFile(fs, filepath.FromSlash(filename), []byte(content), 0755); err != nil {
		t.Fatalf("Failed to write file: %s", err)
	}
}

func readDestination(t testing.TB, fs *hugofs.Fs, filename string) string {
	t.Helper()
	return readFileFromFs(t, fs.Destination, filename)
}

func destinationExists(fs *hugofs.Fs, filename string) bool {
	b, err := helpers.Exists(filename, fs.Destination)
	if err != nil {
		panic(err)
	}
	return b
}

func readSource(t *testing.T, fs *hugofs.Fs, filename string) string {
	return readFileFromFs(t, fs.Source, filename)
}

func readFileFromFs(t testing.TB, fs afero.Fs, filename string) string {
	t.Helper()
	filename = filepath.Clean(filename)
	b, err := afero.ReadFile(fs, filename)
	if err != nil {
		// Print some debug info
		hadSlash := strings.HasPrefix(filename, helpers.FilePathSeparator)
		start := 0
		if hadSlash {
			start = 1
		}
		end := start + 1

		parts := strings.Split(filename, helpers.FilePathSeparator)
		if parts[start] == "work" {
			end++
		}

		root := filepath.Join(parts[start:end]...)
		if hadSlash {
			root = helpers.FilePathSeparator + root
		}

		//helpers.PrintFs(fs, root, os.Stdout)
		t.Fatalf("Failed to read file: %s", err)
	}
	return string(b)
}

const testPageTemplate = `---
title: "%s"
publishdate: "%s"
weight: %d
---
# Doc %s
`

func newTestPage(title, date string, weight int) string {
	return fmt.Sprintf(testPageTemplate, title, date, weight, title)
}

func writeNewContentFile(t *testing.T, fs afero.Fs, title, date, filename string, weight int) {
	content := newTestPage(title, date, weight)
	writeToFs(t, fs, filename, content)
}

type multiSiteTestBuilder struct {
	configData   interface{}
	config       string
	configFormat string

	*sitesBuilder
}

func newMultiSiteTestDefaultBuilder(t testing.TB) *multiSiteTestBuilder {
	return newMultiSiteTestBuilder(t, "", "", nil)
}

func (b *multiSiteTestBuilder) WithNewConfig(config string) *multiSiteTestBuilder {
	b.WithConfigTemplate(b.configData, b.configFormat, config)
	return b
}

func (b *multiSiteTestBuilder) WithNewConfigData(data interface{}) *multiSiteTestBuilder {
	b.WithConfigTemplate(data, b.configFormat, b.config)
	return b
}

func newMultiSiteTestBuilder(t testing.TB, configFormat, config string, configData interface{}) *multiSiteTestBuilder {
	if configData == nil {
		configData = map[string]interface{}{
			"DefaultContentLanguage":         "fr",
			"DefaultContentLanguageInSubdir": true,
		}
	}

	if config == "" {
		config = multiSiteTOMLConfigTemplate
	}

	if configFormat == "" {
		configFormat = "toml"
	}

	b := newTestSitesBuilder(t).WithConfigTemplate(configData, configFormat, config)
	b.WithContent("root.en.md", `---
title: root
weight: 10000
slug: root
publishdate: "2000-01-01"
---
# root
`,
		"sect/doc1.en.md", `---
title: doc1
weight: 1
slug: doc1-slug
tags:
 - tag1
publishdate: "2000-01-01"
---
# doc1
*some "content"*

{{< shortcode >}}

{{< lingo >}}

NOTE: slug should be used as URL
`,
		"sect/doc1.fr.md", `---
title: doc1
weight: 1
plaques:
 - FRtag1
 - FRtag2
publishdate: "2000-01-04"
---
# doc1
*quelque "contenu"*

{{< shortcode >}}

{{< lingo >}}

NOTE: should be in the 'en' Page's 'Translations' field.
NOTE: date is after "doc3"
`,
		"sect/doc2.en.md", `---
title: doc2
weight: 2
publishdate: "2000-01-02"
---
# doc2
*some content*
NOTE: without slug, "doc2" should be used, without ".en" as URL
`,
		"sect/doc3.en.md", `---
title: doc3
weight: 3
publishdate: "2000-01-03"
aliases: [/en/al/alias1,/al/alias2/]
tags:
 - tag2
 - tag1
url: /superbob/
---
# doc3
*some content*
NOTE: third 'en' doc, should trigger pagination on home page.
`,
		"sect/doc4.md", `---
title: doc4
weight: 4
plaques:
 - FRtag1
publishdate: "2000-01-05"
---
# doc4
*du contenu francophone*
NOTE: should use the defaultContentLanguage and mark this doc as 'fr'.
NOTE: doesn't have any corresponding translation in 'en'
`,
		"other/doc5.fr.md", `---
title: doc5
weight: 5
publishdate: "2000-01-06"
---
# doc5
*autre contenu francophone*
NOTE: should use the "permalinks" configuration with :filename
`,
		// Add some for the stats
		"stats/expired.fr.md", `---
title: expired
publishdate: "2000-01-06"
expiryDate: "2001-01-06"
---
# Expired
`,
		"stats/future.fr.md", `---
title: future
weight: 6
publishdate: "2100-01-06"
---
# Future
`,
		"stats/expired.en.md", `---
title: expired
weight: 7
publishdate: "2000-01-06"
expiryDate: "2001-01-06"
---
# Expired
`,
		"stats/future.en.md", `---
title: future
weight: 6
publishdate: "2100-01-06"
---
# Future
`,
		"stats/draft.en.md", `---
title: expired
publishdate: "2000-01-06"
draft: true
---
# Draft
`,
		"stats/tax.nn.md", `---
title: Tax NN
weight: 8
publishdate: "2000-01-06"
weight: 1001
lag:
- Sogndal
---
# Tax NN
`,
		"stats/tax.nb.md", `---
title: Tax NB
weight: 8
publishdate: "2000-01-06"
weight: 1002
lag:
- Sogndal
---
# Tax NB
`,
		// Bundle
		"bundles/b1/index.en.md", `---
title: Bundle EN
publishdate: "2000-01-06"
weight: 2001
---
# Bundle Content EN
`,
		"bundles/b1/index.md", `---
title: Bundle Default
publishdate: "2000-01-06"
weight: 2002
---
# Bundle Content Default
`,
		"bundles/b1/logo.png", `
PNG Data
`)

	return &multiSiteTestBuilder{sitesBuilder: b, configFormat: configFormat, config: config, configData: configData}
}
