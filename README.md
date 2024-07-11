io
==

A micro blogging platform written in go.

Put posts in `posts/` and static content in `static`.

Run the usual way, put it behind `nginx`, whatever. Should be secure enough. No
guarantees.

Example
-------

```
---
title: "Lorem Ipsum"
date: "2024-07-11T16:07:51+02:00"
tags: foo, bar
draft: true  # if `true` the post won't show up in the index. default: `false` 
---

Lorem ipsum blah blah.

![foo bar](/static/img/foo/bar.png)

It even supports footnotes[^1].

[^1]: Yeah, for real. One line per footnote though. No line breaks. Even if the note ends up being very very very long. Yeah? Yeah.
```

Save it in `posts/blah.md` and if `draft` is `false` you'll see it
in the index. Magic.