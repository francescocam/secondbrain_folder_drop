# secondbrain-folder-drop

Go service that monitors a folder and pushes files into both a self-hosted Blinko instance and a self-hosted AFFiNE workspace.

## Behavior
- `.md` / `.markdown`: create one Blinko note from file content.
- `.md` / `.markdown`: create one AFFiNE document from file content.
- Other files: upload as Blinko attachment and create one linked note; upload the file to AFFiNE and create one companion document that links to it.
- Source file is deleted on success by default.
- Success requires both destinations to complete.
- On permanent failure, file is moved to `failed/` with a `*.error.json` sidecar.

## Commands
```bash
secondbrain-folder-drop version
secondbrain-folder-drop validate-config --config /path/config.yaml
secondbrain-folder-drop run --config /path/config.yaml
```

## Config
Start from `configs/config.example.yaml`.

AFFiNE requirements:
- the configured token must be a personal access token accepted as `Authorization: Bearer ...`
- the target workspace must expose AFFiNE's built-in MCP HTTP endpoint at `/api/workspaces/:workspaceId/mcp/`
- that endpoint must offer the `create_document` tool for the token/workspace pair

Environment overrides:
- `BFD_BASE_URL`
- `BFD_JWT_TOKEN`
- `BFD_AFFINE_BASE_URL`
- `BFD_AFFINE_AUTH_TOKEN`
- `BFD_AFFINE_WORKSPACE_ID`
- `BFD_INPUT_DIR`
- `BFD_FAILED_DIR`
- `BFD_RECURSIVE`
- `BFD_STABLE_FOR`
- `BFD_SCAN_EVERY`
- `BFD_WORKERS`
- `BFD_MAX_RETRIES`
- `BFD_RETRY_BASE_DELAY`
- `BFD_DELETE_ON_SUCCESS`
- `BFD_ARCHIVE_DIR`
- `BFD_QUEUE_SIZE`
- `BFD_HTTP_TIMEOUT`
- `BFD_LOG_LEVEL`
- `BFD_METRICS_ENABLED`
- `BFD_METRICS_LISTEN_ADDR`

## Linux build
```bash
go build -o secondbrain-folder-drop ./cmd/secondbrain-folder-drop
```

Cross build:
```bash
GOOS=linux GOARCH=amd64 go build -o dist/secondbrain-folder-drop-linux-amd64 ./cmd/secondbrain-folder-drop
GOOS=linux GOARCH=arm64 go build -o dist/secondbrain-folder-drop-linux-arm64 ./cmd/secondbrain-folder-drop
GOOS=windows GOARCH=amd64 go build -o dist/secondbrain-folder-drop-windows-amd64.exe ./cmd/secondbrain-folder-drop
```

Systemd unit is in `deploy/systemd/secondbrain-folder-drop.service`.
