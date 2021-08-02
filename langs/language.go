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

package langs

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	translators "github.com/gohugoio/localescompressed"
	"github.com/gohugoio/locales"
	"github.com/gohugoio/hugo/common/maps"
	"github.com/gohugoio/hugo/config"
)

// These are the settings that should only be looked up in the global Viper
// config and not per language.
// This list may not be complete, but contains only settings that we know
// will be looked up in both.
// This isn't perfect, but it is ultimately the user who shoots him/herself in
// the foot.
// See the pathSpec.
var globalOnlySettings = map[string]bool{
	strings.ToLower("defaultContentLanguageInSubdir"): true,
	strings.ToLower("defaultContentLanguage"):         true,
	strings.ToLower("multilingual"):                   true,
	strings.ToLower("assetDir"):                       true,
	strings.ToLower("resourceDir"):                    true,
	strings.ToLower("build"):                          true,
}

// Language manages specific-language configuration.
type Language struct {
	Lang              string
	LanguageName      string
	LanguageDirection string
	Title             string
	Weight            int

	Disabled bool

	// If set per language, this tells Hugo that all content files without any
	// language indicator (e.g. my-page.en.md) is in this language.
	// This is usually a path relative to the working dir, but it can be an
	// absolute directory reference. It is what we get.
	ContentDir string

	// Global config.
	Cfg config.Provider

	// Language specific config.
	LocalCfg config.Provider

	// Composite config.
	config.Provider

	// These are params declared in the [params] section of the language merged with the
	// site's params, the most specific (language) wins on duplicate keys.
	params    map[string]interface{}
	paramsMu  sync.Mutex
	paramsSet bool

	// Used for date formatting etc. We don't want these exported to the
	// templates.
	// TODO(bep) do the same for some of the others.
	translator locales.Translator

	location *time.Location

	// Error during initialization. Will fail the buld.
	initErr error
}

func (l *Language) String() string {
	return l.Lang
}

// NewLanguage creates a new language.
func NewLanguage(lang string, cfg config.Provider) *Language {
	// Note that language specific params will be overridden later.
	// We should improve that, but we need to make a copy:
	params := make(map[string]interface{})
	for k, v := range cfg.GetStringMap("params") {
		params[k] = v
	}
	maps.PrepareParams(params)

	localCfg := config.New()
	compositeConfig := config.NewCompositeConfig(cfg, localCfg)
	translator := translators.GetTranslator(lang)
	if translator == nil {
		translator = translators.GetTranslator(cfg.GetString("defaultContentLanguage"))
		if translator == nil {
			translator = translators.GetTranslator("en")
		}
	}

	l := &Language{
		Lang:       lang,
		ContentDir: cfg.GetString("contentDir"),
		Cfg:        cfg, LocalCfg: localCfg,
		Provider:   compositeConfig,
		params:     params,
		translator: translator,
	}

	if err := l.loadLocation(cfg.GetString("timeZone")); err != nil {
		l.initErr = err
	}

	return l
}

// NewDefaultLanguage creates the default language for a config.Provider.
// If not otherwise specified the default is "en".
func NewDefaultLanguage(cfg config.Provider) *Language {
	defaultLang := cfg.GetString("defaultContentLanguage")

	if defaultLang == "" {
		defaultLang = "en"
	}

	return NewLanguage(defaultLang, cfg)
}

// Languages is a sortable list of languages.
type Languages []*Language

// NewLanguages creates a sorted list of languages.
// NOTE: function is currently unused.
func NewLanguages(l ...*Language) Languages {
	languages := make(Languages, len(l))
	for i := 0; i < len(l); i++ {
		languages[i] = l[i]
	}
	sort.Sort(languages)
	return languages
}

func (l Languages) Len() int { return len(l) }
func (l Languages) Less(i, j int) bool {
	wi, wj := l[i].Weight, l[j].Weight

	if wi == wj {
		return l[i].Lang < l[j].Lang
	}

	return wj == 0 || wi < wj
}

func (l Languages) Swap(i, j int) { l[i], l[j] = l[j], l[i] }

// Params returns language-specific params merged with the global params.
func (l *Language) Params() maps.Params {
	// TODO(bep) this construct should not be needed. Create the
	// language params in one go.
	l.paramsMu.Lock()
	defer l.paramsMu.Unlock()
	if !l.paramsSet {
		maps.PrepareParams(l.params)
		l.paramsSet = true
	}
	return l.params
}

func (l Languages) AsSet() map[string]bool {
	m := make(map[string]bool)
	for _, lang := range l {
		m[lang.Lang] = true
	}

	return m
}

func (l Languages) AsOrdinalSet() map[string]int {
	m := make(map[string]int)
	for i, lang := range l {
		m[lang.Lang] = i
	}

	return m
}

// IsMultihost returns whether there are more than one language and at least one of
// the languages has baseURL specificed on the language level.
func (l Languages) IsMultihost() bool {
	if len(l) <= 1 {
		return false
	}

	for _, lang := range l {
		if lang.GetLocal("baseURL") != nil {
			return true
		}
	}
	return false
}

// SetParam sets a param with the given key and value.
// SetParam is case-insensitive.
func (l *Language) SetParam(k string, v interface{}) {
	l.paramsMu.Lock()
	defer l.paramsMu.Unlock()
	if l.paramsSet {
		panic("params cannot be changed once set")
	}
	l.params[k] = v
}

// GetLocal gets a configuration value set on language level. It will
// not fall back to any global value.
// It will return nil if a value with the given key cannot be found.
func (l *Language) GetLocal(key string) interface{} {
	if l == nil {
		panic("language not set")
	}
	key = strings.ToLower(key)
	if !globalOnlySettings[key] {
		return l.LocalCfg.Get(key)
	}
	return nil
}

func (l *Language) Set(k string, v interface{}) {
	k = strings.ToLower(k)
	if globalOnlySettings[k] {
		return
	}
	l.Provider.Set(k, v)
}

// Merge is currently not supported for Language.
func (l *Language) Merge(key string, value interface{}) {
	panic("Not supported")
}

// IsSet checks whether the key is set in the language or the related config store.
func (l *Language) IsSet(key string) bool {
	key = strings.ToLower(key)
	if !globalOnlySettings[key] {
		return l.Provider.IsSet(key)
	}
	return l.Cfg.IsSet(key)
}

// Internal access to unexported Language fields.
// This construct is to prevent them from leaking to the templates.

func GetTranslator(l *Language) locales.Translator {
	return l.translator
}

func GetLocation(l *Language) *time.Location {
	return l.location
}

func (l *Language) loadLocation(tzStr string) error {
	location, err := time.LoadLocation(tzStr)
	if err != nil {
		return errors.Wrapf(err, "invalid timeZone for language %q", l.Lang)
	}
	l.location = location

	return nil
}
