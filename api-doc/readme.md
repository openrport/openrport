# API Documentation

This folder contains the sources of rportd API documentation following the openapi 3.0.1 standard.

If you came by here to read the API documentation go to [apidoc.rport.io](https://apidoc.rport.io/master/) to switch to the rendered HTML version.
For those preferring swagger-style rendering, use [this link](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/cloudradar-monitoring/rport/master/api-doc/openapi/openapi.yaml#/)

## Build the documentation from the sources

There a many tools out there to convert the yaml sources into different formats.  For example [Swagger Codegen](https://swagger.io/docs/open-source-tools/swagger-codegen/) or the [Open API Codegenerator](https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/5.0.0/).
Both are java command line tools.

More comfort for reading and writing Open API docs is provided by [Redoc](https://github.com/Redocly/redoc) and there command line tool [Redoc CLI](https://redocly.com/docs/redoc/deployment/cli/).
With NodeJS installed you can directly launch the tools with `npx`. See below.

### Run a local webserver

Running a local webserver is very handy for writing the documentation. Changes to the files are immediately rendered.

```shell
cd ./api-doc/openapi
npx @redocly/cli preview-docs openapi.yaml
```

### Use the linter

Before pushing changes to the repository verify the linter does not throw errors.

```shell
cd ./api-doc/openapi
npx @redocly/cli lint openapi.yaml
```

Details about the applied rules and there output can be found [here](https://redocly.com/docs/cli/resources/built-in-rules/)

The linter is integrated into the [GitHubWorkflow](../.github/workflows/apidoc.yml) and merge request are rejected if the linter throws errors or warnings.

### Render to HTML

To render the API documentation into a single dependency-free html file use:

```shell
npx redoc-cli build -o index.html openapi.yaml
```
