# Enjin

Go toolkit for HTTP backends — server-rendered (templ + htmx + Alpine) or JSON APIs. A library, not a framework: you compose everything in your own `main`.

- HTTP engine with graceful shutdown, named routes, error/panic mapping, trusted proxies, CORS, security headers
- Sessions (cookie or bearer), roles and policy guards
- Manual validation with results that survive a redirect
- Web layer: CSRF, flash, head/assets, theme, cookie consent, redirect, Turnstile, htmx and Alpine helpers
- OpenAPI 3.1 from route registration
- Postgres (pgx), Redis/Valkey cache, object storage, distributed lock, per-IP limiter, abuse guard
- Event bus (in-memory or NATS), mail (MJML + templ, SMTP, SES, Brevo), SSE, i18n

```bash
go get github.com/daarxwalker/enjin
```

| Nested module | Contains |
|---|---|
| `github.com/daarxwalker/enjin/app/openapi` | OpenAPI document (swaggest) |
| `github.com/daarxwalker/enjin/mail/ses` | AWS SES mail driver |
| `github.com/daarxwalker/enjin/filestore/minio` | MinIO / S3 file store driver |
| `github.com/daarxwalker/enjin/bus/nats` | NATS bus driver (core or JetStream) |
| `github.com/daarxwalker/enjin/google` | Google OAuth / service account, `google/spreadsheet` |
| `github.com/daarxwalker/enjin/enjintest` | Postgres, Valkey, SeaweedFS, NATS test containers |

Runnable apps: [`examples/api`](examples/api) (JSON API) and [`examples/web`](examples/web) (server-rendered).

## Contents

