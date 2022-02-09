// Copyright 2021 The Hugo Authors. All rights reserved.
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

package js_test

import (
	"path/filepath"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/gohugoio/hugo/htesting"
	"github.com/gohugoio/hugo/hugolib"
)

func TestBuildVariants(t *testing.T) {
	c := qt.New(t)

	mainWithImport := `
-- config.toml --
disableKinds=["page", "section", "taxonomy", "term", "sitemap", "robotsTXT"]
-- assets/js/main.js --
import { hello1, hello2 } from './util1';
hello1();
hello2();
-- assets/js/util1.js --
import { hello3 } from './util2';
export function hello1() {
	return 'abcd';
}
export function hello2() {
	return hello3();
}
-- assets/js/util2.js --
export function hello3() {
	return 'efgh';
}
-- layouts/index.html --
{{ $js := resources.Get "js/main.js" | js.Build }}
JS Content:{{ $js.Content }}:End:
	
			`

	c.Run("Basic", func(c *qt.C) {
		b := hugolib.NewIntegrationTestBuilder(hugolib.IntegrationTestConfig{T: c, NeedsOsFS: true, TxtarString: mainWithImport}).Build()

		b.AssertFileContent("public/index.html", `abcd`)
	})

	c.Run("Edit Import", func(c *qt.C) {
		b := hugolib.NewIntegrationTestBuilder(hugolib.IntegrationTestConfig{T: c, Running: true, NeedsOsFS: true, TxtarString: mainWithImport}).Build()

		b.AssertFileContent("public/index.html", `abcd`)
		b.EditFileReplace("assets/js/util1.js", func(s string) string { return strings.ReplaceAll(s, "abcd", "1234") }).Build()
		b.AssertFileContent("public/index.html", `1234`)
	})

	c.Run("Edit Import Nested", func(c *qt.C) {
		b := hugolib.NewIntegrationTestBuilder(hugolib.IntegrationTestConfig{T: c, Running: true, NeedsOsFS: true, TxtarString: mainWithImport}).Build()

		b.AssertFileContent("public/index.html", `efgh`)
		b.EditFileReplace("assets/js/util2.js", func(s string) string { return strings.ReplaceAll(s, "efgh", "1234") }).Build()
		b.AssertFileContent("public/index.html", `1234`)
	})
}

func TestBuildWithModAndNpm(t *testing.T) {
	if !htesting.IsCI() {
		t.Skip("skip (relative) long running modules test when running locally")
	}

	c := qt.New(t)

	files := `
-- config.toml --
baseURL = "https://example.org"
disableKinds=["page", "section", "taxonomy", "term", "sitemap", "robotsTXT"]
[module]
[[module.imports]]
path="github.com/gohugoio/hugoTestProjectJSModImports"
-- go.mod --
module github.com/gohugoio/tests/testHugoModules
		
go 1.16
		
require github.com/gohugoio/hugoTestProjectJSModImports v0.9.0 // indirect
-- package.json --
{
	"dependencies": {
	"date-fns": "^2.16.1"
	}
}
	
`
	b := hugolib.NewIntegrationTestBuilder(
		hugolib.IntegrationTestConfig{
			T:               c,
			NeedsOsFS:       true,
			NeedsNpmInstall: true,
			TxtarString:     files,
			Verbose:         true,
		}).Build()

	b.AssertFileContent("public/js/main.js", `
greeting: "greeting configured in mod2"
Hello1 from mod1: $
return "Hello2 from mod1";
var Hugo = "Rocks!";
Hello3 from mod2. Date from date-fns: ${today}
Hello from lib in the main project
Hello5 from mod2.
var myparam = "Hugo Rocks!";
shim cwd
`)

	// React JSX, verify the shimming.
	b.AssertFileContent("public/js/like.js", filepath.FromSlash(`@v0.9.0/assets/js/shims/react.js
module.exports = window.ReactDOM;
`))
}

func TestBuildWithNpm(t *testing.T) {
	if !htesting.IsCI() {
		t.Skip("skip (relative) long running modules test when running locally")
	}

	c := qt.New(t)

	files := `
-- assets/js/included.js --
console.log("included");
-- assets/js/main.js --
import "./included";
	import { toCamelCase } from "to-camel-case";
	
	console.log("main");
	console.log("To camel:", toCamelCase("space case"));
-- assets/js/myjsx.jsx --
import * as React from 'react'
import * as ReactDOM from 'react-dom'

	ReactDOM.render(
	<h1>Hello, world!</h1>,
	document.getElementById('root')
	);
-- assets/js/myts.ts --
function greeter(person: string) {
	return "Hello, " + person;
}
let user = [0, 1, 2];
document.body.textContent = greeter(user);
-- config.toml --
disablekinds = ['taxonomy', 'term', 'page']
-- content/p1.md --
Content.
-- data/hugo.toml --
slogan = "Hugo Rocks!"
-- i18n/en.yaml --
hello:
   other: "Hello"
-- i18n/fr.yaml --
hello:
   other: "Bonjour"
-- layouts/index.html --
{{ $options := dict "minify" false "externals" (slice "react" "react-dom") }}
{{ $js := resources.Get "js/main.js" | js.Build $options }}
JS:  {{ template "print" $js }}
{{ $jsx := resources.Get "js/myjsx.jsx" | js.Build $options }}
JSX: {{ template "print" $jsx }}
{{ $ts := resources.Get "js/myts.ts" | js.Build (dict "sourcemap" "inline")}}
TS: {{ template "print" $ts }}
{{ $ts2 := resources.Get "js/myts.ts" | js.Build (dict "sourcemap" "external" "TargetPath" "js/myts2.js")}}
TS2: {{ template "print" $ts2 }}
{{ define "print" }}RelPermalink: {{.RelPermalink}}|MIME: {{ .MediaType }}|Content: {{ .Content | safeJS }}{{ end }}
-- package.json --
{
	"scripts": {},

	"dependencies": {
	"to-camel-case": "1.0.0"
	}
}
`

	b := hugolib.NewIntegrationTestBuilder(
		hugolib.IntegrationTestConfig{
			T:               c,
			NeedsOsFS:       true,
			NeedsNpmInstall: true,
			TxtarString:     files,
		}).Build()

	b.AssertFileContent("public/js/myts.js", `//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJz`)
	b.AssertFileContent("public/js/myts2.js.map", `"version": 3,`)
	b.AssertFileContent("public/index.html", `
		console.log(&#34;included&#34;);
		if (hasSpace.test(string))
		var React = __toESM(__require(&#34;react&#34;));
		function greeter(person) {
`)
}
