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

package tplimpl

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"io/ioutil"
	"log"
	"os"

	"github.com/spf13/afero"
	"github.com/spf13/hugo/config"
	"github.com/spf13/hugo/deps"
	"github.com/spf13/hugo/helpers"
	"github.com/spf13/hugo/hugofs"
	"github.com/spf13/hugo/i18n"
	"github.com/spf13/hugo/tpl"
	"github.com/spf13/hugo/tpl/internal"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

var (
	logger = jww.NewNotepad(jww.LevelFatal, jww.LevelFatal, os.Stdout, ioutil.Discard, "", log.Ldate|log.Ltime)
)

func newDepsConfig(cfg config.Provider) deps.DepsCfg {
	l := helpers.NewLanguage("en", cfg)
	l.Set("i18nDir", "i18n")
	return deps.DepsCfg{
		Language:            l,
		Cfg:                 cfg,
		Fs:                  hugofs.NewMem(l),
		Logger:              logger,
		TemplateProvider:    DefaultTemplateProvider,
		TranslationProvider: i18n.NewTranslationProvider(),
	}
}

func TestTemplateFuncsExamples(t *testing.T) {
	t.Parallel()

	workingDir := "/home/hugo"

	v := viper.New()

	v.Set("workingDir", workingDir)
	v.Set("multilingual", true)

	fs := hugofs.NewMem(v)

	afero.WriteFile(fs.Source, filepath.Join(workingDir, "README.txt"), []byte("Hugo Rocks!"), 0755)

	d, err := deps.New(newDepsConfig(v))
	require.NoError(t, err)

	var data struct {
		Title   string
		Section string
		Params  map[string]interface{}
	}

	data.Title = "**BatMan**"
	data.Section = "blog"
	data.Params = map[string]interface{}{"langCode": "en"}

	for _, nsf := range internal.TemplateFuncsNamespaceRegistry {
		ns := nsf(d)
		for i, example := range ns.Examples {
			in, expected := example[0], example[1]
			d.WithTemplate = func(templ tpl.TemplateHandler) error {
				require.NoError(t, templ.AddTemplate("test", in))
				return nil
			}
			require.NoError(t, d.LoadResources())

			var b bytes.Buffer
			require.NoError(t, d.Tmpl.Lookup("test").Execute(&b, &data))
			if b.String() != expected {
				t.Fatalf("%s[%d]: got %q expected %q", ns.Name, i, b.String(), expected)
			}
		}
	}

}

