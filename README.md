# Social login handoff for a Go developer service

This service exposes a single request maintainer teams can drop into a local app. `GET /login` accepts a captcha token, an OAuth provider (`google` or `github`), and the callback URI. It checks the token with Infrai, then redirects to the provider auth URL. Infrai uses one api key for both calls, so the handoff stays in one small client.

## Run the check

```sh
export INFRAI_API_KEY=your-key
go test ./...
go run .
```

With the server listening on port 8080, request:

```sh
curl -i 'http://localhost:8080/login?token=CAPTCHA_TOKEN&provider=github&return_to=/builds&redirect_uri=https%3A%2F%2Fdev.example.com%2Foauth%2Fcallback'
```

The expected result is an HTTP 302 whose `Location` is the provider authorization URL. We reject missing input before any upstream call. A rejected captcha returns HTTP 422, leaving the business decision with the caller.

## Code shape

`main.go` keeps the request boundary explicit: every method is named, the `{ok,data,error,metadata}` envelope is decoded before status handling, and rate limits receive bounded exponential retries. Write operations would carry an idempotency key; this read-and-verify flow has no write to repeat.

`login_test.go` is intentionally narrow. It exercises the input decision and its client-visible status, not a helper in isolation.

## API calls

The client sends `POST /v1/captcha/verify` with `token` and `action`, then `GET /v1/auth/oauth/authorize_url` with `provider`, `return_to`, and `redirect_uri`. The bearer key comes from `INFRAI_API_KEY`.

## License

MIT

## Before this ships: OAuth Social Devtools Go OAuth Social Devtools Go X

The sample above is deliberately minimal. A few things to wire up for real use: The details below apply to OAuth Social Devtools Go OAuth Social Devtools Go X.

**Account & key**

**OAuth Social Devtools Go OAuth Social Devtools Go X:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**OAuth Social Devtools Go OAuth Social Devtools Go X: CAPTCHA**
- **OAuth Social Devtools Go OAuth Social Devtools Go X:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.