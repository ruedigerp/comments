

# Binary release build: 

[![Build artifacts & Release](https://github.com/ruedigerp/comments/actions/workflows/release.yaml/badge.svg)](https://github.com/ruedigerp/comments/actions/workflows/release.yaml)


# Docker builds 

* Main: [![(prod) Build, Package, Release](https://github.com/ruedigerp/comments/actions/workflows/build-prod.yaml/badge.svg)](https://github.com/ruedigerp/comments/actions/workflows/build-prod.yaml)


# Dokumentation und Installation

* [Install doc](docs/README.md)
* [Docker-compose](docs/docker-compose/README.md)
* Kuberenetes
* [helm](docs/helm/README.md)
* [FluxCD Installation](docs/fluxcd/)

* [API Docs](docs/api/README.md)

* [Redis Command](docs/redis/README.md)

# Einbinden in ein CMS / Theme

## Die Dtaei `comment-widget.js` im Theme Folder

Die Datei `comment-widget.js` im Javascript Ordner ablegen und Append im Header der Seite laden. 

```html
<!DOCTYPE html>
<html>
<head>
...
<script src="/js/comment-widget.js"></script>
</head>
...
```

## Empfohlen: vom Append Server herunterladen 

Wenn der Comment Server aktualisiert wird und sich die `comment-widget.js` geändert hat, wird immer die aktuelle Version genutzt. Das Theme muss in diesem Fall nicht angepasst werden. 

```html
<script src="https://comments.example.com/static/js/comment-widget.js"></script>
```


# Einfügen im Template

Im Template `comments` aktivieren wo es angezeigt werden soll. 
Zum Beispiel mit dem Link zur aktuellen Seite. 

```html
<script>
    CommentWidget.init({{.Link}});
</script>
```

Das `{{.Link}}` ist hier speziefisch für [InkPaper, a static blog generator](https://github.com/InkProject/ink) . Je nach eingesetztem CMS muss diese Variable angepasst werden. 

Comments kann auch für mehrer Seiten/Blogs genutzt werden. Dafür kann man einfach eine eindeutige Blog-ID nutzen. Zum Beispiel: `blog-exmaple-net` oder `123`, die dann vor der Post-ID gesetzt wird: 

```html
<script>
    CommentWidget.init(blog-example-net/{{.Link}});
</script>
```

oder: 

```html
<script>
    CommentWidget.init(123/{{.Link}});
</script>
```


## Alternative: 

Auf jeder Seite oder jedem Artikel direkt den Code mit der entsprechenden ID eintragen:

```html
# Date + Title
<div data-comment-post-id="2025-06-19-git-merge-script"></div>
# or Title
<div data-comment-post-id="git-merge-script"></div>
# or URL Path 
<div data-comment-post-id="posts/2025-06-19-git-merge-script.html"></div>
```

Mit Blog-ID bei der Nutzung auf mehreren Seite (siehe oben): 

```html
# Date + Title
<div data-comment-post-id="blog-example-net/2025-06-19-git-merge-script"></div>
# or Title
<div data-comment-post-id="blog-example-net/git-merge-script"></div>
# or URL Path 
<div data-comment-post-id="blog-example-net/posts/2025-06-19-git-merge-script.html"></div>
```

oder: 

```html
# Date + Title
<div data-comment-post-id="123/2025-06-19-git-merge-script"></div>
# or Title
<div data-comment-post-id="123/git-merge-script"></div>
# or URL Path 
<div data-comment-post-id="123/posts/2025-06-19-git-merge-script.html"></div>
```


