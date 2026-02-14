# Kitchen Sink Example

This example demonstrates how to use the Go sidecar client with tools, hooks, permissions, structured output, and typed message helpers.

## Running

```sh
SIDECAR_ADDR=localhost:50051 go run .
```

Or use the wrapper to start the sidecar automatically:

```sh
../../../../scripts/sidecar-run.sh -- go run .
```

macOS Keychain support (optional):

```sh
SIDECAR_KEYCHAIN_ENABLE=1 \
SIDECAR_KEYCHAIN_ACCOUNT="your-account" \
../../../../scripts/sidecar-run.sh -- go run .
```

The wrapper writes keychain JSON to `~/.claude/.credentials.json` if missing. To force overwrite:

```sh
SIDECAR_KEYCHAIN_WRITE_MODE=force
```

Keychain secrets may be JSON or base64-encoded JSON. If the entry is a raw token string, run `claude` login once to regenerate `~/.claude/.credentials.json`.

## Optional toggles

- `SIDECAR_INPUT_STREAM=1` uses the typed input stream builders (`UserBlocksEvent`) instead of `Query`.
- `SIDECAR_CONFIRM=1` enables interactive ask/confirm prompts for permissions.
- `SIDECAR_LIVE=1` disables test mode and runs against a live Claude session.
