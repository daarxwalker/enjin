# Security Policy

## Supported versions

Enjin is pre-1.0. Only the latest `v0.x` release of each module (the core
module and every nested adapter module) receives security fixes. Fixes are not
backported to older minor versions.

## Reporting a vulnerability

Please do not open a public issue for a security problem.

Report it privately through GitHub: open the repository's **Security** tab and
choose **Report a vulnerability**. Include:

- the affected module and version (or commit),
- a description of the issue and its impact,
- steps or a minimal program that reproduces it.

You will get an acknowledgement within 7 days. Once a fix is ready, it is
released together with a GitHub security advisory that credits the reporter,
unless you ask to stay anonymous.

## Scope

In scope: code in this repository, including the middleware defaults (cookies,
CSRF, trusted proxies, allowed hosts, rate limiting, error responses) and the
nested adapter modules.

Out of scope: vulnerabilities in third-party dependencies that are already
reported upstream, and misconfiguration of an application that uses Enjin
contrary to `README.md` (for example running without `httpx.Proxy` behind a
reverse proxy, or exposing ops routes without an ops token).