- [app](#app) · [httpx](#httpx)
- [infra/db](#infradb) · [infra/cache](#infracache) · [filestore](#filestore)
- [client](#client) · [session](#session) · [security](#security) · [cookie](#cookie) · [token](#token)
- [validate](#validate)
- [web](#web) · [web/render](#webrender) · [web/head, web/assets](#webhead-webassets) · [web/flash](#webflash) · [web/csrf](#webcsrf) · [web/theme](#webtheme) · [web/consent](#webconsent) · [web/redirect](#webredirect) · [web/backlink](#webbacklink) · [web/antibot](#webantibot) · [web/htmx](#webhtmx) · [web/alpine](#webalpine)
- [localize](#localize) · [mail](#mail) · [limit](#limit) · [lock](#lock) · [abuse](#abuse) · [bus](#bus) · [sse](#sse)
- [Nested modules](#nested-modules) · [Middleware order](#middleware-order)

## app

- Engine: core middlewares (recoverer, `/healthz`, trusted proxy, optional CORS, compression), graceful shutdown on SIGINT/SIGTERM, cleanups after shutdown.
- Handlers are `func(w, r) error`; errors and panics go through one handler.
- Defaults: `0.0.0.0:80`, read header 10 s, read 30 s, write 60 s, idle 120 s, body 10 MiB.
- All `Use` calls must come before routes; a late `Use` is returned as an error from `Start()`.

```go
handler := app.NewDefaultHandler(
    app.WithHandlerLogger(logger),
    app.WithHandlerOnError(func(w http.ResponseWriter, r *http.Request, err *httpx.Err) {
        _ = httpx.JSON(w, err.Status, map[string]string{"error": http.StatusText(err.Status)})
    }),
)

engine, err := app.NewEngine(
    app.WithContext(ctx),
    app.WithRouter(app.NewRouter(chi.NewRouter(), handler)),
    app.WithPort(8080),
    app.WithMiddleware(httpx.Middleware, store.Middleware),
    app.WithTrustedProxies(proxies...),
    app.WithAllowedHosts("example.com"),
    app.WithDefaultHost("example.com"),
    app.WithCORS(cors.Options{AllowedOrigins: []string{"https://app.example.com"}}),
    app.WithSecurityHeaders(security.NewHeaders(security.WithHSTS(365*24*time.Hour))),
    app.WithServer(app.Server{WriteTimeout: 2 * time.Minute}),
    app.WithMetrics(promhttp.Handler()),
    app.WithOpsToken(opsToken),
    app.WithCleanup(func(ctx context.Context) error { return pool.Close(ctx) }),
)
if err != nil {
    return err
}
return engine.Start()
```

- Other options: `WithHost`, `WithSpec`, `WithCoreMiddlewares(false)`, `WithDevTools`, `WithProfiler`, `WithCloudflare`.
- `/metrics` and `/debug/pprof/` need `WithOpsToken` and `Authorization: Bearer <token>`.
- Without `WithHandlerOnError`: 4xx sends `err.Raw` text (keep it a public message), 5xx sends only the status text.

Routes and URLs:

```go
r := engine.Router
r.Get("productDetail", "/products/{slug}", c.renderProduct)
r.Post("productCreate", "/products", c.create)
r.Group("/admin", func(r *app.Router) {
    r.With(session.Restrict(security.RequireRole(admin))).Get("adminHome", "/", c.renderAdmin)
})

app.MustURL(ctx, "productDetail", map[string]any{"slug": "cream", "page": 2}) // → /products/cream?page=2
path, err := app.URL(ctx, "productDetail", params)                            // app.ErrRouteNotFound, app.ErrNoRouter
app.IsRouteActive(ctx, "productDetail")
```

- The same name with the same method twice is an error from `Start()`.
- Path params are inserted as they are; wrap untrusted values in `url.PathEscape`.

Static files (register on `engine.Root`, so they skip app middlewares):

```go
err := app.ServeImmutableFS(engine.Root, http.FS(webFS), "/public", "")    // max-age 1 year, immutable
err = app.ServeFS(engine.Root, http.FS(assetsFS), "/assets", "/assets", time.Hour)
err = app.ServeFile(engine.Root, assetsFS, "/favicon.ico", "img/favicon.ico", time.Hour)
```

OpenAPI ([app/openapi](#nested-modules) + `app/routespec`):

```go
app.WithSpec(openapi.New(openapi.WithTitle("Products API"), openapi.WithBearer("Token from POST /login")))

r.Post("productCreate", "/products", c.create,
    routespec.WithSummary("Create product"),
    routespec.WithTags("products"),
    routespec.WithBearer(),
    routespec.WithRequest[createProductRequest](),
    routespec.WithJSON[product](http.StatusCreated, "Created"),
    routespec.WithError(http.StatusUnprocessableEntity, "Invalid input"),
)
```

- Served at `GET /spec.json`.

## httpx

```go
httpx.JSON(w, http.StatusOK, payload)
httpx.String(w, http.StatusOK, "ok")
httpx.HTML(w, http.StatusOK, "<p>hi</p>")
httpx.NoContent(w, http.StatusNoContent)
httpx.Stream(w, "export.csv", data)
httpx.Redirect(w, r, "/done")
httpx.Refresh(w, r)

return httpx.Error(http.StatusBadRequest, errors.New(t(msgInvalid)))
return httpx.StatusError(http.StatusNotFound)

httpx.From(ctx).URL
httpx.CreateURL(r, "/verify")
httpx.HostFrom(r)
httpx.PathPrefix(r)
httpx.ClientIP(r)
httpx.IsLocalPath("/next")

hc := httpx.NewClient(httpx.WithClientTimeout(30 * time.Second))
res, err := hc.Do(req)
```

- `httpx.Middleware` must be the first app middleware.
- `X-Forwarded-*` headers count only from `app.WithTrustedProxies`.
- `Do` returns an error for status ≥ 400 with the first 64 KiB of the body.

## infra/db

```go
pool, err := db.Connect(ctx, uri, db.WithMaxConns(50), db.WithConnectTimeout(10*time.Second))

err = db.WithTx(ctx, pool, "create order", func(tx pgx.Tx) error {
    _, err := tx.Exec(ctx, query, args...)
    return err
})

columns := []db.Column[product]{
    db.ColumnText("name", func(p product) any { return p.Name }),
    db.ColumnInt("price", func(p product) any { return p.Price }),
}
persist := db.NewPersist("products", columns,
    db.WithPersistID(func(p product) string { return p.ID }),
    db.WithPersistEntities(products...),
)
err = db.BulkInsert(ctx, tx, persist)
err = db.BulkPersist(ctx, tx, persist)
```

- Repositories depend on `db.Querier` (pool and tx both satisfy it).
- `BulkPersist`: empty ID inserts, otherwise updates; chunks stay under the 65 535 parameter limit.

## infra/cache

```go
rdb, err := cache.Connect(ctx, addr, cache.WithPassword(pwd), cache.WithTLS(true))
c := cache.NewClient(cache.NewRedisDriver(rdb))

err = c.SetJSON(ctx, key, value, time.Hour)
found, err := c.GetJSON(ctx, key, &value)
found, err = c.TakeJSON(ctx, key, &value)
s, err := c.GetString(ctx, key)
n, err := c.Increment(ctx, key, 1, time.Minute)
ok, err := c.UpdateExpiration(ctx, key, time.Hour)
err = c.Delete(ctx, key)
```

- Every read returns the cache error; an outage is never a miss.
- `Take*` reads and deletes in one step (one-time tokens).

## filestore

```go
store := filestore.NewClient(minio.NewDriver(minioClient, "bucket"))

err := store.PutBytes(ctx, "images/a.png", data, "image/png")
data, found, err := store.GetBytes(ctx, "images/a.png")
files, err := store.List(ctx, "images/")
err = store.Delete(ctx, "images/a.png")
```

## client

Anonymous browser identity (`X-Client` cookie) with a per-client key/value store.

```go
cm := client.NewManager(cacheClient)

client.IDFrom(ctx)
client.AttachValue(ctx, "tour", true, 24*time.Hour)
client.ValueFrom[bool](ctx, "tour")
client.DestroyValue(ctx, "tour")
```

- Needed before `flash`, `validate` and `csrf`.

## session

```go
store := session.NewStore(cacheClient,
    session.WithRoles(map[string]*security.Role{admin.Name: admin}),
    session.WithConfig(cookie.WithTTL(30*24*time.Hour)),
    session.WithBearer(),
)

profile := session.Attach(w, r, session.WithID(user.ID), session.WithKey(user.Email), session.WithRole(admin))
session.From(ctx)
session.Exists(ctx)
session.Destroy(w, r)
token, err := store.Seed(ctx, session.WithID(id))

r.With(session.Restrict(security.RequireRole(admin))).Get("admin", "/admin", c.renderAdmin)
r.With(session.Restrict(security.AnyAccount(), session.WithRedirectName("login"), session.WithRedirectBack())).Get("account", "/account", c.renderAccount)
r.With(session.Forbid(session.WithRedirectName("home"))).Get("login", "/login", c.renderLogin)
```

- `Attach` always issues a new token (no session fixation); with no options it does nothing.
- Sliding expiration; a session removed by a parallel logout stays removed.
- `WithBearer`: `Authorization: Bearer <token>` first, then the cookie.
- `WithRedirectBack`: a rejected GET or HEAD redirects to `/login?redirect=/account`, ready for `redirect.FromQuery`; it needs a target such as `WithRedirectName`.

## security

```go
admin := &security.Role{Name: "admin", Securables: []security.Securable{"products.write"}}

security.AnyAccount()
security.AnyRole()
security.RequireRole(admin)
security.RequireSecurable("products.write")
security.And(security.AnyRole(), security.Not(security.RequireRole(guest)))
security.RestrictUUIDv7("id")
security.RequireBearer(opsToken)

security.NewHeaders(
    security.WithHSTS(365*24*time.Hour),
    security.WithCSP("default-src 'self'"),
    security.WithHeader("X-Frame-Options", "SAMEORIGIN"),
)
```

- Default headers: `nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy`, `Cross-Origin-Opener-Policy`, `Permissions-Policy`.

## cookie

```go
cfg := cookie.NewConfig(
    cookie.WithName("X-Session"),
    cookie.WithTTL(30*24*time.Hour),
    cookie.WithDomain("example.com"),
    cookie.WithSameSite(http.SameSiteLaxMode),
    cookie.WithSecure(false),
)
cfg.Attach(w, value)
cfg.Destroy(w)
```

- Defaults: path `/`, `SameSite=Lax`, `HttpOnly`, `Secure`.

## token

```go
t := token.New()
token.Valid(t)
```

## validate

```go
func parseLogin(r *http.Request) (loginRequest, validate.Result) {
    v := make(validate.Result)
    req := loginRequest{Email: r.FormValue("email")}
    v.Text("email", req.Email, 1, 255)
    v.Email("email", req.Email)
    v.UUIDv7("id", r.FormValue("id"))
    return req, v
}

if !v.Zero() {
    return httpx.JSON(w, http.StatusUnprocessableEntity, v)
}
```

Across a redirect (web):

```go
vm := validate.NewManager(cacheClient)

validate.Attach(ctx, v)
validation := validate.From(ctx)
```

- Verdicts: `Required`, `MinLen`, `MaxLen`, `Min`, `Max`, `Invalid`.
- `validate.From` reads once and deletes; without `client.Middleware` it returns nil.

## web

All web middlewares in the right order:

```go
app.WithMiddleware(web.CreateMiddlewareManager(
    web.WithClient(clientManager),
    web.WithRedirect(redirectManager),
    web.WithLocalizer(localizer),
    web.WithSession(store),
    web.WithFlash(flashManager),
    web.WithValidate(validateManager),
    web.WithTheme(themeManager),
    web.WithConsent(consentManager),
    web.WithCSRF(csrfManager),
    web.WithAssets(manifest, "/"),
)...)
```

## web/render

```go
return render.Templ(w, r, http.StatusOK, view(vm))
return render.Templ(w, r, http.StatusOK, view(vm), layout)
return render.TemplFragments(w, r, http.StatusOK, form(vm), "email")
```

## web/head, web/assets

```go
manifest, err := assets.LoadManifest(staticFS, "manifest.json")

head.Attach(ctx,
    head.WithTitle("Products"),
    head.WithDescription("All products"),
    head.WithMeta("robots", "noindex"),
)
```

```templ
<head>
    @head.Tags()
</head>
```

## web/flash

```go
fm := flash.NewManager(cacheClient)

flash.Attach(ctx, flash.WithSuccess("Saved"))
flash.Attach(ctx, flash.WithError("Failed"), flash.WithInfo("…"), flash.WithWarning("…"))
messages := flash.From(ctx)
```

- Shown on the next request that reads them (POST → redirect → GET).

## web/csrf

```go
cm := csrf.NewInputManager(csrf.NewRedisDriver(rdb),
    csrf.WithRejectHandler(rejectHandler),
)

cm.CreateInput(ctx)
csrf.NewInputFrom(ctx)

r.With(cm.Require).Post("contact", "/contact", c.send)
r.With(cm.Verify).Post("login", "/login", c.login)
if csrf.RejectedFrom(ctx) { … }

api := csrf.NewCookieManager()
r.With(api.Require).Post("order", "/orders", c.create)
```

- Single-use tokens from the form body or the `X-CSRF-Token` header, never the query string.
- `Verify` only flags the request, `Require` rejects it.
- `CookieManager`: double-submit cookie for APIs with cookie auth.

## web/theme

```go
tm := theme.NewManager(theme.WithDefaultTheme(theme.Dark))

theme.From(ctx)
theme.IsDark(ctx)
theme.Attach(w, r, theme.Light)
```

- `?theme=dark` sets the cookie and redirects without the parameter.

## web/consent

```go
cm := consent.NewManager()
consent.Mount(engine.Router, "consent")

v := consent.From(ctx)
if v.Exists && v.Cookies.Analytic { … }
consent.Attach(w, r, consent.Cookies{Analytic: true, Preference: true})
```

- The mounted `POST /__consent__` reads `action` (`all`, `necessary`, `selection`), the category fields and `redirect`.

## web/redirect

```go
rm := redirect.NewManager()

redirect.Attach(w, r, redirect.FromQuery(r))
target := redirect.FromCookie(w, r)
```

- Local paths only.

## web/backlink

```go
query, ok := backlink.Query(r, app.MustURL(ctx, "orders"))
```

- Query of a same-host `Referer` with exactly that path; re-validate it before use.

## web/antibot

```go
verifier := antibot.NewVerifier(antibot.NewTurnstileDriver(httpClient, siteKey, secret))

r.With(verifier.Require).Post("register", "/register", c.register)
verifier.CreateWidget(ctx)
if antibot.RejectedFrom(ctx) { … }
```

- The widget needs an `x-turnstile` Alpine directive in your frontend; it loads the Turnstile script on demand.
- Skipped for logged-in sessions. The token comes from the `cf-turnstile-response` field.

## web/htmx

htmx 4 attributes, headers, events and response helpers.

```go
htmx.Is(r)
htmx.AttachRedirect(w, "/done")
htmx.AttachTrigger(w, "saved")
htmx.AttachRefresh(w)
htmx.Restrict

htmx.Target(htmx.ModInherited)
htmx.Status("422")
htmx.Map(map[string]any{"id": 1})
htmx.ConfigMeta(map[string]any{htmx.ConfigDefaultSettleDelay: 0})
```

- Inheritance is explicit (`ModInherited`); 4xx/5xx responses are swapped by default.

## web/alpine

```go
alpine.Data, alpine.Show, alpine.Text, alpine.Model
alpine.On("click"), alpine.Bind("class")
alpine.Click(alpine.Prevent), alpine.KeyDown(alpine.Escape)
alpine.Magic.Dispatch("saved", data)
alpine.Map(m)
```

## localize

```go
l, err := localize.NewLocalizer(
    localize.WithFS(localesFS),
    localize.WithSubpath("locales"),
    localize.WithExtension("toml", toml.Unmarshal),
    localize.WithLangs(language.Czech, language.English),
    localize.WithDefaultLang(language.Czech),
)

t := localize.TranslateFrom(ctx)
t(msgGreeting, map[string]any{"Name": name})
t(msgProducts, map[string]any{"Count": n})
localize.LangFrom(ctx)
```

- Files `locales/active.<lang>.toml`; `?lang=` wins, then `Accept-Language`.
- `Count` picks the CLDR plural form (`One`, `Few`, `Other`, …).

## mail

```go
mailer := mail.NewManager(mail.NewSMTPDriver(host, port, user, pass))

body, err := mail.Render(ctx, loginMail(link))
err = mailer.Send(ctx, mail.Message{
    From:        "Example <no-reply@example.com>",
    To:          []string{email},
    Bcc:         []string{"audit@example.com"},
    Subject:     "Sign in",
    HTML:        body,
    Text:        "Sign in: " + link,
    Embeds:      []mail.File{{Name: "logo", Bytes: logo}},
    Attachments: []mail.File{{Name: "invoice.pdf", ContentType: "application/pdf", Bytes: pdf}},
})
```

- `mail.Render` compiles MJML inside a templ component to HTML.
- SMTP options: `mail.WithSMTPEncryption(mail.SMTPEncryptionStartTLS)` (also `None`, `ImplicitTLS`; default `Auto` = implicit TLS on 465, otherwise STARTTLS when offered), `WithSMTPHelo(name)` (default `os.Hostname()`), `WithSMTPTLSConfig(cfg)`, `WithSMTPOAuth2(tokenSource)`, `WithSMTPTimeout(d)` (default 30 s), `WithSMTPLogger(log)`.
- Empty username skips auth (Mailpit, IP relay). Otherwise PLAIN, LOGIN or CRAM-MD5 by what the server offers; a cleartext password goes only over TLS or to localhost.
- Once the server accepts the message, a failed QUIT is only logged, so a retry does not send it twice.
- SES driver in [mail/ses](#nested-modules).
- Brevo API driver in `mail/brevo` (core module): `mail.NewManager(brevo.NewDriver(apiKey))`. No embeds and one Reply-To at most; for those use the Brevo SMTP relay: `mail.NewSMTPDriver("smtp-relay.brevo.com", "587", login, smtpKey)`.

## limit

Per-IP counter of an action (e.g. one registration per day).

```go
limiter := limit.NewLimiter(cacheClient, "register", 1, limit.WithTTL(24*time.Hour))

r.With(limiter.Verify).Post("register", "/register", c.register)

r.With(limiter.Load).Post("export", "/export", c.export)
allowed, err := limiter.Consume(ctx)
limiter.Current(ctx)
limiter.ResetIn(ctx)
```

- `Verify` answers `429` over the limit; skipped for logged-in sessions.

## lock

```go
locker := lock.NewManager(lock.NewRedisDriver(rdb), 30*time.Second)

lease, err := locker.Lock(ctx, "sync", productID)
defer locker.Unlock(ctx, lease)
```

## abuse

Route guard against automated traffic.

```go
store := abuse.NewMemoryStore(ctx)
guard := abuse.NewGuard(store,
    abuse.WithBotVerifier(abuse.NewBotVerifier(store)),
    abuse.WithGoodBotPolicy(abuse.Policy{Name: "bot", Rules: []abuse.Rule{
        abuse.CreateRateLimitRule(abuse.CountedByIP, 600, time.Minute, abuse.VerdictDelay),
    }}),
    abuse.WithRecorder(recorder), // RecordEvent(ctx, abuse.Event)
)

login := abuse.Policy{Name: "login", Rules: []abuse.Rule{
    abuse.CreateUserAgentRule(abuse.VerdictBlock),
    abuse.CreateRateLimitRule(abuse.CountedByIP, 20, time.Minute, abuse.VerdictDelay),
    abuse.CreateRateLimitRule(abuse.CreateCountedByFormField("email"), 5, time.Hour, abuse.VerdictBlock),
}}
products := abuse.Policy{Name: "products", Rules: []abuse.Rule{
    abuse.CreateDistinctURLParamRule(abuse.CountedByClientID, "slug", 200, time.Hour, abuse.VerdictBlock),
}}

r.With(guard.Protect(login)).Post("login", "/login", c.login)
r.With(guard.Protect(products)).Get("product", "/products/{slug}", c.renderProduct)

snapshot, err := store.MarshalBinary()
err = store.UnmarshalBinary(snapshot)
```

- Counted by `CountedByIP`, `CountedByClientID`, `CountedByUserID` or a form field.
- Verdicts: `VerdictAllow`, `VerdictDelay`, `VerdictBlock` (`429` with `Retry-After`).
- Repeated blocks within one policy escalate to a 1 h, then 24 h IP block.
- `Policy.DryRun` only logs; other options: `WithBlockHandler`, `WithMetrics`, `WithLogger`.
- `WithRecorder` hands every delay and block (dry runs too) to the app as an `abuse.Event`: time, IP and its counted prefix, policy, rule (`abuse.RulePenalty` for an escalated block), client, user, method, path, user agent, retry after. It runs on the request path, so queue the event and store it elsewhere. The event carries the raw IP (the log keeps only a daily hash), so retention is up to the app.

## bus

```go
var LoginMail = bus.NewEvent[loginMail](16)
var Progress = bus.NewEvent[progress](16, bus.WithSubject("sync.progress"))

err := LoginMail.Publish(ctx, loginMail{To: email})
err = Progress.TryPublish(progress{Done: 10})

ch, err := LoginMail.Subscribe(8)
defer LoginMail.Unsubscribe(ch)
for msg := range ch { … }

err = bus.Use(natsDriver)
err = LoginMail.Close(ctx)
```

- In-memory unless the event has a subject and a driver is installed with `bus.Use`.
- `TryPublish` returns `bus.ErrFull` instead of waiting.

## sse

```go
flusher, err := sse.Begin(w)
err = sse.Event(w, flusher, "progress", payload)
err = sse.Data(w, flusher, payload)
```

## Nested modules

```go
app.WithSpec(openapi.New(openapi.WithTitle("API"), openapi.WithVersion("1.0.0")))

sesClient, err := ses.Connect(ctx, region, accessKey, secret)
mailer := mail.NewManager(ses.NewDriver(sesClient))

minioClient, err := minio.Connect(endpoint, accessKey, secretKey)
store := filestore.NewClient(minio.NewDriver(minioClient, "bucket"))

nc, err := nats.Connect(uri)
driver, err := nats.NewDriver(ctx, nc, nats.WithPrefix("myapp"), nats.WithJetStream("MYAPP", 24*time.Hour))
err = bus.Use(driver)

saClient, err := google.NewSAClient(google.NewSAConfig(projectID, privateID, privateKey, clientID, clientEmail, certURL), sheets.SpreadsheetsScope)
sheet, err := spreadsheet.GetSheet(ctx, sheetsService, spreadsheetID, "Products")
```

Tests (Docker; skipped with `go test -short`):

```go
pool := enjintest.NewPostgres(ctx, t, enjintest.WithMigrationsFS(migrations.FS))
rdb := enjintest.NewValkey(ctx, t)
s3 := enjintest.NewSeaweedFS(ctx, t, "bucket")
nc := enjintest.NewNATS(ctx, t)
srv := enjintest.NewServer(router)
```

## Middleware order

Web:

```
httpx → your middlewares → client → redirect → localize → session → flash → validate → theme → consent → csrf → head → assets
```

JSON API:

```
httpx → your middlewares → localize → session
```

---

[RELEASE.md](RELEASE.md) · [SECURITY.md](SECURITY.md) · [CHANGELOG.md](CHANGELOG.md) · [LICENSE](LICENSE)
