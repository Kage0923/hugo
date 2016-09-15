// Copyright 2016-present The Hugo Authors. All rights reserved.
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

// Package helpers implements general utility functions that work with
// and on content.  The helper functions defined here lay down the
// foundation of how Hugo works with files and filepaths, and perform
// string operations on content.
package helpers

import (
	"github.com/spf13/viper"
)

// ConfigProvider provides the configuration settings for Hugo.
type ConfigProvider interface {
	GetString(key string) string
	GetInt(key string) int
	GetStringMap(key string) map[string]interface{}
	GetStringMapString(key string) map[string]string
}

// Config returns the currently active Hugo config. This will be set
// per site (language) rendered.
func Config() ConfigProvider {
	return viper.Get("CurrentContentLanguage").(ConfigProvider)
}
