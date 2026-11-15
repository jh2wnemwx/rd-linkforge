# Linkforge

A URL shortener with real-time click analytics, built incrementally as the course project for **Golang for developers**.

The design is specified in the project TDD; this repository is the skeleton it describes.

## Current state

**M1–M2 implemented; the rest is skeleton.** Every seam in the architecture exists as an interface with a Null Object implementation that satisfies it and does nothing; each milestone replaces one set of `Noop*` types with real ones. So far the base62 codec, the in-memory link store and the `Shortener` service (`link.Service`) are real, and `linkforged` wires them together as a small demo. Everything else is still a `Noop*`.

Each file declares exactly one interface alongside its no-op implementation, so the compile-time contract and its placeholder stay together. Types shared by several interfaces live in the package's `model.go`.

## The Shortener

`link.Service` (in `internal/link`) is the first real implementation of `link.Shortener`, the domain service every transport sits in front of. It works against any `link.Store`; `internal/store` provides an in-memory one.

- **Shorten** validates the target (`http`/`https` only), the owner and the expiry, then stores the link under either a custom alias (3–32 chars of `[A-Za-z0-9_-]`, reserved words such as `api` and `admin` rejected) or a generated 6-character base62 code.
- **Resolve** looks a code up and returns `ErrExpired` once the link's expiry has passed.
- **List** returns an owner's links, cursor-paginated (default 20, max 100 per page).
- **Delete** removes a link, but only for its owner (`ErrForbidden` otherwise).

Try it from the command line — `linkforged` shortens every URL it is given and prints the result:

```bash
go run ./cmd/linkforged https://example.com https://go.dev
```

## Running the tests

```bash
go test ./...                  # application module
go test -race ./...            # same, with the race detector
go test ./internal/link/...    # just the Shortener
cd pkg && go test ./...        # the separate pkg/ module isn't covered by the root ./...
```

`make test` runs both modules with `-race`.

## Layout

```
cmd/      linkforged        service binary — for now a demo: shortens its URL args and lists them
          linkctl           operator CLI   — prints its version, nothing more

pkg/                        reusable libraries — a separate Go module
  shortid/  Codec           unused stub — base62 lives in internal/base62
            Generator       collision-resistant code issuing           M4
  cache/    Cache[K,V]      bounded cache on the redirect hot path     M4

internal/                   application code
  link/     Store           link persistence, declared by its consumer M1 / M7
            Shortener       the domain service behind every transport  M2
  store/    Store           in-memory link.Store implementation        M1
  base62/                   base62 encode/decode used for link codes   M1
  click/    Ingester        non-blocking submission from the hot path  M5
            Store           append-only raw event persistence          M7
            Aggregator      time series and top-N queries              M5
            Publisher       fan-out to live WebSocket subscribers      M9
  auth/     Issuer          access token minting                       M9
            Verifier        token validation for middleware            M9
            UserStore       account persistence                        M7
  transport/Server          uniform listener lifecycle                 M6 / M8
  config/   Loader          environment-only configuration             M3
  observability/Metrics     counters and histograms                    M10
```

The `pkg/` tree is a second module so that reusable libraries stay free of any dependency on application code; `go.work` ties the two together for local development.

## Design rules

These hold for every milestone.

1. **Interfaces are declared by the consumer.** `link.Store` lives in `internal/link` because the shortener is what needs it — not beside any implementation. Constructors return concrete types.
2. **The dependency arrow points inwards.** `internal/link` and `internal/click` may not import a transport or a storage package.
3. **Analytics never blocks a redirect.** `Ingester.Submit` drops events under saturation rather than applying backpressure.
4. **Every listener shares one lifecycle contract**, which is what makes a single coordinated graceful shutdown possible.

## Working on it

```bash
make            # vet, lint and test — the same gates CI runs
make build
make test
make cover
make bench
make tidy
```

Requires Go 1.25+. Linting uses [golangci-lint](https://golangci-lint.run) (`brew install golangci-lint`).

## Milestones

| # | Lessons | Scope |
|---|---|---|
| M0 | 1 | Toolchain, module layout, CI |
| M1 | 2–3 | base62 codec, `Link` model, in-memory store |
| M2 | 4–5 | Methods, interfaces, generics, error values |
| M3 | 6–7 | Files and `context`, CLI, `slog`, tests |
| M4 | 8–9 | Own hash table, LRU, trie, top-N heap |
| M5 | 10–11 | Ingestion pipeline, worker pool, synchronisation |
| M6 | 12–13 | TCP resolver, graceful shutdown, Docker |
| M7 | 14 | PostgreSQL and MongoDB stores |
| M8 | 15–16 | `net/http` first, then a framework; REST and Swagger |
| M9 | 17–19 | Validation, JWT and middleware, WebSockets |
| M10 | 20–21 | Production hardening and project defence |
