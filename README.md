# russel-demo-notes

A tiny in-memory JSON notes API (Go stdlib, no dependencies) for demoing [Russel](https://github.com/daschinmoy21/russel) deploys from a git URL.

## Deploy with Russel

```bash
russel deploy https://github.com/daschinmoy21/russel-demo-notes.git
```

`Russelfile.toml` sets the service name (`notes-api`), port 3000, 128mb memory, and the `container` runtime. `flake.nix` builds the binary.

## Routes

| Method | Path | Description |
|---|---|---|
| GET | `/` | Service info (greeting, hostname, uptime) |
| GET | `/health` | `ok` |
| GET | `/notes` | List notes |
| POST | `/notes` | Create: `{"text": "..."}` |
| GET | `/notes/{id}` | Get one note |
| DELETE | `/notes/{id}` | Delete a note |

```bash
curl -X POST http://HOST:PORT/notes -d '{"text":"hello"}'
curl http://HOST:PORT/notes
```

Notes are held in memory and reset on restart. Set `GREETING` in `[service.env]` to change the message on `/`.

## Run locally

```bash
nix build && PORT=3000 ./result/bin/notes-api
# or: go run .
```
