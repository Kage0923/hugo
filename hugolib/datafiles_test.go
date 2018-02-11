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

package hugolib

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"io/ioutil"
	"log"
	"os"

	"github.com/gohugoio/hugo/deps"
	jww "github.com/spf13/jwalterweatherman"

	"fmt"
	"runtime"

	"github.com/stretchr/testify/require"
)

func TestDataDir(t *testing.T) {
	t.Parallel()
	equivDataDirs := make([]dataDir, 3)
	equivDataDirs[0].addSource("data/test/a.json", `{ "b" : { "c1": "red" , "c2": "blue" } }`)
	equivDataDirs[1].addSource("data/test/a.yaml", "b:\n  c1: red\n  c2: blue")
	equivDataDirs[2].addSource("data/test/a.toml", "[b]\nc1 = \"red\"\nc2 = \"blue\"\n")
	expected := map[string]interface{}{
		"test": map[string]interface{}{
			"a": map[string]interface{}{
				"b": map[string]interface{}{
					"c1": "red",
					"c2": "blue",
				},
			},
		},
	}
	doTestEquivalentDataDirs(t, equivDataDirs, expected)
}

// Unable to enforce equivalency for int values as
// the JSON, YAML and TOML parsers return
// float64, int, int64 respectively. They all return
// float64 for float values though:
func TestDataDirNumeric(t *testing.T) {
	t.Parallel()
	equivDataDirs := make([]dataDir, 3)
	equivDataDirs[0].addSource("data/test/a.json", `{ "b" : { "c1": 1.7 , "c2": 2.9 } }`)
	equivDataDirs[1].addSource("data/test/a.yaml", "b:\n  c1: 1.7\n  c2: 2.9")
	equivDataDirs[2].addSource("data/test/a.toml", "[b]\nc1 = 1.7\nc2 = 2.9\n")
	expected := map[string]interface{}{
		"test": map[string]interface{}{
			"a": map[string]interface{}{
				"b": map[string]interface{}{
					"c1": 1.7,
					"c2": 2.9,
				},
			},
		},
	}
	doTestEquivalentDataDirs(t, equivDataDirs, expected)
}

func TestDataDirBoolean(t *testing.T) {
	t.Parallel()
	equivDataDirs := make([]dataDir, 3)
	equivDataDirs[0].addSource("data/test/a.json", `{ "b" : { "c1": true , "c2": false } }`)
	equivDataDirs[1].addSource("data/test/a.yaml", "b:\n  c1: true\n  c2: false")
	equivDataDirs[2].addSource("data/test/a.toml", "[b]\nc1 = true\nc2 = false\n")
	expected := map[string]interface{}{
		"test": map[string]interface{}{
			"a": map[string]interface{}{
				"b": map[string]interface{}{
					"c1": true,
					"c2": false,
				},
			},
		},
	}
	doTestEquivalentDataDirs(t, equivDataDirs, expected)
}

func TestDataDirTwoFiles(t *testing.T) {
	t.Parallel()
	equivDataDirs := make([]dataDir, 2)

	equivDataDirs[0].addSource("data/test/foo.json", `{ "bar": "foofoo"  }`)
	equivDataDirs[0].addSource("data/test.json", `{ "hello": [ { "world": "foo" } ] }`)

	equivDataDirs[1].addSource("data/test/foo.yaml", "bar: foofoo")
	equivDataDirs[1].addSource("data/test.yaml", "hello:\n- world: foo")

	// TODO Unresolved Issue #3890
	/*
		equivDataDirs[2].addSource("data/test/foo.toml", "bar = \"foofoo\"")
		equivDataDirs[2].addSource("data/test.toml", "[[hello]]\nworld = \"foo\"")
	*/

	expected :=
		map[string]interface{}{
			"test": map[string]interface{}{
				"hello": []interface{}{
					map[string]interface{}{"world": "foo"},
				},
				"foo": map[string]interface{}{
					"bar": "foofoo",
				},
			},
		}

	doTestEquivalentDataDirs(t, equivDataDirs, expected)
}

func TestDataDirOverriddenValue(t *testing.T) {
	t.Parallel()
	equivDataDirs := make([]dataDir, 3)

	// filepath.Walk walks the files in lexical order, '/' comes before '.'. Simulate this:
	equivDataDirs[0].addSource("data/a.json", `{"a": "1"}`)
	equivDataDirs[0].addSource("data/test/v1.json", `{"v1-2": "2"}`)
	equivDataDirs[0].addSource("data/test/v2.json", `{"v2": ["2", "3"]}`)
	equivDataDirs[0].addSource("data/test.json", `{"v1": "1"}`)

	equivDataDirs[1].addSource("data/a.yaml", "a: \"1\"")
	equivDataDirs[1].addSource("data/test/v1.yaml", "v1-2: \"2\"")
	equivDataDirs[1].addSource("data/test/v2.yaml", "v2:\n- \"2\"\n- \"3\"")
	equivDataDirs[1].addSource("data/test.yaml", "v1: \"1\"")

	equivDataDirs[2].addSource("data/a.toml", "a = \"1\"")
	equivDataDirs[2].addSource("data/test/v1.toml", "v1-2 = \"2\"")
	equivDataDirs[2].addSource("data/test/v2.toml", "v2 = [\"2\", \"3\"]")
	equivDataDirs[2].addSource("data/test.toml", "v1 = \"1\"")

	expected :=
		map[string]interface{}{
			"a": map[string]interface{}{"a": "1"},
			"test": map[string]interface{}{
				"v1": map[string]interface{}{"v1-2": "2"},
				"v2": map[string]interface{}{"v2": []interface{}{"2", "3"}},
			},
		}

	doTestEquivalentDataDirs(t, equivDataDirs, expected)
}

