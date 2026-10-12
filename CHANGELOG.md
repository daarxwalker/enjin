# Changelog

Changes since the last release, grouped under `### Added`, `### Changed`,
`### Fixed` and `### Security`. `task release` publishes this section as the
release notes (annotation of the core tag and the GitHub release) and empties
it; notes of past releases live on the GitHub releases page.

## [Unreleased]

First public release. `v0.1.0` is retracted and republished as `v0.1.1`. The
entries below describe the changes from the internal version that was used only
inside a monorepo workspace.

### Added

- `localize.Translate` selects the plural form from the `Count` template value
  when the message defines plural forms (`One`, `Few`, …), following the
  language's CLDR rules. Messages with only `Other` are unaffected.
- `abuse.MemoryStore` implements `encoding.BinaryMarshaler` and
  `BinaryUnmarshaler`: a gob snapshot of the live counters, distinct sets,
  penalties and marks (expired entries are skipped), to save on shutdown and
  restore before serving. Distinct set members are hashed with a salted FNV-1a
  carried in the snapshot, so members seen before a restart still match.
- `app.WithCleanup` can be passed several times; all cleanups run in order and
  their errors are joined.
- `cache.Commander.TakeString` and `TakeJSON` (`Driver.Take`, Redis `GETDEL`)
  read and delete a key in one step, so a one-time token is consumed by exactly
  one request.
- `app.WithServer(app.Server)`: `ReadHeaderTimeout`, `ReadTimeout`,
  `WriteTimeout`, `IdleTimeout`, `MaxHeaderBytes` and `MaxBodyBytes`, with
  defaults `10s`, `30s`, `60s`, `120s`, `1 MiB` and `10 MiB`. The body limit is
  a `Root` middleware answering `413` for an oversized `Content-Length`.
- `sse.Begin` clears the write deadline so streams outlive `WriteTimeout`.
- `security.Headers` with `app.WithSecurityHeaders`: opt-in response headers
  (`nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy`,
  `Cross-Origin-Opener-Policy`, `Permissions-Policy`) on every response,
  static files included. `security.WithHSTS`, `security.WithCSP`,
  `security.WithCSPReportOnly`, `security.WithPermissionsPolicy` and
  `security.WithHeader` add, override or drop headers.
- `httpx.Proxy` with `app.WithTrustedProxies` and `app.WithAllowedHosts`:
  forwarded headers are accepted only from trusted peers, the client IP is the
  right-most untrusted `X-Forwarded-For` entry, and unknown hosts get `421`.
- Rejecting `Require` middleware for `csrf.InputManager`, `csrf.CookieManager`
  and `antibot.Verifier`, next to the flag-only `Verify`. `csrf.WithRejectHandler`
  and `antibot.WithRejectHandler` replace the default `403` response.
- `web.CreateMiddlewareManager` returns the web middleware stack in the right order from
  the managers passed as options (`web.WithClient`, `web.WithSession`, …).
- `httpx.ContextValue.Host` carries `httpx.HostFrom(r)`, so templ components
  can read the request host from `ctx`.
- `head.Tags()` templ component renders the title, metas, links and scripts
  attached with `head.Attach`.
- `head.From` returns an empty value instead of nil without
  `head.Middleware`, so a layout cannot panic on it.
- `httpx.ParsePrefixes` parses trusted proxy ranges from configuration
  (bare IPs allowed, empty values skipped).
- `csrf.InputManager` reads the token from the `X-CSRF-Token` header when the
  form field is missing.
- `csrf.CookieManager`: double-submit cookie checked against the
  `X-CSRF-Token` header, for htmx and JSON clients.
- `app.WithMetrics(http.Handler)`, `app.WithProfiler`, `app.WithOpsToken`: ops
  routes (`/metrics`, `/debug`) behind a bearer token and a rate limit.
- `app.WithDevTools` and `app.WithCleanup`.
- `app/routespec`: dependency-free operation description; `app.Spec` port and
  `app.WithSpec`.
- `mail` MIME composer: HTML and text parts, Cc, Bcc, inline embeds and
  attachments, `Envelope` with all recipients.
- `httpx.ClientIP`, `httpx.IsLocalPath`, `httpx.Err.Unwrap`.
- `internal/testredis` (miniredis) for tests that need Redis.
- `examples/api` and `examples/web` (module `enjin/examples`): runnable JSON
  API and server-rendered app with smoke tests that walk the whole flow.
- `session.WithBearer`: the session middleware reads
  `Authorization: Bearer <token>` before the cookie. Bearer requests get no
  `Set-Cookie` and `session.Attach` keeps their token.
- `routespec.WithBearer` and `openapi.WithBearer` describe bearer auth in the
  OpenAPI document.
- `app.WithCORS(cors.Options)` runs `github.com/go-chi/cors` among the core
  middlewares, right after the proxy.
