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

package helpers

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/gohugoio/hugo/common/text"

	"github.com/gohugoio/hugo/config"

	"github.com/gohugoio/hugo/hugofs"

	"github.com/gohugoio/hugo/common/hugio"
	_errors "github.com/pkg/errors"
	"github.com/spf13/afero"
)

// MakePath takes a string with any characters and replace it
// so the string could be used in a path.
// It does so by creating a Unicode-sanitized string, with the spaces replaced,
// whilst preserving the original casing of the string.
// E.g. Social Media -> Social-Media
func (p *PathSpec) MakePath(s string) string {
	return p.UnicodeSanitize(s)
}

// MakePathsSanitized applies MakePathSanitized on every item in the slice
func (p *PathSpec) MakePathsSanitized(paths []string) {
	for i, path := range paths {
		paths[i] = p.MakePathSanitized(path)
	}
}

// MakePathSanitized creates a Unicode-sanitized string, with the spaces replaced
func (p *PathSpec) MakePathSanitized(s string) string {
	if p.DisablePathToLower {
		return p.MakePath(s)
	}
	return strings.ToLower(p.MakePath(s))
}

// ToSlashTrimLeading is just a filepath.ToSlaas with an added / prefix trimmer.
func ToSlashTrimLeading(s string) string {
	return strings.TrimPrefix(filepath.ToSlash(s), "/")
}

// MakeTitle converts the path given to a suitable title, trimming whitespace
// and replacing hyphens with whitespace.
func MakeTitle(inpath string) string {
	return strings.Replace(strings.TrimSpace(inpath), "-", " ", -1)
}

// From https://golang.org/src/net/url/url.go
func ishex(c rune) bool {
	switch {
	case '0' <= c && c <= '9':
		return true
	case 'a' <= c && c <= 'f':
		return true
	case 'A' <= c && c <= 'F':
		return true
	}
	return false
}

// UnicodeSanitize sanitizes string to be used in Hugo URL's, allowing only
// a predefined set of special Unicode characters.
// If RemovePathAccents configuration flag is enabled, Unicode accents
// are also removed.
// Hyphens in the original input are maintained.
// Spaces will be replaced with a single hyphen, and sequential replacement hyphens will be reduced to one.
func (p *PathSpec) UnicodeSanitize(s string) string {
	if p.RemovePathAccents {
		s = text.RemoveAccentsString(s)
	}

	source := []rune(s)
	target := make([]rune, 0, len(source))
	var (
		prependHyphen bool
		wasHyphen     bool
	)

	for i, r := range source {
		isAllowed := r == '.' || r == '/' || r == '\\' || r == '_' || r == '#' || r == '+' || r == '~' || r == '-'
		isAllowed = isAllowed || unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r)
		isAllowed = isAllowed || (r == '%' && i+2 < len(source) && ishex(source[i+1]) && ishex(source[i+2]))

		if isAllowed {
			// track explicit hyphen in input; no need to add a new hyphen if
			// we just saw one.
			wasHyphen = r == '-'

			if prependHyphen {
				// if currently have a hyphen, don't prepend an extra one
				if !wasHyphen {
					target = append(target, '-')
				}
				prependHyphen = false
			}
			target = append(target, r)
		} else if len(target) > 0 && !wasHyphen && unicode.IsSpace(r) {
			prependHyphen = true
		}
	}

	return string(target)
}

func makePathRelative(inPath string, possibleDirectories ...string) (string, error) {
	for _, currentPath := range possibleDirectories {
		if strings.HasPrefix(inPath, currentPath) {
			return strings.TrimPrefix(inPath, currentPath), nil
		}
	}
	return inPath, errors.New("can't extract relative path, unknown prefix")
}

// Should be good enough for Hugo.
var isFileRe = regexp.MustCompile(`.*\..{1,6}$`)

// GetDottedRelativePath expects a relative path starting after the content directory.
// It returns a relative path with dots ("..") navigating up the path structure.
func GetDottedRelativePath(inPath string) string {
	inPath = filepath.Clean(filepath.FromSlash(inPath))

	if inPath == "." {
		return "./"
	}

	if !isFileRe.MatchString(inPath) && !strings.HasSuffix(inPath, FilePathSeparator) {
		inPath += FilePathSeparator
	}

	if !strings.HasPrefix(inPath, FilePathSeparator) {
		inPath = FilePathSeparator + inPath
	}

	dir, _ := filepath.Split(inPath)

	sectionCount := strings.Count(dir, FilePathSeparator)

	if sectionCount == 0 || dir == FilePathSeparator {
		return "./"
	}

	var dottedPath string

	for i := 1; i < sectionCount; i++ {
		dottedPath += "../"
	}

	return dottedPath
}

