package target

import (
	"fmt"
	"io"
	"path"

	"github.com/spf13/hugo/helpers"
)

type Publisher interface {
	Publish(string, io.Reader) error
}

type Translator interface {
	Translate(string) (string, error)
}

type Output interface {
	Publisher
	Translator
}

type Filesystem struct {
	UglyUrls         bool
	DefaultExtension string
	PublishDir       string
}

func (fs *Filesystem) Publish(path string, r io.Reader) (err error) {

	translated, err := fs.Translate(path)
	if err != nil {
		return
	}

	return helpers.WriteToDisk(translated, r)
}

func (fs *Filesystem) Translate(src string) (dest string, err error) {
	if src == "/" {
		if fs.PublishDir != "" {
			return path.Join(fs.PublishDir, "index.html"), nil
		}
		return "index.html", nil
	}

	dir, file := path.Split(src)
	ext := fs.extension(path.Ext(file))
	name := filename(file)
	if fs.PublishDir != "" {
		dir = path.Join(fs.PublishDir, dir)
	}

	if fs.UglyUrls || file == "index.html" {
		return path.Join(dir, fmt.Sprintf("%s%s", name, ext)), nil
	}

	return path.Join(dir, name, fmt.Sprintf("index%s", ext)), nil
}

func (fs *Filesystem) extension(ext string) string {
	switch ext {
	case ".md", ".rst": // TODO make this list configurable.  page.go has the list of markup types.
		return ".html"
	}

	if ext != "" {
		return ext
	}

	if fs.DefaultExtension != "" {
		return fs.DefaultExtension
	}

	return ".html"
}

func filename(f string) string {
	ext := path.Ext(f)
	if ext == "" {
		return f
	}

	return f[:len(f)-len(ext)]
}