- `app.WithDefaultHost` (`httpx.WithDefaultHost`): the host `httpx.HostFrom`
  and `httpx.CreateURL` use when the request carries none. Without a host
  `CreateURL` returns just the path instead of `http:///…`.
- `cache.Driver.UpdateExpiration` and `cache.Client.UpdateExpiration` extend
  the TTL of an existing key and report whether it existed (Redis `PEXPIRE`).

### Changed

- The module path is `github.com/daarxwalker/enjin`; nested modules sit below
  it (`github.com/daarxwalker/enjin/mail/ses`, …) and require the core and
  `enjintest` at the released version.
- `httpx.HostFrom`, `httpx.CreateURL` and `httpx.PathPrefix` read
  `X-Forwarded-Host/Proto/Prefix` only for requests that `httpx.Proxy`
  accepted from a trusted proxy. Without the proxy middleware the headers are
  ignored.
- Dependencies with known vulnerabilities updated: `google.golang.org/grpc`
  v1.83.1 (`enjin/google`), `github.com/moby/go-archive` v0.3.0
  (`enjin/enjintest`).

- Heavy SDKs moved to nested modules: `enjin/app/openapi` (swaggest),
  `enjin/mail/ses` (AWS SES), `enjin/filestore/minio` (MinIO),
  `enjin/bus/nats` (NATS), `enjin/google` (Google APIs and
  `google/spreadsheet`), `enjin/enjintest` (testcontainers). The core module no
  longer depends on any of them or on Prometheus.
- The root `infra` package is gone: use `db.Connect`, `cache.Connect`,
  `minio.Connect`, `ses.Connect`, `nats.Connect` and the `google` constructors.
- `app.NewEngine` returns an error and `Engine.Start` returns an error instead
  of calling `log.Fatal`. Cleanup runs after `server.Shutdown`.
- Enjin no longer reads `APP_ENV`. Dev behaviour is explicit
  (`app.WithDevTools`, `cookie.WithSecure`).
- `mail.Driver.Send(ctx, Envelope, raw)`; `mail.Message` has `HTML`, `Text`,
  `Cc`, `Bcc` and `Attachments` instead of `Body` and `ContentType`.
- `head.ContextValue.Lang` removed; it was never set or read.
- `cache.Client.Exists`, `TTL`, `GetString` and `GetJSON` return an error, so a
  Redis outage is no longer reported as a cache miss.
- `limit.Manager` takes the maximum as a required argument and counts
  atomically per client IP without the port.
- Templ rendering moved from `httpx` to `web/render`.
- Context values of `csrf`, `redirect`, `client` and `session` are unexported;
  use the package accessors.
- `session` slides an active session by extending the TTL of the cached
  profile instead of rewriting it on every request. A cache outage skips the
  session for that request and keeps the cookie; only an undecodable profile
  or one with an unknown role is destroyed. A cookie of a session that no
  longer exists is cleared.
- `abuse`: the IP penalty grows only on `VerdictBlock` (a delay no longer
  escalates it) and is kept per policy, so a block from one policy does not
  lock the IP out of other protected routes.
- `app.URL` returns `(string, error)` with `app.ErrRouteNotFound` and
  `app.ErrNoRouter` instead of panicking or returning the text of the error
  as the URL. `app.MustURL` keeps the old one-value form and panics on error.
- `flash.Attach` and `head.Attach` take a `context.Context` instead of
  `*http.Request`. Functions that write a cookie keep `(w, r, …)`, the others
  take `ctx`.
- `validate.ContextValue` is unexported. `theme.Manager` and `consent.Manager`
  no longer embed `*cookie.Config`, so its fields and `Attach`/`Destroy` are not
  part of their API.
- `bus.Event.Subscribe` returns `(<-chan T, error)`: `bus.ErrClosed` after
  `Close`, or the driver error when the remote subscription fails (the next
  `Subscribe` tries again). It used to return a closed channel silently.
- `app.Engine` mounts the ops routes in `Start()` instead of `NewEngine`, so
  `Root.Use` keeps working after `NewEngine`. `app.Router.Use` after a route
  no longer panics: the error is recorded and returned by `Start()`.
- `mail.SMTPDriver.Send` honours the context and has a 30 s connection
  timeout; it negotiates STARTTLS and AUTH like `smtp.SendMail`.
- `csrf.WithTTL` and `csrf.WithLimit` ignore values that are not positive.
- `abuse.MemoryStore.HitCounter` and `CountDistinct` return
  `abuse.ErrInvalidWindow` for a window that is not positive instead of
  panicking; the guard logs it and lets the request through.

### Fixed

- `web/redirect` never returned the stored value, so redirect after login did
  not work.
- `db.WithTx` wrapped the wrong error on rollback and did not roll back on
  panic.
- `app.Router.Group` panicked on route registration, and nested groups lost
  their prefix.
- `db` bulk helpers: empty inserts, rollback of transactions they do not own,
  nil ID without `WithPersistID`, missing chunking over the parameter limit.
