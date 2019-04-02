// Copyright 2017 The Hugo Authors. All rights reserved.
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

// Package partials provides template functions for working with reusable
// templates.
package partials

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"strings"
	"sync"
	texttemplate "text/template"

	"github.com/gohugoio/hugo/tpl"

	bp "github.com/gohugoio/hugo/bufferpool"
	"github.com/gohugoio/hugo/deps"
)

// TestTemplateProvider is global deps.ResourceProvider.
// NOTE: It's currently unused.
var TestTemplateProvider deps.ResourceProvider

// partialCache represents a cache of partials protected by a mutex.
type partialCache struct {
	sync.RWMutex
	p map[string]interface{}
}

func (p *partialCache) clear() {
	p.Lock()
	defer p.Unlock()
	p.p = make(map[string]interface{})
}

// New returns a new instance of the templates-namespaced template functions.
func New(deps *deps.Deps) *Namespace {
	cache := &partialCache{p: make(map[string]interface{})}
	deps.BuildStartListeners.Add(
		func() {
			cache.clear()
		})

	return &Namespace{
		deps:           deps,
		cachedPartials: cache,
	}
}

// Namespace provides template functions for the "templates" namespace.
type Namespace struct {
	deps           *deps.Deps
	cachedPartials *partialCache
}

// contextWrapper makes room for a return value in a partial invocation.
type contextWrapper struct {
	Arg    interface{}
	Result interface{}
}

// Set sets the return value and returns an empty string.
func (c *contextWrapper) Set(in interface{}) string {
	c.Result = in
	return ""
}

// Include executes the named partial.
// If the partial contains a return statement, that value will be returned.
// Else, the rendered output will be returned:
// A string if the partial is a text/template, or template.HTML when html/template.
func (ns *Namespace) Include(name string, contextList ...interface{}) (interface{}, error) {
	if strings.HasPrefix(name, "partials/") {
		name = name[8:]
	}
	var context interface{}

	if len(contextList) == 0 {
		context = nil
	} else {
		context = contextList[0]
	}

	n := "partials/" + name
	templ, found := ns.deps.Tmpl.Lookup(n)

	if !found {
		// For legacy reasons.
		templ, found = ns.deps.Tmpl.Lookup(n + ".html")
	}

	if !found {
		return "", fmt.Errorf("partial %q not found", name)
	}

	var info tpl.Info
	if ip, ok := templ.(tpl.TemplateInfoProvider); ok {
		info = ip.TemplateInfo()
	}

	var w io.Writer

	if info.HasReturn {
		// Wrap the context sent to the template to capture the return value.
		// Note that the template is rewritten to make sure that the dot (".")
		// and the $ variable points to Arg.
		context = &contextWrapper{
			Arg: context,
		}

		// We don't care about any template output.
		w = ioutil.Discard
	} else {
		b := bp.GetBuffer()
		defer bp.PutBuffer(b)
		w = b
	}

	if err := templ.Execute(w, context); err != nil {
		return "", err
	}

	var result interface{}

	if ctx, ok := context.(*contextWrapper); ok {
		result = ctx.Result
	} else if _, ok := templ.(*texttemplate.Template); ok {
		result = w.(fmt.Stringer).String()
	} else {
		result = template.HTML(w.(fmt.Stringer).String())
	}

	if ns.deps.Metrics != nil {
		ns.deps.Metrics.TrackValue(n, result)
	}

	return result, nil

}

// IncludeCached executes and caches partial templates.  An optional variant
// string parameter (a string slice actually, but be only use a variadic
// argument to make it optional) can be passed so that a given partial can have
// multiple uses. The cache is created with name+variant as the key.
func (ns *Namespace) IncludeCached(name string, context interface{}, variant ...string) (interface{}, error) {
	key := name
	if len(variant) > 0 {
		for i := 0; i < len(variant); i++ {
			key += variant[i]
		}
	}
	return ns.getOrCreate(key, name, context)
}

func (ns *Namespace) getOrCreate(key, name string, context interface{}) (interface{}, error) {

	ns.cachedPartials.RLock()
	p, ok := ns.cachedPartials.p[key]
	ns.cachedPartials.RUnlock()

	if ok {
		return p, nil
	}

	p, err := ns.Include(name, context)
	if err != nil {
		return nil, err
	}

	ns.cachedPartials.Lock()
	defer ns.cachedPartials.Unlock()
	// Double-check.
	if p2, ok := ns.cachedPartials.p[key]; ok {
		return p2, nil
	}
	ns.cachedPartials.p[key] = p

	return p, nil
}
