---
title: "Index Category Example"
date: "2013-07-01"
linktitle: "Example - Categories"
groups: ["extras"]
groups_weight: 40
---

This page demonstrates an example of using indexes to provide categories for your site.

### config.yaml
First step is to define the index in your config file.
*Because we use both the singular and plural name of the index in our rendering it's
important to provide both here. We require this, rather than using inflection in
effort to support as many languages as possible.*

    ---
    indexes:
        category: "categories"
    baseurl: "http://spf13.com/"
    title: "Steve Francia is spf13.com"
    ---

### /layouts/indexes/category.html

For each index type a template needs to be provided to render the index page.
In the case of categories, this will render the content for /categories/CATEGORYNAME/.

    {{ template "chrome/header.html" . }}
    {{ template "chrome/subheader.html" . }}

    <section id="main">
      <div>
       <h1 id="title">{{ .Title }}</h1>
        {{ range .Data.Pages }}
            {{ .Render "summary"}}
        {{ end }}
      </div>
    </section>

    {{ template "chrome/footer.html" }}


### Assigning indexes to content

Make sure that the index is set in the front matter:

    {
        "title": "Hugo: A fast and flexible static site generator",
        "categories": [
            "Development",
            "golang",
            "Blogging"
        ],
        "slug": "hugo"
    }

