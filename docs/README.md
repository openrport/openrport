# About documentation

The documentation system is based on [Hugo](https://gohugo.io/) and the [Geekdocs Theme](https://geekdocs.de/)

## Write
Use markdown with the [shortcodes from Geekdocs](https://geekdocs.de/shortcodes/).

Start a local Hugo server to get a live preview of your changes
```shell
cd docs
hugo server
```

## Check
Before pushing, check your file with [markdownlint-cli](https://github.com/igorshubovych/markdownlint-cli)
```shell
npx markdownlint-cli content/
```