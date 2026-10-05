# User guide

## Current status

MistHelper-Go is under active development, not a complete replacement for
Python MistHelper. The API, menu, output, SSH, and web packages are wired and
tested.

The current executable registers 89 menu entries. **Option 26, List Org
Inventory, is implemented. The other 88 entries are placeholders.** A
placeholder prints that the operation is not yet implemented; it does not
perform the named operation. Menu numbers in the current scaffold do not
match the complete Python menu.

CSV and SQLite writers are implemented. ArangoDB and Redis are goals, not
available output backends in this build. The HTTP server has JSON status and
health routes, not a graphical web dashboard. See the [actual screens](interface.md).

## Container start

Production deployment is container-only. Examples use Podman.
Direct executable use is for local development only.

```bash
mkdir -p data
cp .env.example .env
# Edit .env: set MIST_API_TOKEN, MIST_ORG_ID, and a strong SSH_PASSWORD.
podman pull ghcr.io/jmorrison-juniper/misthelper-go:latest
podman run -d --name misthelper-go \
  -p 127.0.0.1:2200:2200 -p 127.0.0.1:8055:8055 \
  -v "${PWD}/data:/app/data:rw" \
  -v "${PWD}/.env:/app/.env:ro" \
  ghcr.io/jmorrison-juniper/misthelper-go:latest
```

These port mappings restrict access to the local host. Remote access requires
an approved network configuration. Never expose the default SSH password.
Do not commit `.env`, tokens, host keys, or exported network data.

## Access and output

```bash
ssh -p 2200 misthelper@127.0.0.1
curl --fail http://127.0.0.1:8055/
curl --fail http://127.0.0.1:8055/health
```

The SSH connection opens the menu, not a system shell. Select a number shown
in the menu. Close the SSH session to leave the interactive loop.
`--menu 0` exits a direct executable invocation; the current interactive menu
does not register a quit entry.

| Interface or output | Location |
| --- | --- |
| SSH menu | Container port 2200 by default |
| JSON status / health | Container port 8055; `/` and `/health` |
| CSV exports | `data/{endpoint}_{timestamp}.csv` |
| SQLite | `data/mist_data.db` |
| SSH host key | `data/ssh_host_rsa_key` |
| SSH session directories | `data/sessions/session_<id>/` |

The default output is CSV. Set `OUTPUT_FORMAT=sqlite` in `.env` or pass
`--format sqlite` to the executable to use SQLite. The current configuration
reader uses `OUTPUT_FORMAT`, not the planned `MIST_OUTPUT_BACKEND` setting in
the example file.

Option 26 fetches organization inventory and writes it with the
`getOrgInventory` endpoint strategy. SQLite uses natural or composite business
keys where defined, with replacement on matching keys. An empty result creates
no CSV file. Do not run option 26 during an offline screen capture.

## Support and license

Maintainer: [@jmorrison-juniper](https://github.com/jmorrison-juniper).
Report reproducible problems in
[GitHub issues](https://github.com/jmorrison-juniper/MistHelper-Go/issues).
Include the version, environment, menu number, and redacted logs.
Read the [security policy](../SECURITY.md) for vulnerability reports and the
[code of conduct](../CODE_OF_CONDUCT.md) before contributing.

Licensed under [Apache 2.0](../LICENSE).
