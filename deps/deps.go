package deps

import (
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/gohugoio/hugo/cache/filecache"
	"github.com/gohugoio/hugo/common/loggers"
	"github.com/gohugoio/hugo/config"
	"github.com/gohugoio/hugo/helpers"
	"github.com/gohugoio/hugo/hugofs"
	"github.com/gohugoio/hugo/langs"
	"github.com/gohugoio/hugo/media"
	"github.com/gohugoio/hugo/metrics"
	"github.com/gohugoio/hugo/output"
	"github.com/gohugoio/hugo/resource"
	"github.com/gohugoio/hugo/source"
	"github.com/gohugoio/hugo/tpl"
	jww "github.com/spf13/jwalterweatherman"
)

// Deps holds dependencies used by many.
// There will be normally only one instance of deps in play
// at a given time, i.e. one per Site built.
type Deps struct {

	// The logger to use.
	Log *loggers.Logger `json:"-"`

	// Used to log errors that may repeat itself many times.
	DistinctErrorLog *helpers.DistinctLogger

	// The templates to use. This will usually implement the full tpl.TemplateHandler.
	Tmpl tpl.TemplateFinder `json:"-"`

	// We use this to parse and execute ad-hoc text templates.
	TextTmpl tpl.TemplateParseFinder `json:"-"`

	// The file systems to use.
	Fs *hugofs.Fs `json:"-"`

	// The PathSpec to use
	*helpers.PathSpec `json:"-"`

	// The ContentSpec to use
	*helpers.ContentSpec `json:"-"`

	// The SourceSpec to use
	SourceSpec *source.SourceSpec `json:"-"`

	// The Resource Spec to use
	ResourceSpec *resource.Spec

	// The configuration to use
	Cfg config.Provider `json:"-"`

	// The file cache to use.
	FileCaches filecache.Caches

	// The translation func to use
	Translate func(translationID string, args ...interface{}) string `json:"-"`

	Language *langs.Language

	// All the output formats available for the current site.
	OutputFormatsConfig output.Formats

	templateProvider ResourceProvider
	WithTemplate     func(templ tpl.TemplateHandler) error `json:"-"`

	translationProvider ResourceProvider

	Metrics metrics.Provider

	// Timeout is configurable in site config.
	Timeout time.Duration

	// BuildStartListeners will be notified before a build starts.
	BuildStartListeners *Listeners

	*globalErrHandler
}

type globalErrHandler struct {
	// Channel for some "hard to get to" build errors
	buildErrors chan error
}

// SendErr sends the error on a channel to be handled later.
// This can be used in situations where returning and aborting the current
// operation isn't practical.
func (e *globalErrHandler) SendError(err error) {
	if e.buildErrors != nil {
		select {
		case e.buildErrors <- err:
		default:
		}
		return
	}

	jww.ERROR.Println(err)
}

func (e *globalErrHandler) StartErrorCollector() chan error {
	e.buildErrors = make(chan error, 10)
	return e.buildErrors
}

// Listeners represents an event listener.
type Listeners struct {
	sync.Mutex

	// A list of funcs to be notified about an event.
	listeners []func()
}

// Add adds a function to a Listeners instance.
func (b *Listeners) Add(f func()) {
	b.Lock()
	defer b.Unlock()
	b.listeners = append(b.listeners, f)
}

// Notify executes all listener functions.
func (b *Listeners) Notify() {
	b.Lock()
	defer b.Unlock()
	for _, notify := range b.listeners {
		notify()
	}
}

// ResourceProvider is used to create and refresh, and clone resources needed.
type ResourceProvider interface {
	Update(deps *Deps) error
	Clone(deps *Deps) error
}

// TemplateHandler returns the used tpl.TemplateFinder as tpl.TemplateHandler.
func (d *Deps) TemplateHandler() tpl.TemplateHandler {
	return d.Tmpl.(tpl.TemplateHandler)
}

// LoadResources loads translations and templates.
func (d *Deps) LoadResources() error {
	// Note that the translations need to be loaded before the templates.
	if err := d.translationProvider.Update(d); err != nil {
		return err
	}

	if err := d.templateProvider.Update(d); err != nil {
		return err
	}

	return nil
}

