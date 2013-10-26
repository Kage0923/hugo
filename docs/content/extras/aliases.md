---
title: "Aliases"
date: "2013-07-09"
aliases:
    - /doc/redirects/
    - /doc/alias/
    - /doc/aliases/
groups: ["extras"]
groups_weight: 20
---

For people migrating existing published content to Hugo theres a good chance
you need a mechanism to handle redirecting old urls.

Luckily, this can be handled easily with aliases in Hugo.

## Example
**content/posts/my-awesome-blog-post.md**

    ---
    aliases:
        - /posts/my-original-url/
        - /2010/even-earlier-url.html
    ---

Now when you go to any of the aliases locations they
will redirect to the page.

## Important Behaviors

1. *Hugo makes no assumptions about aliases. They also don't change based
on your UglyUrls setting. You need to provide absolute path to your webroot and the
complete filename or directory.*

2. *Aliases are rendered prior to any content and will be overwritten by
any content with the same location.*
