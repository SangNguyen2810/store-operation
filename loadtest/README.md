# Load tests (k6)

Two scripts drive the HTTP API with [k6](https://k6.io). They need a running gateway and API to talk to; they do not start them.

| Script | What it does | Make target |
|---|---|---|
| `smoke.js` | 1 user, 1 pass over every endpoint, answers checked (status, content type, JSON shape, problem bodies for bad input). Every check must pass. | `make k6-smoke` |
| `load.js` | Ramps to 20 users for about 45 s. Each user opens the dashboard, then changes filters three times. p95 under 200 ms, under 1% errors. | `make k6-load` |
| `compare.sh` | Starts the API twice (without and with the Redis cache) and a gateway in front of the cached one, runs `load.js` against all three and prints them side by side. | `make k6-compare` |

```sh
brew install k6                      # once
make dev                             # or the API and gateway: make dev-api, make dev-gateway
make k6-smoke
make k6-load
BASE_URL=http://localhost:8081 make k6-load   # straight to the API, skipping the gateway
```

`CACHE=on|off` only adds a `cache` tag to every metric, so results from different setups can be told apart.
`BASE_URL` defaults to `http://localhost:8080`, the gateway. `load.js` sends more requests per second from one address than the gateway's default per-IP limit (50 per second), so run the gateway with `RATE_LIMIT_RPS=0` for it; `smoke.js` stays under the limit.