// New initializes a Dep struct.
// Defaults are set for nil values,
// but TemplateProvider, TranslationProvider and Language are always required.
func New(cfg DepsCfg) (*Deps, error) {
	var (
		logger = cfg.Logger
		fs     = cfg.Fs
	)

	if cfg.TemplateProvider == nil {
		panic("Must have a TemplateProvider")
	}

	if cfg.TranslationProvider == nil {
		panic("Must have a TranslationProvider")
	}

	if cfg.Language == nil {
		panic("Must have a Language")
	}

	if logger == nil {
		logger = loggers.NewErrorLogger()
	}

	if fs == nil {
		// Default to the production file system.
		fs = hugofs.NewDefault(cfg.Language)
	}

	ps, err := helpers.NewPathSpec(fs, cfg.Language)

	if err != nil {
		return nil, err
	}

	fileCaches, err := filecache.NewCaches(ps)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create file caches from configuration")
	}

	resourceSpec, err := resource.NewSpec(ps, fileCaches, logger, cfg.OutputFormats, cfg.MediaTypes)
	if err != nil {
		return nil, err
	}

	contentSpec, err := helpers.NewContentSpec(cfg.Language)
	if err != nil {
		return nil, err
	}

	sp := source.NewSourceSpec(ps, fs.Source)

	timeoutms := cfg.Language.GetInt("timeout")
	if timeoutms <= 0 {
		timeoutms = 3000
	}

	distinctErrorLogger := helpers.NewDistinctLogger(logger.ERROR)

	d := &Deps{
		Fs:                  fs,
		Log:                 logger,
		DistinctErrorLog:    distinctErrorLogger,
		templateProvider:    cfg.TemplateProvider,
		translationProvider: cfg.TranslationProvider,
		WithTemplate:        cfg.WithTemplate,
		PathSpec:            ps,
		ContentSpec:         contentSpec,
		SourceSpec:          sp,
		ResourceSpec:        resourceSpec,
		Cfg:                 cfg.Language,
		Language:            cfg.Language,
		FileCaches:          fileCaches,
		BuildStartListeners: &Listeners{},
		Timeout:             time.Duration(timeoutms) * time.Millisecond,
		globalErrHandler:    &globalErrHandler{},
	}

	if cfg.Cfg.GetBool("templateMetrics") {
		d.Metrics = metrics.NewProvider(cfg.Cfg.GetBool("templateMetricsHints"))
	}

	return d, nil
}

// ForLanguage creates a copy of the Deps with the language dependent
// parts switched out.
func (d Deps) ForLanguage(cfg DepsCfg) (*Deps, error) {
	l := cfg.Language
	var err error

	d.PathSpec, err = helpers.NewPathSpecWithBaseBaseFsProvided(d.Fs, l, d.BaseFs)
	if err != nil {
		return nil, err
	}

	d.ContentSpec, err = helpers.NewContentSpec(l)
	if err != nil {
		return nil, err
	}

	// The resource cache is global so reuse.
	// TODO(bep) clean up these inits.
	resourceCache := d.ResourceSpec.ResourceCache
	d.ResourceSpec, err = resource.NewSpec(d.PathSpec, d.ResourceSpec.FileCaches, d.Log, cfg.OutputFormats, cfg.MediaTypes)
	if err != nil {
		return nil, err
	}
	d.ResourceSpec.ResourceCache = resourceCache

	d.Cfg = l
	d.Language = l

	if err := d.translationProvider.Clone(&d); err != nil {
		return nil, err
	}

	if err := d.templateProvider.Clone(&d); err != nil {
		return nil, err
	}

	d.BuildStartListeners = &Listeners{}

	return &d, nil

}

// DepsCfg contains configuration options that can be used to configure Hugo
// on a global level, i.e. logging etc.
// Nil values will be given default values.
type DepsCfg struct {

	// The Logger to use.
	Logger *loggers.Logger

	// The file systems to use
	Fs *hugofs.Fs

	// The language to use.
	Language *langs.Language

	// The configuration to use.
	Cfg config.Provider

	// The media types configured.
	MediaTypes media.Types

	// The output formats configured.
	OutputFormats output.Formats

	// Template handling.
	TemplateProvider ResourceProvider
	WithTemplate     func(templ tpl.TemplateHandler) error

	// i18n handling.
	TranslationProvider ResourceProvider

	// Whether we are in running (server) mode
	Running bool
}
