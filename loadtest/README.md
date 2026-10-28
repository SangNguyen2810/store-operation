# Load tests (k6)

Two scripts drive the HTTP API with [k6](https://k6.io). They need an API to talk to; they do not start one.

| Script | What it does | Make target |
|---|---|---|
| `smoke.js` | 1 user, 1 pass over every endpoint, answers checked (status, content type, JSON shape, problem bodies for bad input). Every check must pass. | `make k6-smoke` |
| `load.js` | Ramps to 20 users for about 45 s. Each user opens the dashboard, then changes filters three times. p95 under 200 ms, under 1% errors. | `make k6-load` |
| `compare.sh` | Starts the API twice, without and with the Redis cache, runs `load.js` against both and prints the two side by side. | `make k6-compare` |

```sh
brew install k6                      # once
make dev-api                         # or any running API
make k6-smoke
make k6-load
BASE_URL=http://localhost:8081 make k6-load   # another address, for example the gateway later
```

`CACHE=on|off` only adds a `cache` tag to every metric, so results from different setups can be told apart.
`BASE_URL` defaults to `http://localhost:8080`.
