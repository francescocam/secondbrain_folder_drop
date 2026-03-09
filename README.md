# secondbrain-folder-drop

Go service that monitors a folder and pushes files into a self-hosted Blinko instance, a self-hosted AFFiNE workspace, or both.

## Behavior
- `.md` / `.markdown`: create one Blinko note from file content when Blinko is enabled.
- `.md` / `.markdown`: create one AFFiNE document from file content when AFFiNE is enabled.
- Other files: upload as a Blinko attachment and linked note when Blinko is enabled; upload the file to AFFiNE and create one companion document when AFFiNE is enabled.
- Source file is deleted on success by default.
- Success requires all selected destinations to complete.
- On permanent failure, file is moved to `failed/` with a `*.error.json` sidecar.

## Commands
```bash
secondbrain-folder-drop version
secondbrain-folder-drop validate-config --config /path/config.yaml
secondbrain-folder-drop run --config /path/config.yaml
secondbrain-folder-drop run --config /path/config.yaml --target blinko
secondbrain-folder-drop run --config /path/config.yaml --target affine
```

`--target` accepts `both` (default), `blinko`, or `affine`.

## Config
Start from `configs/config.example.yaml`.
Detailed field-by-field setup instructions are in `docs/config.md`.

When running with `--target blinko`, only the `blinko` section is required.
When running with `--target affine`, only the `affine` section is required.

AFFiNE requirements:
- the configured token must be a personal access token accepted as `Authorization: Bearer ...`
- `node` must be available on `PATH` because AFFiNE document writes are performed through an embedded helper that uses AFFiNE's Socket.IO and Yjs update flow

AFFiNE implementation credit:
- The AFFiNE document-write approach in this project was derived from `DAWNCR0W/affine-mcp-server`: https://github.com/DAWNCR0W/affine-mcp-server

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

Runtime requirement:
- Blinko-only mode requires the Go binary.
- AFFiNE mode requires `node` on `PATH` in addition to the Go binary.

Cross build:
```bash
GOOS=linux GOARCH=amd64 go build -o dist/secondbrain-folder-drop-linux-amd64 ./cmd/secondbrain-folder-drop
GOOS=linux GOARCH=arm64 go build -o dist/secondbrain-folder-drop-linux-arm64 ./cmd/secondbrain-folder-drop
GOOS=windows GOARCH=amd64 go build -o dist/secondbrain-folder-drop-windows-amd64.exe ./cmd/secondbrain-folder-drop
```

Systemd unit is in `deploy/systemd/secondbrain-folder-drop.service`.