func TestFuncsInTemplate(t *testing.T) {
	t.Parallel()

	workingDir := "/home/hugo"

	v := viper.New()

	v.Set("workingDir", workingDir)
	v.Set("multilingual", true)

	fs := hugofs.NewMem(v)

	afero.WriteFile(fs.Source, filepath.Join(workingDir, "README.txt"), []byte("Hugo Rocks!"), 0755)

	// Add the examples from the docs: As a smoke test and to make sure the examples work.
	// TODO(bep): docs: fix title example
	// TODO(bep) namespace remove when done
	in :=
		`absLangURL: {{ "index.html" | absLangURL }}
absURL: {{ "http://gohugo.io/" | absURL }}
absURL: {{ "mystyle.css" | absURL }}
absURL: {{ 42 | absURL }}
base64Decode 1: {{ "SGVsbG8gd29ybGQ=" | base64Decode }}
base64Decode 2: {{ 42 | base64Encode | base64Decode }}
base64Encode: {{ "Hello world" | base64Encode }}
crypto.MD5: {{ crypto.MD5 "Hello world, gophers!" }}
dateFormat: {{ dateFormat "Monday, Jan 2, 2006" "2015-01-21" }}
emojify: {{ "I :heart: Hugo" | emojify }}
htmlEscape 1: {{ htmlEscape "Cathal Garvey & The Sunshine Band <cathal@foo.bar>" | safeHTML}}
htmlEscape 2: {{ htmlEscape "Cathal Garvey & The Sunshine Band <cathal@foo.bar>"}}
htmlUnescape 1: {{htmlUnescape "Cathal Garvey &amp; The Sunshine Band &lt;cathal@foo.bar&gt;" | safeHTML}}
htmlUnescape 2: {{"Cathal Garvey &amp;amp; The Sunshine Band &amp;lt;cathal@foo.bar&amp;gt;" | htmlUnescape | htmlUnescape | safeHTML}}
htmlUnescape 3: {{"Cathal Garvey &amp;amp; The Sunshine Band &amp;lt;cathal@foo.bar&amp;gt;" | htmlUnescape | htmlUnescape }}
htmlUnescape 4: {{ htmlEscape "Cathal Garvey & The Sunshine Band <cathal@foo.bar>" | htmlUnescape | safeHTML }}
htmlUnescape 5: {{ htmlUnescape "Cathal Garvey &amp; The Sunshine Band &lt;cathal@foo.bar&gt;" | htmlEscape | safeHTML }}
humanize 1: {{ humanize "my-first-post" }}
humanize 2: {{ humanize "myCamelPost" }}
humanize 3: {{ humanize "52" }}
humanize 4: {{ humanize 103 }}
jsonify: {{ (slice "A" "B" "C") | jsonify }}
markdownify: {{ .Title | markdownify}}
md5: {{ md5 "Hello world, gophers!" }}
print: {{ print "works!" }}
printf: {{ printf "%s!" "works" }}
println: {{ println "works!" -}}
plainify: {{ plainify  "Hello <strong>world</strong>, gophers!" }}
pluralize: {{ "cat" | pluralize }}
readDir: {{ range (readDir ".") }}{{ .Name }}{{ end }}
readFile: {{ readFile "README.txt" }}
relLangURL: {{ "index.html" | relLangURL }}
relURL 1: {{ "http://gohugo.io/" | relURL }}
relURL 2: {{ "mystyle.css" | relURL }}
relURL 3: {{ mul 2 21 | relURL }}
safeCSS: {{ "Bat&Man" | safeCSS | safeCSS }}
safeHTML: {{ "Bat&Man" | safeHTML | safeHTML }}
safeHTML: {{ "Bat&Man" | safeHTML }}
safeJS: {{ "(1*2)" | safeJS | safeJS }}
safeURL: {{ "http://gohugo.io" | safeURL | safeURL }}
sha1: {{ sha1 "Hello world, gophers!" }}
sha256: {{ sha256 "Hello world, gophers!" }}
singularize: {{ "cats" | singularize }}
strings.TrimPrefix: {{ strings.TrimPrefix "Goodbye,, world!" "Goodbye," }}
time: {{ (time "2015-01-21").Year }}
urlize: {{ "Bat Man" | urlize }}
`

	expected := `absLangURL: http://mysite.com/hugo/en/index.html
absURL: http://gohugo.io/
absURL: http://mysite.com/hugo/mystyle.css
absURL: http://mysite.com/hugo/42
base64Decode 1: Hello world
base64Decode 2: 42
base64Encode: SGVsbG8gd29ybGQ=
crypto.MD5: b3029f756f98f79e7f1b7f1d1f0dd53b
dateFormat: Wednesday, Jan 21, 2015
emojify: I ❤️ Hugo
htmlEscape 1: Cathal Garvey &amp; The Sunshine Band &lt;cathal@foo.bar&gt;
htmlEscape 2: Cathal Garvey &amp;amp; The Sunshine Band &amp;lt;cathal@foo.bar&amp;gt;
htmlUnescape 1: Cathal Garvey & The Sunshine Band <cathal@foo.bar>
htmlUnescape 2: Cathal Garvey & The Sunshine Band <cathal@foo.bar>
htmlUnescape 3: Cathal Garvey &amp; The Sunshine Band &lt;cathal@foo.bar&gt;
htmlUnescape 4: Cathal Garvey & The Sunshine Band <cathal@foo.bar>
htmlUnescape 5: Cathal Garvey &amp; The Sunshine Band &lt;cathal@foo.bar&gt;
humanize 1: My first post
humanize 2: My camel post
humanize 3: 52nd
humanize 4: 103rd
jsonify: ["A","B","C"]
markdownify: <strong>BatMan</strong>
md5: b3029f756f98f79e7f1b7f1d1f0dd53b
print: works!
printf: works!
println: works!
plainify: Hello world, gophers!
pluralize: cats
readDir: README.txt
readFile: Hugo Rocks!
relLangURL: /hugo/en/index.html
relURL 1: http://gohugo.io/
relURL 2: /hugo/mystyle.css
relURL 3: /hugo/42
safeCSS: Bat&amp;Man
safeHTML: Bat&Man
safeHTML: Bat&Man
safeJS: (1*2)
safeURL: http://gohugo.io
sha1: c8b5b0e33d408246e30f53e32b8f7627a7a649d4
sha256: 6ec43b78da9669f50e4e422575c54bf87536954ccd58280219c393f2ce352b46
singularize: cat
strings.TrimPrefix: , world!
time: 2015
urlize: bat-man
`

	var b bytes.Buffer

	var data struct {
		Title   string
		Section string
		Params  map[string]interface{}
	}

	data.Title = "**BatMan**"
	data.Section = "blog"
	data.Params = map[string]interface{}{"langCode": "en"}

	v.Set("baseURL", "http://mysite.com/hugo/")
	v.Set("CurrentContentLanguage", helpers.NewLanguage("en", v))

	config := newDepsConfig(v)
	config.WithTemplate = func(templ tpl.TemplateHandler) error {
		if err := templ.AddTemplate("test", in); err != nil {
			t.Fatal("Got error on parse", err)
		}
		return nil
	}
	config.Fs = fs

	d, err := deps.New(config)
	if err != nil {
		t.Fatal(err)
	}

	if err := d.LoadResources(); err != nil {
		t.Fatal(err)
	}

	err = d.Tmpl.Lookup("test").Execute(&b, &data)

	if err != nil {
		t.Fatal("Got error on execute", err)
	}

	if b.String() != expected {
		sl1 := strings.Split(b.String(), "\n")
		sl2 := strings.Split(expected, "\n")
		t.Errorf("Diff:\n%q", helpers.DiffStringSlices(sl1, sl2))
	}
}