- `session` did not persist `Profile.Values`.
- `validate.Text` counted bytes instead of runes.
- `localize` did not parse `Accept-Language` with quality values and accepted
  unsupported `?lang=` values.
- `bus`: deadlock between a blocked publisher and `Unsubscribe`, panic on
  `Publish` after `Close`.
- `web/theme` ignored the configured cookie name and default theme, trusted
  any cookie value and fell back to `Light` instead of the default for an
  invalid `?theme=`. Unknown values from the cookie, the query,
  `WithDefaultTheme` and `Attach` are now ignored.
- `web/assets` added CSS and JS in random order; they now follow the sorted
  manifest keys.
- `sse.Data` split data only on `\n`. A `\r` or `\r\n` in the data ended the
  event early or let the data inject its own fields (`event:`, `id:`); all line
  breaks now become separate `data:` lines. `sse.Event` rejects an event name
  with a line break.
- `validate.Result.String` (and `Error`) listed keys in random order; they are
  now sorted.
- `web/flash` saved messages with a context detached from the request, losing
  its values (trace IDs) in logs.
- `enjintest` containers were not terminated when the test passed a context
  that is canceled before cleanup (`t.Context()`).
- `client.ValueFrom` returned the zero value for anything but strings and
  booleans after a round trip through the cache (an `int` came back as
  `float64`, a struct as a map). Stored values are now decoded into the
  requested type.
- `cache.Increment` with a TTL that is not positive deleted the key right
  after incrementing it; the key now has no expiry.
- `db.WithTx` replaced the original panic with the rollback error when the
  rollback after a panic failed; both are kept now.
- `web/theme` redirected `?theme=` without the `X-Forwarded-Prefix` of a
  trusted proxy.
- `csrf.WithTTL(0)` created tokens that expired at once, so every form was
  rejected.
- `mail.SMTPDriver.Send` ignored the context and could block forever on a
  stuck server.
- `session.Destroy` left the profile in the request context, so `From` and
  `Exists` still reported a session after logout in the same request. A cache
  error or an invalid cached profile never cleared the cookie.
- `abuse.BotVerifier` did not cache a missing reverse DNS record or a failed
  lookup, so every request from such an IP with a bot user agent waited for
  DNS again. A missing record is cached like a negative result, a failed
  lookup for five minutes.
- `abuse.MemoryStore` snapshot restore raced with live traffic on the member
  salt and ignored the store capacity.
- `enjintest.NewSeaweedFS` connected to the first port the image exposes
  instead of the S3 port, so the container could not be reached on hosts that
  publish only the requested port (GitHub Actions).

### Security

- `session` rotates the token on every `Attach` (login, role change), which
  prevents session fixation.
- `lock` stores a random token and releases the lock with a compare-and-delete
  script, so a slow holder can no longer delete someone else's lock.
- `web/csrf` consumes tokens atomically and rejects on cache errors.
  `InputManager.Verify` without `InputManager.Middleware` in the stack now
  flags the request as rejected instead of letting it through unflagged.
- With `app.WithCoreMiddlewares(false)` a client could set
  `X-Forwarded-Host` and poison absolute links (`httpx.CreateURL`, e-mails);
  forwarded headers now require the trusted proxy middleware.
- `web/consent` and `web/redirect` accept only local paths (no `//host` or
  `/\host`), and the consent form is protected against cross-origin posts.
- `web/htmx` config meta and `web/alpine` helpers escape their output.
- The default error handler no longer sends internal error text for panics
  and non-`httpx.Err` errors, and without `WithHandlerOnError` it answers 5xx
  with the status text only, never `httpx.Err.Raw`.
- Connection strings and URIs with credentials are no longer logged.
- Cookies are `Secure` by default; `Domain` is written; options can turn flags
  off without being overwritten by defaults.
- `web/theme` redirected `?theme=` on a path starting with `//` to another
  host (open redirect); the redirect target is now always a local path.
- The SMTP driver never writes a `Bcc` header.
- Turnstile verification and the outbound `httpx.Client` have timeouts and a
  response body limit.
- `web/flash` and `validate` without a client ID stored everything under one
  shared cache key, so visitors could read each other's messages. Without a
  client ID they now do nothing.
- `abuse.BotVerifier` no longer trusts `.googleusercontent.com`; any Google
  Cloud VM has such a reverse DNS name and passed as a good bot.
- `abuse` form field counters store a fixed-length hash of the value, so a
  huge field value can no longer exhaust memory.
- A session deleted by logout could be written back by a parallel request of
  the same session.
- `mail` rejects an embed name or a `File.ContentType` that would inject MIME
  part headers.
- `web/csrf` reads the token only from the request body or the header, never
  from the query string, where it would leak into logs and `Referer`.
- `golang.org/x/net` v0.61.0 (GO-2026-6617, HTTP/2) in every module that uses
  it, with `x/crypto`, `x/sys`, `x/sync` and `x/text` updated alongside.