type NamedSlice struct {
	Name  string
	Slice []string
}

func (n NamedSlice) String() string {
	if len(n.Slice) == 0 {
		return n.Name
	}
	return fmt.Sprintf("%s%s{%s}", n.Name, FilePathSeparator, strings.Join(n.Slice, ","))
}

func ExtractAndGroupRootPaths(paths []string) []NamedSlice {
	if len(paths) == 0 {
		return nil
	}

	pathsCopy := make([]string, len(paths))
	hadSlashPrefix := strings.HasPrefix(paths[0], FilePathSeparator)

	for i, p := range paths {
		pathsCopy[i] = strings.Trim(filepath.ToSlash(p), "/")
	}

	sort.Strings(pathsCopy)

	pathsParts := make([][]string, len(pathsCopy))

	for i, p := range pathsCopy {
		pathsParts[i] = strings.Split(p, "/")
	}

	var groups [][]string

	for i, p1 := range pathsParts {
		c1 := -1

		for j, p2 := range pathsParts {
			if i == j {
				continue
			}

			c2 := -1

			for i, v := range p1 {
				if i >= len(p2) {
					break
				}
				if v != p2[i] {
					break
				}

				c2 = i
			}

			if c1 == -1 || (c2 != -1 && c2 < c1) {
				c1 = c2
			}
		}

		if c1 != -1 {
			groups = append(groups, p1[:c1+1])
		} else {
			groups = append(groups, p1)
		}
	}

	groupsStr := make([]string, len(groups))
	for i, g := range groups {
		groupsStr[i] = strings.Join(g, "/")
	}

	groupsStr = UniqueStringsSorted(groupsStr)

	var result []NamedSlice

	for _, g := range groupsStr {
		name := filepath.FromSlash(g)
		if hadSlashPrefix {
			name = FilePathSeparator + name
		}
		ns := NamedSlice{Name: name}
		for _, p := range pathsCopy {
			if !strings.HasPrefix(p, g) {
				continue
			}

			p = strings.TrimPrefix(p, g)
			if p != "" {
				ns.Slice = append(ns.Slice, p)
			}
		}

		ns.Slice = UniqueStrings(ExtractRootPaths(ns.Slice))

		result = append(result, ns)
	}

	return result
}

// ExtractRootPaths extracts the root paths from the supplied list of paths.
// The resulting root path will not contain any file separators, but there
// may be duplicates.
// So "/content/section/" becomes "content"
func ExtractRootPaths(paths []string) []string {
	r := make([]string, len(paths))
	for i, p := range paths {
		root := filepath.ToSlash(p)
		sections := strings.Split(root, "/")
		for _, section := range sections {
			if section != "" {
				root = section
				break
			}
		}
		r[i] = root
	}
	return r
}

// FindCWD returns the current working directory from where the Hugo
// executable is run.
func FindCWD() (string, error) {
	serverFile, err := filepath.Abs(os.Args[0])
	if err != nil {
		return "", fmt.Errorf("can't get absolute path for executable: %v", err)
	}

	path := filepath.Dir(serverFile)
	realFile, err := filepath.EvalSymlinks(serverFile)
	if err != nil {
		if _, err = os.Stat(serverFile + ".exe"); err == nil {
			realFile = filepath.Clean(serverFile + ".exe")
		}
	}

	if err == nil && realFile != serverFile {
		path = filepath.Dir(realFile)
	}

	return path, nil
}

// SymbolicWalk is like filepath.Walk, but it follows symbolic links.
func SymbolicWalk(fs afero.Fs, root string, walker hugofs.WalkFunc) error {
	if _, isOs := fs.(*afero.OsFs); isOs {
		// Mainly to track symlinks.
		fs = hugofs.NewBaseFileDecorator(fs)
	}

	w := hugofs.NewWalkway(hugofs.WalkwayConfig{
		Fs:     fs,
		Root:   root,
		WalkFn: walker,
	})

	return w.Walk()
}

// LstatIfPossible can be used to call Lstat if possible, else Stat.
func LstatIfPossible(fs afero.Fs, path string) (os.FileInfo, error) {
	if lstater, ok := fs.(afero.Lstater); ok {
		fi, _, err := lstater.LstatIfPossible(path)
		return fi, err
	}

	return fs.Stat(path)
}

