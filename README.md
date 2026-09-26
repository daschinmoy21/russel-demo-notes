# russel-demo-notes

A small notes web app and JSON API (Go stdlib, no dependencies) for demoing [Russel](https://github.com/daschinmoy21/russel) deploys from a git URL.

## Deploy with Russel

```bash
russel deploy https://github.com/daschinmoy21/russel-demo-notes.git
```

Then open http://127.0.0.1:3100 (or http://notes-api.russel.local with Traefik routing).

`Russelfile.toml` fully describes the deployment, with every field written out:

| Field | Value | Why |
|---|---|---|
| `service.type` | `container` | Rootless Podman |
| `service.port` | `3000` | App listens here; Russel injects it as `PORT` |
| `service.memory` | `128mb` | Plenty for a static Go binary |
| `service.restart` | `unless-stopped` | Podman restarts it if it crashes |
| `[ingress].host` | `notes-api.russel.local` | Traefik route |
| `[ingress].port` | `3100` | Stable host port across updates |
| `[[volumes]]` `data` → `/data` | `rw`, `keep` | Notes persist across restarts, updates, and destroy |
| `service.env.NOTES_FILE` | `/data/notes.json` | Where the app stores notes |

`flake.nix` builds the binary.

## Routes

| Method | Path | Description |
|---|---|---|
| GET | `/` | Web UI for adding and deleting notes |
| GET | `/api/info` | Service info as JSON (greeting, hostname, storage, uptime) |
| GET | `/health` | `ok` |
| GET | `/notes` | List notes |
| POST | `/notes` | Create: `{"text": "..."}` |
| GET | `/notes/{id}` | Get one note |
| DELETE | `/notes/{id}` | Delete a note |

```bash
curl -X POST http://127.0.0.1:3100/notes -d '{"text":"hello"}'
curl http://127.0.0.1:3100/notes
```

## Configuration

| Env | Default | Meaning |
|---|---|---|
| `NOTES_FILE` | unset (memory only) | JSON file to persist notes to |
| `GREETING` | `hello from russel` | Message shown in the UI and at `/api/info` |

## Run locally

```bash
nix build && PORT=3000 NOTES_FILE=./notes.json ./result/bin/notes-api
# or: go run .
```
