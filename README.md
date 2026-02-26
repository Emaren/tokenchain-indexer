# tokenchain-indexer

`tokenchain-indexer` serves indexed and curated API data for `api.tokenchain.tokentap.ca`.

## Scope (initial)
- Chain status and metadata endpoints
- Health and version endpoints
- Foundation for reward, token, and merchant analytics read APIs
- Testnet faucet API (`tokenchain-faucet`) for noob onboarding

## Run
```bash
go run ./cmd/tokenchain-indexer
```

Run faucet:
```bash
go run ./cmd/tokenchain-faucet
```

Environment variables:
- `LISTEN_ADDR` (default `:3321`)
- `CHAIN_ID` (default `tokenchain-testnet-1`)
- `NETWORK` (default `testnet`)
- `RPC_ADDR` (default `http://127.0.0.1:26657`)
- `REST_ADDR` (default `http://127.0.0.1:1317`)
- `ADMIN_API_TOKEN` (default empty/disabled)
- `ADMIN_FROM_KEY` (default `founder`)
- `CHAIN_HOME` (default `/var/lib/tokenchain-testnet`)
- `KEYRING_BACKEND` (default `test`)
- `TX_FEES` (default `5000utoken`)
- `TX_GAS` (default `200000`)
- `TOKENCHAIND_BIN` (default `/usr/local/bin/tokenchaind`)

Faucet environment variables:
- `FAUCET_ENABLED` (default `true`)
- `STATE_FILE` (default `/var/lib/tokenchain-testnet/faucet-state.json`)
- `NODE_RPC` (default `http://127.0.0.1:26657`)
- `CHAIN_HOME`, `KEYRING_BACKEND`, `FROM_KEY`
- `DENOM`, `DISPENSE_AMOUNT`, `TX_FEES`
- `ADDRESS_COOLDOWN_MINUTES`, `IP_COOLDOWN_MINUTES`, `MAX_REQUESTS_PER_DAY`

## Initial endpoints
- `GET /healthz`
- `GET /v1/network`
- `GET /v1/status`
- `GET /v1/loyalty/merchant-routing?limit=25&verified_only=true`
- `GET /v1/loyalty/merchant-allocations?limit=25&date=YYYY-MM-DD&denom=factory/...`
- `POST /v1/admin/loyalty/merchant-routing` (Bearer token auth; disabled unless `ADMIN_API_TOKEN` set)
- `POST /v1/admin/loyalty/merchant-allocation` (Bearer token auth; disabled unless `ADMIN_API_TOKEN` set)
- `GET /v1/endpoints`
- `GET /v1/version`

Faucet endpoints:
- `GET /healthz`
- `GET /v1/faucet/policy`
- `POST /v1/faucet/request` with body `{ "address": "tokenchain1..." }`

## Next
- Add RPC/REST ingestion workers
- Add Postgres schema and migrations
- Add loyalty analytics endpoints
