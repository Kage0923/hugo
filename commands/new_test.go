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

package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/hugo/hugofs"
	"github.com/stretchr/testify/assert"
)

// Issue #1133
func TestNewContentPathSectionWithForwardSlashes(t *testing.T) {
	p, s := newContentPathSection("/post/new.md")
	assert.Equal(t, filepath.FromSlash("/post/new.md"), p)
	assert.Equal(t, "post", s)
}

func checkNewSiteInited(basepath string, t *testing.T) {
	paths := []string{
		filepath.Join(basepath, "layouts"),
		filepath.Join(basepath, "content"),
		filepath.Join(basepath, "archetypes"),
		filepath.Join(basepath, "static"),
		filepath.Join(basepath, "data"),
		filepath.Join(basepath, "config.toml"),
	}

	for _, path := range paths {
		_, err := hugofs.Source().Stat(path)
		assert.Nil(t, err)
	}
}

func TestDoNewSite(t *testing.T) {
	basepath := filepath.Join(os.TempDir(), "blog")
	hugofs.InitMemFs()
	err := doNewSite(basepath, false)
	assert.Nil(t, err)

	checkNewSiteInited(basepath, t)
}

func TestDoNewSite_noerror_base_exists_but_empty(t *testing.T) {
	basepath := filepath.Join(os.TempDir(), "blog")
	hugofs.InitMemFs()
	hugofs.Source().MkdirAll(basepath, 777)
	err := doNewSite(basepath, false)
	assert.Nil(t, err)
}

func TestDoNewSite_error_base_exists(t *testing.T) {
	basepath := filepath.Join(os.TempDir(), "blog")
	hugofs.InitMemFs()
	hugofs.Source().MkdirAll(basepath, 777)
	hugofs.Source().Create(filepath.Join(basepath, "foo"))
	// Since the directory already exists and isn't empty, expect an error
	err := doNewSite(basepath, false)
	assert.NotNil(t, err)
}

func TestDoNewSite_force_empty_dir(t *testing.T) {
	basepath := filepath.Join(os.TempDir(), "blog")
	hugofs.InitMemFs()
	hugofs.Source().MkdirAll(basepath, 777)
	err := doNewSite(basepath, true)
	assert.Nil(t, err)

	checkNewSiteInited(basepath, t)
}

func TestDoNewSite_error_force_dir_inside_exists(t *testing.T) {
	basepath := filepath.Join(os.TempDir(), "blog")
	contentPath := filepath.Join(basepath, "content")
	hugofs.InitMemFs()
	hugofs.Source().MkdirAll(contentPath, 777)
	err := doNewSite(basepath, true)
	assert.NotNil(t, err)
}

func TestDoNewSite_error_force_config_inside_exists(t *testing.T) {
	basepath := filepath.Join(os.TempDir(), "blog")
	configPath := filepath.Join(basepath, "config.toml")
	hugofs.InitMemFs()
	hugofs.Source().MkdirAll(basepath, 777)
	hugofs.Source().Create(configPath)
	err := doNewSite(basepath, true)
	assert.NotNil(t, err)
}
