# HackForger development setup

HackForger is a Forgejo fork. Keep local and instance-specific configuration outside the public Git tree.

## Build

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend
make frontend
```

Copy `custom/conf/app.example.ini` to an ignored local configuration path and adjust it for your development database. Do not commit real domains, accounts, credentials, or machine paths.

Start the application with a local custom path:

```bash
./gitea web --custom-path /tmp/hackforger-dev-custom
```

Use `http://localhost:3000` unless your local configuration selects another port.

## Test

```bash
go test ./models/hackforger/...
go test ./services/hackforger/...
bash scripts/check-public-repository-boundary.sh
```

Use neutral fixtures created by the test itself. Store runtime screenshots and reports as CI artifacts, not tracked files.

## Private instance configuration

Branded content, deployment targets, production runbooks, and environment-specific smoke tests belong to the corresponding private business repository. Public deployment tooling accepts explicit configuration and must fail closed when required configuration is absent.