func TestDefault(t *testing.T) {
	t.Parallel()
	for i, this := range []struct {
		input    interface{}
		tpl      string
		expected string
		ok       bool
	}{
		{map[string]string{"foo": "bar"}, `{{ index . "foo" | default "nope" }}`, `bar`, true},
		{map[string]string{"foo": "pop"}, `{{ index . "bar" | default "nada" }}`, `nada`, true},
		{map[string]string{"foo": "cat"}, `{{ default "nope" .foo }}`, `cat`, true},
		{map[string]string{"foo": "dog"}, `{{ default "nope" .foo "extra" }}`, ``, false},
		{map[string]interface{}{"images": []string{}}, `{{ default "default.jpg" (index .images 0) }}`, `default.jpg`, true},
	} {

		tmpl := newTestTemplate(t, "test", this.tpl)

		buf := new(bytes.Buffer)
		err := tmpl.Execute(buf, this.input)
		if (err == nil) != this.ok {
			t.Errorf("[%d] execute template returned unexpected error: %s", i, err)
			continue
		}

		if buf.String() != this.expected {
			t.Errorf("[%d] execute template got %v, but expected %v", i, buf.String(), this.expected)
		}
	}
}

func TestPartialCached(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		partial string
		tmpl    string
		variant string
	}{
		// name and partial should match between test cases.
		{"test1", "{{ .Title }} seq: {{ shuffle (seq 1 20) }}", `{{ partialCached "test1" . }}`, ""},
		{"test1", "{{ .Title }} seq: {{ shuffle (seq 1 20) }}", `{{ partialCached "test1" . "%s" }}`, "header"},
		{"test1", "{{ .Title }} seq: {{ shuffle (seq 1 20) }}", `{{ partialCached "test1" . "%s" }}`, "footer"},
		{"test1", "{{ .Title }} seq: {{ shuffle (seq 1 20) }}", `{{ partialCached "test1" . "%s" }}`, "header"},
	}

	var data struct {
		Title   string
		Section string
		Params  map[string]interface{}
	}

	data.Title = "**BatMan**"
	data.Section = "blog"
	data.Params = map[string]interface{}{"langCode": "en"}

	for i, tc := range testCases {
		var tmp string
		if tc.variant != "" {
			tmp = fmt.Sprintf(tc.tmpl, tc.variant)
		} else {
			tmp = tc.tmpl
		}

		config := newDepsConfig(viper.New())

		config.WithTemplate = func(templ tpl.TemplateHandler) error {
			err := templ.AddTemplate("testroot", tmp)
			if err != nil {
				return err
			}
			err = templ.AddTemplate("partials/"+tc.name, tc.partial)
			if err != nil {
				return err
			}

			return nil
		}

		de, err := deps.New(config)
		require.NoError(t, err)
		require.NoError(t, de.LoadResources())

		buf := new(bytes.Buffer)
		templ := de.Tmpl.Lookup("testroot")
		err = templ.Execute(buf, &data)
		if err != nil {
			t.Fatalf("[%d] error executing template: %s", i, err)
		}

		for j := 0; j < 10; j++ {
			buf2 := new(bytes.Buffer)
			err := templ.Execute(buf2, nil)
			if err != nil {
				t.Fatalf("[%d] error executing template 2nd time: %s", i, err)
			}

			if !reflect.DeepEqual(buf, buf2) {
				t.Fatalf("[%d] cached results do not match:\nResult 1:\n%q\nResult 2:\n%q", i, buf, buf2)
			}
		}
	}
}

