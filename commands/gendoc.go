package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/hugo/helpers"
	"github.com/spf13/hugo/hugofs"
	jww "github.com/spf13/jwalterweatherman"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const gendocFrontmatterTemplate = `---
date: %s
title: "%s"
slug: %s
url: %s
---
`

var gendocdir string
var gendocCmd = &cobra.Command{
	Use:   "doc",
	Short: "Generate Markdown documentation for the Hugo CLI.",
	Long: `Generate Markdown documentation for the Hugo CLI.

This command is, mostly, used to create up-to-date documentation
of Hugo's command-line interface for http://gohugo.io/.

It creates one Markdown file per command with front matter suitable
for rendering in Hugo.`,

	Run: func(cmd *cobra.Command, args []string) {
		if !strings.HasSuffix(gendocdir, helpers.FilePathSeparator) {
			gendocdir += helpers.FilePathSeparator
		}
		if found, _ := helpers.Exists(gendocdir, hugofs.OsFs); !found {
			hugofs.OsFs.Mkdir(gendocdir, 0777)
		}
		now := time.Now().Format(time.RFC3339)
		prepender := func(filename string) string {
			name := filepath.Base(filename)
			base := strings.TrimSuffix(name, path.Ext(name))
			url := "/commands/" + strings.ToLower(base) + "/"
			return fmt.Sprintf(gendocFrontmatterTemplate, now, strings.Replace(base, "_", " ", -1), base, url)
		}

		linkHandler := func(name string) string {
			base := strings.TrimSuffix(name, path.Ext(name))
			return "/commands/" + strings.ToLower(base) + "/"
		}

		jww.FEEDBACK.Println("Generating Hugo command-line documentation in", gendocdir, "...")
		cobra.GenMarkdownTreeCustom(cmd.Root(), gendocdir, prepender, linkHandler)
		jww.FEEDBACK.Println("Done.")
	},
}

func init() {
	gendocCmd.PersistentFlags().StringVar(&gendocdir, "dir", "/tmp/hugodoc/", "the directory to write the doc.")
}
