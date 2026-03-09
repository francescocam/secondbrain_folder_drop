# Configuration Guide

This document explains how to fill `config.yaml` for `secondbrain-folder-drop`.

Start from [`configs/config.example.yaml`](/home/fcamisa/code/secondbrain_folder_drop/configs/config.example.yaml) and copy it to your real config path.

## What This Service Needs

The service watches one folder and uploads new files to:

- Blinko
- AFFiNE
- or both, if you run with `--target both`

The CLI target affects which config sections are required:

- `--target both`: both `blinko` and `affine` must be filled
- `--target blinko`: only `blinko` must be filled
- `--target affine`: only `affine` must be filled

Additional runtime requirement:

- If you use `--target affine` or `--target both`, `node` must be installed and available on `PATH`.
- Blinko-only mode does not require Node.js.

## Minimal Examples

Blinko only:

```yaml
blinko:
  base_url: "https://blinko.example.com"
  jwt_token: "eyJ..."

affine:
  base_url: ""
  auth_token: ""
  workspace_id: ""

watch:
  input_dir: "/srv/folder-drop/inbox"
  failed_dir: "/srv/folder-drop/failed"
  recursive: false
  stable_for: 3s
  scan_every: 30s

processing:
  workers: 2
  max_retries: 5
  retry_base_delay: 2s
  delete_on_success: true
  archive_dir: ""
  queue_size: 512

http:
  timeout: 120s

logging:
  level: info

metrics:
  enabled: false
  listen_addr: 127.0.0.1:9095
```

AFFiNE only:

```yaml
blinko:
  base_url: ""
  jwt_token: ""

affine:
  base_url: "https://affine.example.com"
  auth_token: "your-mcp-personal-access-token"
  workspace_id: "your-workspace-id"

watch:
  input_dir: "/srv/folder-drop/inbox"
  failed_dir: "/srv/folder-drop/failed"
  recursive: false
  stable_for: 3s
  scan_every: 30s

processing:
  workers: 2
  max_retries: 5
  retry_base_delay: 2s
  delete_on_success: true
  archive_dir: ""
  queue_size: 512

http:
  timeout: 120s

logging:
  level: info

metrics:
  enabled: false
  listen_addr: 127.0.0.1:9095
```

## Field-By-Field Reference

### `blinko.base_url`

The root URL of your self-hosted Blinko instance.

Examples:

- `http://127.0.0.1:1111`
- `http://192.168.1.20:1111`
- `https://blinko.example.com`

Rules:

- Use the server root, not an API path
- Do not append `/api/v1/note/upsert`
- Do not append `/api/file/upload`
- Trailing `/` is fine, but unnecessary

This service builds these Blinko API calls from `base_url`:

- `POST /api/v1/note/upsert`
- `POST /api/file/upload`

### `blinko.jwt_token`

The Bearer token used to authenticate with Blinko.

Rules:

- Paste only the token value
- Do not include the literal `Bearer `
- It is expected to be a JWT-like token, typically starting with `eyJ`

### `affine.base_url`

The root URL of your self-hosted AFFiNE instance.

Examples:

- `http://127.0.0.1:3000`
- `http://192.168.1.30:3000`
- `https://affine.example.com`

Rules:

- Use the server root, not `/graphql`
- Use the server root, not `/api/workspaces/.../mcp`
- Trailing `/` is fine, but unnecessary

This service builds these AFFiNE calls from `base_url`:

- `POST /graphql`
- `POST /api/workspaces/<workspace_id>/mcp/`

### `affine.auth_token`

The personal access token used for AFFiNE API and Socket.IO access.

Rules:

- Paste only the raw token
- Do not include `Bearer `
- The token must be authorized for the target workspace

### `affine.workspace_id`

The AFFiNE workspace ID that should receive imported documents and files.

This is not the workspace display name. It is the internal workspace identifier used in URLs and API paths.

### `watch.input_dir`

The directory the service monitors for incoming files.

Examples:

- `/var/lib/secondbrain-folder-drop/inbox`
- `/srv/folder-drop/inbox`
- `C:\secondbrain\inbox` on Windows

Behavior:

- The service creates the directory if it does not already exist
- New or changed files are enqueued after they remain stable for `watch.stable_for`

### `watch.failed_dir`

Where permanently failed files are moved.

If empty, the service uses:

```yaml
watch:
  failed_dir: "<input_dir>/failed"
```

Behavior:

- The failed file is moved here
- A sidecar `*.error.json` file is written next to it

