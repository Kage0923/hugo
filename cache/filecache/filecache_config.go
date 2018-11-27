// Copyright 2018 The Hugo Authors. All rights reserved.
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

package filecache

import (
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gohugoio/hugo/helpers"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

const (
	cachesConfigKey = "caches"

	resourcesGenDir = ":resourceDir/_gen"
)

var defaultCacheConfig = cacheConfig{
	MaxAge: -1, // Never expire
	Dir:    ":cacheDir/:project",
}

const (
	cacheKeyGetJSON = "getjson"
	cacheKeyGetCSV  = "getcsv"
	cacheKeyImages  = "images"
	cacheKeyAssets  = "assets"
)

var defaultCacheConfigs = map[string]cacheConfig{
	cacheKeyGetJSON: defaultCacheConfig,
	cacheKeyGetCSV:  defaultCacheConfig,
	cacheKeyImages: cacheConfig{
		MaxAge: -1,
		Dir:    resourcesGenDir,
	},
	cacheKeyAssets: cacheConfig{
		MaxAge: -1,
		Dir:    resourcesGenDir,
	},
}

type cachesConfig map[string]cacheConfig

type cacheConfig struct {
	// Max age of cache entries in this cache. Any items older than this will
	// be removed and not returned from the cache.
	// a negative value means forever, 0 means cache is disabled.
	MaxAge time.Duration

	// The directory where files are stored.
	Dir string

	// Will resources/_gen will get its own composite filesystem that
	// also checks any theme.
	isResourceDir bool
}

// GetJSONCache gets the file cache for getJSON.
func (f Caches) GetJSONCache() *Cache {
	return f[cacheKeyGetJSON]
}

// GetCSVCache gets the file cache for getCSV.
func (f Caches) GetCSVCache() *Cache {
	return f[cacheKeyGetCSV]
}

// ImageCache gets the file cache for processed images.
func (f Caches) ImageCache() *Cache {
	return f[cacheKeyImages]
}

// AssetsCache gets the file cache for assets (processed resources, SCSS etc.).
func (f Caches) AssetsCache() *Cache {
	return f[cacheKeyAssets]
}

func decodeConfig(p *helpers.PathSpec) (cachesConfig, error) {
	c := make(cachesConfig)
	valid := make(map[string]bool)
	// Add defaults
	for k, v := range defaultCacheConfigs {
		c[k] = v
		valid[k] = true
	}

	cfg := p.Cfg

	m := cfg.GetStringMap(cachesConfigKey)

	_, isOsFs := p.Fs.Source.(*afero.OsFs)

	for k, v := range m {
		cc := defaultCacheConfig

		dc := &mapstructure.DecoderConfig{
			Result:           &cc,
			DecodeHook:       mapstructure.StringToTimeDurationHookFunc(),
			WeaklyTypedInput: true,
		}

		decoder, err := mapstructure.NewDecoder(dc)
		if err != nil {
			return c, err
		}

		if err := decoder.Decode(v); err != nil {
			return nil, err
		}

		if cc.Dir == "" {
			return c, errors.New("must provide cache Dir")
		}

		name := strings.ToLower(k)
		if !valid[name] {
			return nil, errors.Errorf("%q is not a valid cache name", name)
		}

		c[name] = cc
	}

	// This is a very old flag in Hugo, but we need to respect it.
	disabled := cfg.GetBool("ignoreCache")

	for k, v := range c {
		dir := filepath.ToSlash(filepath.Clean(v.Dir))
		hadSlash := strings.HasPrefix(dir, "/")
		parts := strings.Split(dir, "/")

		for i, part := range parts {
			if strings.HasPrefix(part, ":") {
				resolved, isResource, err := resolveDirPlaceholder(p, part)
				if err != nil {
					return c, err
				}
				if isResource {
					v.isResourceDir = true
				}
				parts[i] = resolved
			}
		}

		dir = path.Join(parts...)
		if hadSlash {
			dir = "/" + dir
		}
		v.Dir = filepath.Clean(filepath.FromSlash(dir))

		if !v.isResourceDir {
			if isOsFs && !filepath.IsAbs(v.Dir) {
				return c, errors.Errorf("%q must resolve to an absolute directory", v.Dir)
			}

			// Avoid cache in root, e.g. / (Unix) or c:\ (Windows)
			if len(strings.TrimPrefix(v.Dir, filepath.VolumeName(v.Dir))) == 1 {
				return c, errors.Errorf("%q is a root folder and not allowed as cache dir", v.Dir)
			}
		}

		if disabled {
			v.MaxAge = 0
		}

		c[k] = v
	}

	return c, nil
}

// Resolves :resourceDir => /myproject/resources etc., :cacheDir => ...
func resolveDirPlaceholder(p *helpers.PathSpec, placeholder string) (cacheDir string, isResource bool, err error) {
	switch strings.ToLower(placeholder) {
	case ":resourcedir":
		return "", true, nil
	case ":cachedir":
		d, err := helpers.GetCacheDir(p.Fs.Source, p.Cfg)
		return d, false, err
	case ":project":
		return filepath.Base(p.WorkingDir), false, nil
	}

	return "", false, errors.Errorf("%q is not a valid placeholder (valid values are :cacheDir or :resourceDir)", placeholder)
}
