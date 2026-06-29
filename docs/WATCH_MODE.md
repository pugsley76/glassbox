# Watch Mode for Debug Sessions

Glassbox provides a watch mode to rapidly iterate on your smart contracts by automatically rerunning the debug session whenever relevant files change.

## Usage

Enable the feature by using the `--watch-files` flag on the debug command:

```bash
glassbox debug <transaction-hash> --watch-files
```

If you have configured a local source directory via `--contract-source`, the watcher will automatically track it for changes:

```bash
glassbox debug <transaction-hash> --contract-source ./contracts/my_contract --watch-files
```

## How It Works

1. **Directories Watched:** By default, it will watch the current working directory (`.`), tracking configuration changes. If `--contract-source` is provided, that directory will also be recursively monitored.
2. **Debouncing:** File system events are debounced by 500ms. This prevents multiple rapid changes (e.g., from an IDE saving all files simultaneously or running a build tool) from triggering multiple redundant debug session reruns.
3. **Execution Loop:** When a change is detected and the debounce period elapses, the watcher transparently tears down the previous run, re-evaluates the inputs, and reruns the entire debug simulation, retaining all other command line flags and context (such as `--network`, `--json`, or OpenTelemetry settings).

## Distinguishing from Network Watching

Glassbox has two distinct forms of watching:

- `--watch-files`: Monitors local source and configuration file changes (this document).
- `--watch`: Polls the Stellar network waiting for a specific transaction hash to finalize on-chain *before* attempting the initial debug run. (Often used right after submission).

You can combine them, though they serve different phases of the lifecycle.