// Issue #4361
func TestDataDirJSONArrayAtTopLevelOfFile(t *testing.T) {
	t.Parallel()

	var dd dataDir
	dd.addSource("data/test.json", `[ { "hello": "world" }, { "what": "time" }, { "is": "lunch?" } ]`)

	expected :=
		map[string]interface{}{
			"test": []interface{}{
				map[string]interface{}{"hello": "world"},
				map[string]interface{}{"what": "time"},
				map[string]interface{}{"is": "lunch?"},
			},
		}

	doTestDataDir(t, dd, expected)
}

// TODO Issue #3890 unresolved
func TestDataDirYAMLArrayAtTopLevelOfFile(t *testing.T) {
	t.Parallel()

	var dd dataDir
	dd.addSource("data/test.yaml", `
- hello: world
- what: time
- is: lunch?
`)

	//TODO decide whether desired structure map[interface {}]interface{} as shown
	// and as the YAML parser produces, or should it be map[string]interface{}
	// all the way down per Issue #4138
	expected :=
		map[string]interface{}{
			"test": []interface{}{
				map[interface{}]interface{}{"hello": "world"},
				map[interface{}]interface{}{"what": "time"},
				map[interface{}]interface{}{"is": "lunch?"},
			},
		}

	// what we are actually getting as of v0.34
	expectedV0_34 :=
		map[string]interface{}{}
	_ = expected

	doTestDataDir(t, dd, expectedV0_34)
}

// Issue #892
func TestDataDirMultipleSources(t *testing.T) {
	t.Parallel()

	var dd dataDir
	dd.addSource("data/test/first.yaml", "bar: 1")
	dd.addSource("themes/mytheme/data/test/first.yaml", "bar: 2")
	dd.addSource("data/test/second.yaml", "tender: 2")

	expected :=
		map[string]interface{}{
			"test": map[string]interface{}{
				"first": map[string]interface{}{
					"bar": 1,
				},
				"second": map[string]interface{}{
					"tender": 2,
				},
			},
		}

	doTestDataDir(t, dd, expected,
		"theme", "mytheme")

}

// test (and show) the way values from four different sources,
// including theme data, commingle and override
func TestDataDirMultipleSourcesCommingled(t *testing.T) {
	t.Parallel()

	var dd dataDir
	dd.addSource("data/a.json", `{ "b1" : { "c1": "data/a" }, "b2": "data/a", "b3": ["x", "y", "z"] }`)
	dd.addSource("themes/mytheme/data/a.json", `{ "b1": "mytheme/data/a",  "b2": "mytheme/data/a", "b3": "mytheme/data/a" }`)
	dd.addSource("themes/mytheme/data/a/b1.json", `{ "c1": "mytheme/data/a/b1", "c2": "mytheme/data/a/b1" }`)
	dd.addSource("data/a/b1.json", `{ "c1": "data/a/b1" }`)

	// Per handleDataFile() comment:
	// 1. A theme uses the same key; the main data folder wins
	// 2. A sub folder uses the same key: the sub folder wins
	expected :=
		map[string]interface{}{
			"a": map[string]interface{}{
				"b1": map[string]interface{}{
					"c1": "data/a/b1",
					"c2": "mytheme/data/a/b1",
				},
				"b2": "data/a",
				"b3": []interface{}{"x", "y", "z"},
			},
		}

	doTestDataDir(t, dd, expected, "theme", "mytheme")
}

// TODO Issue #4366 unresolved
func _TestDataDirMultipleSourcesCollidingChildArrays(t *testing.T) {
	t.Parallel()

	var dd dataDir
	dd.addSource("themes/mytheme/data/a/b2.json", `["Q", "R", "S"]`)
	dd.addSource("data/a.json", `{ "b1" : "data/a", "b2" : ["x", "y", "z"] }`)
	dd.addSource("data/a/b2.json", `["1", "2", "3"]`)

	// Per handleDataFile() comment:
	// 1. A theme uses the same key; the main data folder wins
	// 2. A sub folder uses the same key: the sub folder wins
	expected :=
		map[string]interface{}{
			"a": map[string]interface{}{
				"b1": "data/a",
				"b2": []interface{}{"1", "2", "3"},
			},
		}

	doTestDataDir(t, dd, expected, "theme", "mytheme")
}

