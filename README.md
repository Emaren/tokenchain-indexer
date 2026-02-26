# tokenchain-indexer

`tokenchain-indexer` serves indexed and curated API data for `api.tokenchain.tokentap.ca`.

## Scope (initial)
- Chain status and metadata endpoints
- Health and version endpoints
- Foundation for reward, token, and merchant analytics read APIs

## Run
```bash
go run ./cmd/tokenchain-indexer
```

Environment variables:
- `LISTEN_ADDR` (default `:3321`)
- `CHAIN_ID` (default `tokenchain-testnet-1`)
- `NETWORK` (default `testnet`)

## Initial endpoints
- `GET /healthz`
- `GET /v1/network`
- `GET /v1/version`

## Next
- Add RPC/REST ingestion workers
- Add Postgres schema and migrations
- Add loyalty analytics endpoints
