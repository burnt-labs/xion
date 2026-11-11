# AGENTS.md

## Temporary directories

- When creating a temporary application home for `app.NewWasmApp` in `cmd/xiond/root.go`, defer `os.RemoveAll` in the caller that uses the directory (such as `NewRootCmd`), after successful creation; do not defer removal inside a helper that returns the path. Keep the directory alive until application initialization and command construction finish.