// TODO Issue #4366 unresolved
func _TestDataDirMultipleSourcesCollidingTopLevelArrays(t *testing.T) {
	t.Parallel()

	var dd dataDir
	dd.addSource("themes/mytheme/data/a/b1.json", `["x", "y", "z"]`)
	dd.addSource("data/a/b1.json", `["1", "2", "3"]`)

	expected :=
		map[string]interface{}{
			"a": map[string]interface{}{
				"b1": []interface{}{"1", "2", "3"},
			},
		}

	doTestDataDir(t, dd, expected, "theme", "mytheme")
}

type dataDir struct {
	sources [][2]string
}

func (d *dataDir) addSource(path, content string) {
	d.sources = append(d.sources, [2]string{path, content})
}

func doTestEquivalentDataDirs(t *testing.T, equivDataDirs []dataDir, expected interface{}, configKeyValues ...interface{}) {
	for i, dd := range equivDataDirs {
		err := doTestDataDirImpl(t, dd, expected, configKeyValues...)
		if err != "" {
			t.Errorf("equivDataDirs[%d]: %s", i, err)
		}
	}
}

func doTestDataDir(t *testing.T, dd dataDir, expected interface{}, configKeyValues ...interface{}) {
	err := doTestDataDirImpl(t, dd, expected, configKeyValues...)
	if err != "" {
		t.Error(err)
	}
}

func doTestDataDirImpl(t *testing.T, dd dataDir, expected interface{}, configKeyValues ...interface{}) (err string) {
	var (
		cfg, fs = newTestCfg()
	)

	for i := 0; i < len(configKeyValues); i += 2 {
		cfg.Set(configKeyValues[i].(string), configKeyValues[i+1])
	}

	var (
		logger  = jww.NewNotepad(jww.LevelWarn, jww.LevelWarn, os.Stdout, ioutil.Discard, t.Name(), log.Ldate|log.Ltime)
		depsCfg = deps.DepsCfg{Fs: fs, Cfg: cfg, Logger: logger}
	)

	writeSource(t, fs, filepath.Join("content", "dummy.md"), "content")
	writeSourcesToSource(t, "", fs, dd.sources...)

	expectBuildError := false

	if ok, shouldFail := expected.(bool); ok && shouldFail {
		expectBuildError = true
	}

	// trap and report panics as unmarshaling errors so that test suit can complete
	defer func() {
		if r := recover(); r != nil {
			// Capture the stack trace
			buf := make([]byte, 10000)
			runtime.Stack(buf, false)
			t.Errorf("PANIC: %s\n\nStack Trace : %s", r, string(buf))
		}
	}()

	s := buildSingleSiteExpected(t, expectBuildError, depsCfg, BuildCfg{SkipRender: true})

	if !expectBuildError && !reflect.DeepEqual(expected, s.Data) {
		// This disabled code detects the situation described in the WARNING message below.
		// The situation seems to only occur for TOML data with integer values.
		// Perhaps the TOML parser returns ints in another type.
		// Re-enable temporarily to debug fails that should be passing.
		// Re-enable permanently if reflect.DeepEqual is simply too strict.
		/*
			exp := fmt.Sprintf("%#v", expected)
			got := fmt.Sprintf("%#v", s.Data)
			if exp == got {
				t.Logf("WARNING: reflect.DeepEqual returned FALSE for values that appear equal.\n"+
					"Treating as equal for the purpose of the test, but this maybe should be investigated.\n"+
					"Expected data:\n%v got\n%v\n\nExpected type structure:\n%#[1]v got\n%#[2]v", expected, s.Data)
				return
			}
		*/

		return fmt.Sprintf("Expected data:\n%v got\n%v\n\nExpected type structure:\n%#[1]v got\n%#[2]v", expected, s.Data)
	}

	return
}

func TestDataFromShortcode(t *testing.T) {
	t.Parallel()

	var (
		cfg, fs = newTestCfg()
	)

	writeSource(t, fs, "data/hugo.toml", "slogan = \"Hugo Rocks!\"")
	writeSource(t, fs, "layouts/_default/single.html", `
* Slogan from template: {{  .Site.Data.hugo.slogan }}
* {{ .Content }}`)
	writeSource(t, fs, "layouts/shortcodes/d.html", `{{  .Page.Site.Data.hugo.slogan }}`)
	writeSource(t, fs, "content/c.md", `---
---
Slogan from shortcode: {{< d >}}
`)

	buildSingleSite(t, deps.DepsCfg{Fs: fs, Cfg: cfg}, BuildCfg{})

	content := readSource(t, fs, "public/c/index.html")
	require.True(t, strings.Contains(content, "Slogan from template: Hugo Rocks!"), content)
	require.True(t, strings.Contains(content, "Slogan from shortcode: Hugo Rocks!"), content)

}