### `watch.recursive`

Whether subdirectories under `watch.input_dir` should also be scanned.

- `false`: only the top-level input directory
- `true`: include nested directories

### `watch.stable_for`

How long a file must stop changing before it is considered ready.

Examples:

- `3s`
- `10s`
- `1m`

Use a higher value if another process writes large files slowly.

### `watch.scan_every`

How often the directory is scanned.

Examples:

- `10s`
- `30s`
- `1m`

### `processing.workers`

Number of concurrent upload workers.

- Start with `2`
- Increase if uploads are slow and your servers can handle more concurrency

### `processing.max_retries`

How many times a failed upload is retried before the file is quarantined.

Recommended starting value:

- `5`

### `processing.retry_base_delay`

Base retry delay.

Examples:

- `2s`
- `5s`

### `processing.delete_on_success`

What happens to the source file after successful upload.

- `true`: delete the source file
- `false`: move it to `processing.archive_dir`

If you set this to `false`, you must also set `processing.archive_dir`.

### `processing.archive_dir`

Where successful files go when `delete_on_success: false`.

Example:

```yaml
processing:
  delete_on_success: false
  archive_dir: "/srv/folder-drop/archive"
```

### `processing.queue_size`

Maximum in-memory queue size for discovered files.

Recommended starting value:

- `512`

### `http.timeout`

HTTP timeout for all outbound requests.

Recommended starting value:

- `120s`

Increase this if large uploads to AFFiNE or Blinko time out.

### `logging.level`

Current config supports the field and defaults to `info`.

Use:

- `info`

### `metrics.enabled`

Whether the Prometheus-style metrics endpoint is enabled.

- `true`
- `false`

### `metrics.listen_addr`

Bind address for the metrics HTTP server.

Examples:

- `127.0.0.1:9095`
- `0.0.0.0:9095`

Use `127.0.0.1` unless you explicitly want remote scraping.

## How To Get Blinko Values From A Self-Hosted Instance

### Blinko `base_url`

Use the URL you open in the browser for your Blinko server.

Common self-hosted examples:

- local Docker host: `http://127.0.0.1:1111`
- LAN host: `http://YOUR_SERVER_IP:1111`
- reverse proxy: `https://blinko.example.com`

Blinko’s official access-token documentation uses `http://127.0.0.1:1111` in its API examples and points users to the instance-local API docs at `/api-doc`. That is a good confirmation that `base_url` should be the server root, not a deeper API endpoint.

Practical check:

1. Open `https://your-blinko-host/api-doc` in a browser.
2. If the API docs load, `https://your-blinko-host` is the right `base_url`.

### Blinko `jwt_token`

Blinko’s official docs describe the API token as a JWT-style token and show it being sent as `Authorization: Bearer ...`.

The Blinko docs specifically mention:

- the token is shown in settings
- the token format starts with `eyJ`
- you can test it against `POST /api/v1/note/upsert`

For self-hosted use, the safest workflow is:

1. Sign in to your Blinko instance in the browser.
2. Open the settings area where your API token or Blinko token is shown.
3. Copy the full token string.
4. Paste it into `blinko.jwt_token` without `Bearer `.
5. Verify it by calling your instance’s `/api-doc` page or by running `secondbrain-folder-drop validate-config --config ... --target blinko`.

If you cannot find the token in the UI, check the instance-local API docs at `/api-doc` and the Blinko settings pages for the access-token section.

## How To Get AFFiNE Values From A Self-Hosted Instance

### AFFiNE `base_url`

Use the root URL where your AFFiNE server is reachable.

Common self-hosted examples:

- local Docker host: `http://127.0.0.1:3000`
- LAN host: `http://YOUR_SERVER_IP:3000`
- reverse proxy: `https://affine.example.com`

AFFiNE’s official Docker deployment guide uses port `3000` by default and shows the local URL as `http://localhost:3000`.

Practical check:

1. Open your AFFiNE instance in the browser.
2. Confirm the root URL loads normally.
3. Use that root URL as `affine.base_url`.

### AFFiNE `auth_token` and `workspace_id`

For this project, the best source is AFFiNE’s workspace MCP settings UI.

Current upstream AFFiNE source includes a workspace setting panel called MCP Server. That panel:

- creates a personal access token named `mcp`
- shows a generated JSON config block
- includes the MCP server URL
- includes the `Authorization: Bearer ...` header

The generated JSON has this shape:

