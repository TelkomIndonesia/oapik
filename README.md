# OpenAPI Keren

OpenAPI is awesome (Indonesian: keren), so we created several utilities for working with it.

## Features

### Bundle

Bundle splitted Open API files into one file while trying to persist all their use of `$ref`.

```bash
go run -mod=mod github.com/telkomindonesia/oapik/cmd bundle <path-to-main-spec> <path-to-generated-spec>
```

For testing the functionality, you can use [spec inside testdata directory](./internal/bundle/testdata/profile/), which will produce file(s) similar to [testoutput directory](./internal/bundle/testoutput).

### Proxy

Create a new spec by picking operations from other specs. The main purpose was to derive an OpenAPI spec for a lean [backend-for-frontends](https://microservices.io/patterns/apigateway.html) using OpenAPI spec of services behind it. It introduces a new `x-proxy` extension. It can also generate ready-to-use code which is inspired by oapi-codegen.

```bash
go run -mod=mod github.com/telkomindonesia/oapik/cmd proxy <path-to-proxy-spec> <path-to-generated-spec> [<path-to-generated-go-file>]
```

For testing the functionality, you can use [specs inside testdata directory](./internal/proxy/testdata/), which will produce file(s) similar to [testoutput directory](./internal/proxy/testoutput).

## Limitations

This utilities are still in a very early development stage. Current limitations includes but not restricted to:

- When [bundling](#bundle), all components on non-root files are required to be defined under `components` key [accordingly](https://swagger.io/specification/#components-object).
