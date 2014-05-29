---
aliases:
- /layout/chrome/
date: 2013-07-01
menu:
  main:
    parent: layout
next: /templates/rss
prev: /templates/views
title: Partial Templates
weight: 80
---

It's not a requirement to have this, but in practice it's very
convenient to split out common template portions into a partial template
that can be included anywhere. As you create the rest of your templates
you will include templates from the /layout/partials directory. Hugo
doesn't know anything about partials, it's simply a convention that you
may likely find beneficial.


I've found it helpful to include a header and footer template in
partials so I can include those in all the full page layouts.  There is
nothing special about header.html and footer.html other than they seem
like good names to use for inclusion in your other templates.

    ▾ layouts/
      ▾ partials/
          header.html
          footer.html

By ensuring that we only reference [variables](/layout/variables/)
used for both nodes and pages we can use the same partials for both.

## example header.html
This header template is used for [spf13.com](http://spf13.com).

    <!DOCTYPE html>
    <html class="no-js" lang="en-US" prefix="og: http://ogp.me/ns# fb: http://ogp.me/ns/fb#">
    <head>
        <meta charset="utf-8">

        {{ template "partials/meta.html" . }}

        <base href="{{ .Site.BaseUrl }}">
        <title> {{ .Title }} : spf13.com </title>
        <link rel="canonical" href="{{ .Permalink }}">
        {{ if .RSSlink }}<link href="{{ .RSSlink }}" rel="alternate" type="application/rss+xml" title="{{ .Title }}" />{{ end }}

        {{ template "partials/head_includes.html" . }}
    </head>
    <body lang="en">

## example footer.html
This header template is used for [spf13.com](http://spf13.com).

    <footer>
      <div>
        <p>
        &copy; 2013 Steve Francia.
        <a href="http://creativecommons.org/licenses/by/3.0/" title="Creative Commons Attribution">Some rights reserved</a>; 
        please attribute properly and link back. Hosted by <a href="http://servergrove.com">ServerGrove</a>.
        </p>
      </div>
    </footer>
    <script type="text/javascript">

      var _gaq = _gaq || [];
      _gaq.push(['_setAccount', 'UA-XYSYXYSY-X']);
      _gaq.push(['_trackPageview']);

      (function() {
        var ga = document.createElement('script');
        ga.src = ('https:' == document.location.protocol ? 'https://ssl' : 
            'http://www') + '.google-analytics.com/ga.js';
        ga.setAttribute('async', 'true');
        document.documentElement.firstChild.appendChild(ga);
      })();

    </script>
    </body>
    </html>

**For examples of referencing these templates, see [single content
templates](/templates/content), [list templates](/templates/list) and [homepage templates](/templates/homepage)**
