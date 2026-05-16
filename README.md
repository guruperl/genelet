# Genelet Go

Genelet Go is a small but self-contained Go web framework for JSON-described CRUD-style web applications. A Genelet app keeps its runtime contract in `conf/config.json` and per-component `component.json` files, then supplies generated or hand-written `Filter` and `Model` structs for each component.

The framework includes request routing, SQL-backed CRUD helpers, template rendering, hardened cookies, signed auth/session behavior, issuer-based login flows, OAuth1/OAuth2 helpers, CSRF checks for mutating requests, email helpers, and stable numeric framework errors. It is intended to support generated applications without requiring a larger Go web framework.

## How It Works

Requests are routed by path:

```text
<Script>/<role>/<tag>/<component>
<Script>/<role>/<tag>/<component>/<id>
```

`config.json` defines the script path, roles, chartags, authentication providers, templates, document root, upload settings, and database connection settings. Each component's `component.json` defines tables, keys, available actions, validation, aliases, groups, fields, foreign-key helpers, and optional next-page model calls.

For a component named `question`, the application registers factories that return request-local model and filter values:

```go
controller.ModelFactories["question"] = func() interface{} { return &question.Model{} }
controller.FilterFactories["question"] = func() interface{} { return &question.Filter{} }
```

At request time the controller parses the URL and request body, checks auth, creates the filter and model, fills `ARGS`, optionally attaches a SQL database handle, executes the requested action, and returns JSON or renders a template depending on the chartag.

## Repository Layout

- `*.go` - framework runtime.
- `*_test.go` - package regression tests.
- `samples/` - small test sample applications, when present.
- `test.conf` - test configuration.
- `tmpl.html` - test template fixture.
- `user.tbl` - test table fixture.

## Installation

Use the public Go module:

```sh
go get github.com/guruperl/genelet
```

Import it from generated or hand-written applications:

```go
import "github.com/guruperl/genelet"
```

## Test

Run the local Go test suite:

```sh
go test ./...
```

Some database tests are skipped unless the local MySQL test configuration is present.

## Samples

The repository should include a small test sample application that exercises the public framework surface used by generated apps: config loading, component JSON loading, model/filter registration, routing, auth/session behavior, and at least one CRUD-style component.

## Using Genelet

1. Create a Go app with `conf/config.json` defining `Script`, `Template`, `Pubrole`, `Chartags`, `Roles`, and optional DB/auth settings.
2. For each component, create a `component.json` with `actions`, `current_table` or `current_tables`, and `current_key`.
3. Provide component `Filter` and `Model` structs, usually embedding `genelet.Filter` and `genelet.Model`.
4. Bootstrap `genelet.Controller` from an HTTP entrypoint.
5. Register component model/filter factories and call `http.ListenAndServe`.

Generated apps can use `no_db` for actions that do not need database work and `no_method` for actions handled entirely by filter/template behavior. JSON chartags return response bodies; HTML-like chartags render templates such as:

```text
<Template>/<role>/<component>/<action>.<tag>
```

Database support covers MySQL, PostgreSQL, and SQLite through `ConnectArray` driver names `mysql`, `postgres`/`postgresql`, and `sqlite`/`sqlite3`. PostgreSQL placeholders are rebound automatically from Genelet's internal `?` style to `$1` style. Procedure helpers use MySQL `CALL`, PostgreSQL `SELECT * FROM function(...)`, and return an unsupported error for SQLite.

## Compatibility Notes

Genelet keeps the legacy generated-app surface stable:

- JSON `config.json` and `component.json` are the runtime contract.
- Numeric framework error codes and messages are preserved.
- Cookie and signed auth/session behavior are part of the public surface.
- `ARGS`, `LISTS`, `OTHER`, existing config keys, and nextpage marker names remain part of the generated-app contract.
