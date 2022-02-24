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

// Package goldmark converts Markdown to HTML using Goldmark.
package goldmark

import (
	"bytes"
	"fmt"
	"path/filepath"
	"runtime/debug"

	"github.com/gohugoio/hugo/markup/goldmark/codeblocks"
	"github.com/gohugoio/hugo/markup/goldmark/internal/extensions/attributes"
	"github.com/gohugoio/hugo/markup/goldmark/internal/render"

	"github.com/gohugoio/hugo/identity"

	"github.com/pkg/errors"

	"github.com/spf13/afero"

	"github.com/gohugoio/hugo/hugofs"
	"github.com/gohugoio/hugo/markup/converter"
	"github.com/gohugoio/hugo/markup/tableofcontents"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

// Provider is the package entry point.
var Provider converter.ProviderProvider = provide{}

type provide struct{}

func (p provide) New(cfg converter.ProviderConfig) (converter.Provider, error) {
	md := newMarkdown(cfg)

	return converter.NewProvider("goldmark", func(ctx converter.DocumentContext) (converter.Converter, error) {
		return &goldmarkConverter{
			ctx: ctx,
			cfg: cfg,
			md:  md,
			sanitizeAnchorName: func(s string) string {
				return sanitizeAnchorNameString(s, cfg.MarkupConfig.Goldmark.Parser.AutoHeadingIDType)
			},
		}, nil
	}), nil
}

var _ converter.AnchorNameSanitizer = (*goldmarkConverter)(nil)

type goldmarkConverter struct {
	md  goldmark.Markdown
	ctx converter.DocumentContext
	cfg converter.ProviderConfig

	sanitizeAnchorName func(s string) string
}

func (c *goldmarkConverter) SanitizeAnchorName(s string) string {
	return c.sanitizeAnchorName(s)
}

func newMarkdown(pcfg converter.ProviderConfig) goldmark.Markdown {
	mcfg := pcfg.MarkupConfig
	cfg := pcfg.MarkupConfig.Goldmark
	var rendererOptions []renderer.Option

	if cfg.Renderer.HardWraps {
		rendererOptions = append(rendererOptions, html.WithHardWraps())
	}

	if cfg.Renderer.XHTML {
		rendererOptions = append(rendererOptions, html.WithXHTML())
	}

	if cfg.Renderer.Unsafe {
		rendererOptions = append(rendererOptions, html.WithUnsafe())
	}

	var (
		extensions = []goldmark.Extender{
			newLinks(),
			newTocExtension(rendererOptions),
		}
		parserOptions []parser.Option
	)

	if mcfg.Highlight.CodeFences {
		extensions = append(extensions, codeblocks.New())
	}

	if cfg.Extensions.Table {
		extensions = append(extensions, extension.Table)
	}

	if cfg.Extensions.Strikethrough {
		extensions = append(extensions, extension.Strikethrough)
	}

	if cfg.Extensions.Linkify {
		extensions = append(extensions, extension.Linkify)
	}

	if cfg.Extensions.TaskList {
		extensions = append(extensions, extension.TaskList)
	}

	if cfg.Extensions.Typographer {
		extensions = append(extensions, extension.Typographer)
	}

	if cfg.Extensions.DefinitionList {
		extensions = append(extensions, extension.DefinitionList)
	}

	if cfg.Extensions.Footnote {
		extensions = append(extensions, extension.Footnote)
	}

	if cfg.Parser.AutoHeadingID {
		parserOptions = append(parserOptions, parser.WithAutoHeadingID())
	}

	if cfg.Parser.Attribute.Title {
		parserOptions = append(parserOptions, parser.WithAttribute())
	}

	if cfg.Parser.Attribute.Block {
		extensions = append(extensions, attributes.New())
	}

	md := goldmark.New(
		goldmark.WithExtensions(
			extensions...,
		),
		goldmark.WithParserOptions(
			parserOptions...,
		),
		goldmark.WithRendererOptions(
			rendererOptions...,
		),
	)

	return md
}

var _ identity.IdentitiesProvider = (*converterResult)(nil)

type converterResult struct {
	converter.Result
	toc tableofcontents.Root
	ids identity.Identities
}

func (c converterResult) TableOfContents() tableofcontents.Root {
	return c.toc
}

func (c converterResult) GetIdentities() identity.Identities {
	return c.ids
}

var converterIdentity = identity.KeyValueIdentity{Key: "goldmark", Value: "converter"}

func (c *goldmarkConverter) Convert(ctx converter.RenderContext) (result converter.Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			dir := afero.GetTempDir(hugofs.Os, "hugo_bugs")
			name := fmt.Sprintf("goldmark_%s.txt", c.ctx.DocumentID)
			filename := filepath.Join(dir, name)
			afero.WriteFile(hugofs.Os, filename, ctx.Src, 07555)
			fmt.Print(string(debug.Stack()))
			err = errors.Errorf("[BUG] goldmark: %s: create an issue on GitHub attaching the file in: %s", r, filename)
		}
	}()

	buf := &render.BufWriter{Buffer: &bytes.Buffer{}}
	result = buf
	pctx := c.newParserContext(ctx)
	reader := text.NewReader(ctx.Src)

	doc := c.md.Parser().Parse(
		reader,
		parser.WithContext(pctx),
	)

	rcx := &render.RenderContextDataHolder{
		Rctx: ctx,
		Dctx: c.ctx,
		IDs:  identity.NewManager(converterIdentity),
	}

	w := &render.Context{
		BufWriter:   buf,
		ContextData: rcx,
	}

	if err := c.md.Renderer().Render(w, ctx.Src, doc); err != nil {
		return nil, err
	}

	return converterResult{
		Result: buf,
		ids:    rcx.IDs.GetIdentities(),
		toc:    pctx.TableOfContents(),
	}, nil
}

var featureSet = map[identity.Identity]bool{
	converter.FeatureRenderHooks: true,
}

func (c *goldmarkConverter) Supports(feature identity.Identity) bool {
	return featureSet[feature.GetIdentity()]
}

func (c *goldmarkConverter) newParserContext(rctx converter.RenderContext) *parserContext {
	ctx := parser.NewContext(parser.WithIDs(newIDFactory(c.cfg.MarkupConfig.Goldmark.Parser.AutoHeadingIDType)))
	ctx.Set(tocEnableKey, rctx.RenderTOC)
	return &parserContext{
		Context: ctx,
	}
}

type parserContext struct {
	parser.Context
}

func (p *parserContext) TableOfContents() tableofcontents.Root {
	if v := p.Get(tocResultKey); v != nil {
		return v.(tableofcontents.Root)
	}
	return tableofcontents.Root{}
}
