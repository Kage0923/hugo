
---
date: 2018-03-07
title: "0.37.1"
description: "0.37.1"
categories: ["Releases"]
images:
- images/blog/hugo-bug-poster.png

---

	

This is a bug-fix release with a one important fix:

Image content such as `SVG` cannot be scaled with the built-in image processing methods, but it should still be possible to use them as page resources. This was a regression in Hugo `0.37` and is now fixed. [ba94abbf](https://github.com/gohugoio/hugo/commit/ba94abbf5dd90f989242af8a7027d67a572a6128) [@bep](https://github.com/bep) [#4455](https://github.com/gohugoio/hugo/issues/4455)