```json
{
  "mcpServers": {
    "affine_workspace_<workspace_id>": {
      "type": "streamable-http",
      "url": "https://affine.example.com/api/workspaces/<workspace_id>/mcp",
      "headers": {
        "Authorization": "Bearer <token>"
      }
    }
  }
}
```

Use that generated JSON to fill your config:

- `affine.base_url`: everything before `/api/workspaces/...`
- `affine.workspace_id`: the path segment after `/api/workspaces/`
- `affine.auth_token`: the token value after `Bearer `

Recommended workflow in AFFiNE:

1. Open the target workspace in AFFiNE.
2. Open workspace settings.
3. Open the integrations section.
4. Open `MCP Server`.
5. Create a new personal access token if none exists.
6. Copy the generated JSON immediately.
7. Extract `base_url`, `workspace_id`, and token from that JSON.

Important detail from the upstream UI code:

- the full token is only copyable immediately after creation
- once the token is only stored in redacted form, the “Copy json” action is disabled
- if you lose the token, you may need to delete and recreate it

### AFFiNE prerequisites for this service

This service expects all of the following to work:

- `POST /graphql` for blob upload
- WebSocket access to the AFFiNE Socket.IO endpoint at `/socket.io/`
- `node` available on `PATH` because the embedded AFFiNE writer helper runs on Node.js

If your AFFiNE deployment blocks Socket.IO traffic or `node` is not installed, config validation may pass while runtime uploads still fail.

## Recommended First-Time Setup

1. Copy [`configs/config.example.yaml`](/home/fcamisa/code/secondbrain_folder_drop/configs/config.example.yaml) to your real config file.
2. Fill `watch.input_dir`.
3. Fill either the `blinko` section, the `affine` section, or both.
4. Keep `processing.delete_on_success: true` until you trust the pipeline.
5. Run:

```bash
secondbrain-folder-drop validate-config --config /path/to/config.yaml --target blinko
secondbrain-folder-drop validate-config --config /path/to/config.yaml --target affine
secondbrain-folder-drop validate-config --config /path/to/config.yaml --target both
```

6. Start the service with the target you want:

```bash
secondbrain-folder-drop run --config /path/to/config.yaml --target blinko
secondbrain-folder-drop run --config /path/to/config.yaml --target affine
secondbrain-folder-drop run --config /path/to/config.yaml --target both
```

## Environment Variable Overrides

Every config value that matters operationally can also be overridden by environment variables:

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

## Troubleshooting

### Blinko returns HTML instead of JSON

Your `blinko.base_url` is probably wrong.

Use the server root, for example:

- correct: `https://blinko.example.com`
- wrong: `https://blinko.example.com/api/v1/note/upsert`

### AFFiNE document creation fails

The current writer uses AFFiNE's GraphQL blob upload plus Socket.IO/Yjs document updates.

Check:

- the token belongs to the correct account
- the workspace ID matches the workspace you opened in AFFiNE
- the AFFiNE instance allows Socket.IO connections on `/socket.io/`
- `node` is installed and reachable as `node`

### Files are disappearing and you want to keep originals

Set:

```yaml
processing:
  delete_on_success: false
  archive_dir: "/srv/folder-drop/archive"
```

### Large files fail intermittently

Try increasing:

```yaml
http:
  timeout: 300s

watch:
  stable_for: 10s
```

## Sources

Primary sources used for this document:

- Blinko access-token docs: https://raw.githubusercontent.com/blinkospace/blinko-docs/main/en/settings/access-token.mdx
- Blinko API reference intro: https://raw.githubusercontent.com/blinkospace/blinko-docs/main/api-reference/introduction.mdx
- AFFiNE Docker deployment README: https://raw.githubusercontent.com/toeverything/docker/master/README.md
- AFFiNE write-path reference implementation: https://github.com/DAWNCR0W/affine-mcp-server
- AFFiNE MCP workspace settings UI source: https://raw.githubusercontent.com/toeverything/AFFiNE/canary/packages/frontend/core/src/desktop/dialogs/setting/workspace-setting/integration/mcp-server/setting-panel.tsx
- AFFiNE MCP controller source: https://raw.githubusercontent.com/toeverything/AFFiNE/canary/packages/backend/server/src/plugins/copilot/mcp/controller.ts

Inference note:

- The AFFiNE instructions above for locating `auth_token` and `workspace_id` are based on current upstream AFFiNE source code because the public docs pages for this workflow were not directly accessible from automated browsing during this task.
