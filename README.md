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

## Initial endpoints
- `GET /healthz`
- `GET /v1/network`
- `GET /v1/status`
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