// SafeWriteToDisk is the same as WriteToDisk
// but it also checks to see if file/directory already exists.
func SafeWriteToDisk(inpath string, r io.Reader, fs afero.Fs) (err error) {
	return afero.SafeWriteReader(fs, inpath, r)
}

// WriteToDisk writes content to disk.
func WriteToDisk(inpath string, r io.Reader, fs afero.Fs) (err error) {
	return afero.WriteReader(fs, inpath, r)
}

// OpenFilesForWriting opens all the given filenames for writing.
func OpenFilesForWriting(fs afero.Fs, filenames ...string) (io.WriteCloser, error) {
	var writeClosers []io.WriteCloser
	for _, filename := range filenames {
		f, err := OpenFileForWriting(fs, filename)
		if err != nil {
			for _, wc := range writeClosers {
				wc.Close()
			}
			return nil, err
		}
		writeClosers = append(writeClosers, f)
	}

	return hugio.NewMultiWriteCloser(writeClosers...), nil
}

// OpenFileForWriting opens or creates the given file. If the target directory
// does not exist, it gets created.
func OpenFileForWriting(fs afero.Fs, filename string) (afero.File, error) {
	filename = filepath.Clean(filename)
	// Create will truncate if file already exists.
	// os.Create will create any new files with mode 0666 (before umask).
	f, err := fs.Create(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		if err = fs.MkdirAll(filepath.Dir(filename), 0777); err != nil { //  before umask
			return nil, err
		}
		f, err = fs.Create(filename)
	}

	return f, err
}

// GetCacheDir returns a cache dir from the given filesystem and config.
// The dir will be created if it does not exist.
func GetCacheDir(fs afero.Fs, cfg config.Provider) (string, error) {
	cacheDir := getCacheDir(cfg)
	if cacheDir != "" {
		exists, err := DirExists(cacheDir, fs)
		if err != nil {
			return "", err
		}
		if !exists {
			err := fs.MkdirAll(cacheDir, 0777) // Before umask
			if err != nil {
				return "", _errors.Wrap(err, "failed to create cache dir")
			}
		}
		return cacheDir, nil
	}

	// Fall back to a cache in /tmp.
	return GetTempDir("hugo_cache", fs), nil
}

func getCacheDir(cfg config.Provider) string {
	// Always use the cacheDir config if set.
	cacheDir := cfg.GetString("cacheDir")
	if len(cacheDir) > 1 {
		return addTrailingFileSeparator(cacheDir)
	}

	// See Issue #8714.
	// Turns out that Cloudflare also sets NETLIFY=true in its build environment,
	// but all of these 3 should not give any false positives.
	if os.Getenv("NETLIFY") == "true" && os.Getenv("PULL_REQUEST") != "" && os.Getenv("DEPLOY_PRIME_URL") != "" {
		// Netlify's cache behaviour is not documented, the currently best example
		// is this project:
		// https://github.com/philhawksworth/content-shards/blob/master/gulpfile.js
		return "/opt/build/cache/hugo_cache/"
	}

	// This will fall back to an hugo_cache folder in the tmp dir, which should work fine for most CI
	// providers. See this for a working CircleCI setup:
	// https://github.com/bep/hugo-sass-test/blob/6c3960a8f4b90e8938228688bc49bdcdd6b2d99e/.circleci/config.yml
	// If not, they can set the HUGO_CACHEDIR environment variable or cacheDir config key.
	return ""
}

func addTrailingFileSeparator(s string) string {
	if !strings.HasSuffix(s, FilePathSeparator) {
		s = s + FilePathSeparator
	}
	return s
}

// GetTempDir returns a temporary directory with the given sub path.
func GetTempDir(subPath string, fs afero.Fs) string {
	return afero.GetTempDir(fs, subPath)
}

// DirExists checks if a path exists and is a directory.
func DirExists(path string, fs afero.Fs) (bool, error) {
	return afero.DirExists(fs, path)
}

// IsDir checks if a given path is a directory.
func IsDir(path string, fs afero.Fs) (bool, error) {
	return afero.IsDir(fs, path)
}

// IsEmpty checks if a given path is empty.
func IsEmpty(path string, fs afero.Fs) (bool, error) {
	return afero.IsEmpty(fs, path)
}

// Exists checks if a file or directory exists.
func Exists(path string, fs afero.Fs) (bool, error) {
	return afero.Exists(fs, path)
}

// AddTrailingSlash adds a trailing Unix styled slash (/) if not already
// there.
func AddTrailingSlash(path string) string {
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path
}