func BenchmarkPartial(b *testing.B) {
	config := newDepsConfig(viper.New())
	config.WithTemplate = func(templ tpl.TemplateHandler) error {
		err := templ.AddTemplate("testroot", `{{ partial "bench1" . }}`)
		if err != nil {
			return err
		}
		err = templ.AddTemplate("partials/bench1", `{{ shuffle (seq 1 10) }}`)
		if err != nil {
			return err
		}

		return nil
	}

	de, err := deps.New(config)
	require.NoError(b, err)
	require.NoError(b, de.LoadResources())

	buf := new(bytes.Buffer)
	tmpl := de.Tmpl.Lookup("testroot")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := tmpl.Execute(buf, nil); err != nil {
			b.Fatalf("error executing template: %s", err)
		}
		buf.Reset()
	}
}

func BenchmarkPartialCached(b *testing.B) {
	config := newDepsConfig(viper.New())
	config.WithTemplate = func(templ tpl.TemplateHandler) error {
		err := templ.AddTemplate("testroot", `{{ partialCached "bench1" . }}`)
		if err != nil {
			return err
		}
		err = templ.AddTemplate("partials/bench1", `{{ shuffle (seq 1 10) }}`)
		if err != nil {
			return err
		}

		return nil
	}

	de, err := deps.New(config)
	require.NoError(b, err)
	require.NoError(b, de.LoadResources())

	buf := new(bytes.Buffer)
	tmpl := de.Tmpl.Lookup("testroot")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := tmpl.Execute(buf, nil); err != nil {
			b.Fatalf("error executing template: %s", err)
		}
		buf.Reset()
	}
}

func newTestFuncster() *templateFuncster {
	return newTestFuncsterWithViper(viper.New())
}

func newTestFuncsterWithViper(v *viper.Viper) *templateFuncster {
	config := newDepsConfig(v)
	d, err := deps.New(config)
	if err != nil {
		panic(err)
	}

	if err := d.LoadResources(); err != nil {
		panic(err)
	}

	return d.Tmpl.(*templateHandler).html.funcster
}

func newTestTemplate(t *testing.T, name, template string) tpl.Template {
	config := newDepsConfig(viper.New())
	config.WithTemplate = func(templ tpl.TemplateHandler) error {
		err := templ.AddTemplate(name, template)
		if err != nil {
			return err
		}
		return nil
	}

	de, err := deps.New(config)
	require.NoError(t, err)
	require.NoError(t, de.LoadResources())

	return de.Tmpl.Lookup(name)
}
