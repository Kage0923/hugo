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

package glob

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/gobwas/glob"
	"github.com/gobwas/glob/syntax"
)

const filepathSeparator = string(os.PathSeparator)

var (
	isWindows        = runtime.GOOS == "windows"
	defaultGlobCache = &globCache{
		isCaseSensitive: false,
		isWindows:       isWindows,
		cache:           make(map[string]globErr),
	}

	filenamesGlobCache = &globCache{
		isCaseSensitive: false, // As long as the search strings are all lower case, this does not allocate.
		isWindows:       isWindows,
		cache:           make(map[string]globErr),
	}
)

type globErr struct {
	glob glob.Glob
	err  error
}

type globCache struct {
	// Config
	isCaseSensitive bool
	isWindows       bool

	// Cache
	sync.RWMutex
	cache map[string]globErr
}

func (gc *globCache) GetGlob(pattern string) (glob.Glob, error) {
	var eg globErr

	gc.RLock()
	var found bool
	eg, found = gc.cache[pattern]
	gc.RUnlock()
	if found {
		return eg.glob, eg.err
	}

	var g glob.Glob
	var err error

	pattern = filepath.ToSlash(pattern)

	if gc.isCaseSensitive {
		g, err = glob.Compile(pattern, '/')
	} else {
		g, err = glob.Compile(strings.ToLower(pattern), '/')

	}

	eg = globErr{
		globDecorator{
			g:               g,
			isCaseSensitive: gc.isCaseSensitive,
			isWindows:       gc.isWindows},
		err,
	}

	gc.Lock()
	gc.cache[pattern] = eg
	gc.Unlock()

	return eg.glob, eg.err
}

type globDecorator struct {
	// Whether both pattern and the strings to match will be matched
	// by their original case.
	isCaseSensitive bool

	// On Windows we may get filenames with Windows slashes to match,
	// which wee need to normalize.
	isWindows bool

	g glob.Glob
}

func (g globDecorator) Match(s string) bool {
	if g.isWindows {
		s = filepath.ToSlash(s)
	}
	if !g.isCaseSensitive {
		s = strings.ToLower(s)
	}
	return g.g.Match(s)
}

func GetGlob(pattern string) (glob.Glob, error) {
	return defaultGlobCache.GetGlob(pattern)
}

func NormalizePath(p string) string {
	return strings.Trim(path.Clean(filepath.ToSlash(strings.ToLower(p))), "/.")
}

// ResolveRootDir takes a normalized path on the form "assets/**.json" and
// determines any root dir, i.e. any start path without any wildcards.
func ResolveRootDir(p string) string {
	parts := strings.Split(path.Dir(p), "/")
	var roots []string
	for _, part := range parts {
		if HasGlobChar(part) {
			break
		}
		roots = append(roots, part)
	}

	if len(roots) == 0 {
		return ""
	}

	return strings.Join(roots, "/")
}

// FilterGlobParts removes any string with glob wildcard.
func FilterGlobParts(a []string) []string {
	b := a[:0]
	for _, x := range a {
		if !HasGlobChar(x) {
			b = append(b, x)
		}
	}
	return b
}

// HasGlobChar returns whether s contains any glob wildcards.
func HasGlobChar(s string) bool {
	for i := 0; i < len(s); i++ {
		if syntax.Special(s[i]) {
			return true
		}
	}
	return false
}
