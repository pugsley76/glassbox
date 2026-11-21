# Glassbox Trace Showcase

This is an interactive, web-based showcase for "glassbox" traces. It allows users to visualize transaction execution traces with a sleek, modern interface.

## Features

- **Trace Visualization**: Step-through contract execution with host state and memory views.
- **Premium Design**: Dark mode aesthetic with glassmorphism and smooth animations.
- **Interactive**: Try it out with sample hashes or any input to see the tracer in action.

## Prerequisites

- **Node.js**: A version compatible with Next.js 16 and React 19 (Node.js 20.9 or newer is recommended).
- **npm**: Installed alongside Node.js.

## Getting Started

All commands below are run from the `docs/showcase` directory.

1. Install dependencies:
   ```bash
   npm install
   ```
   Expected output: npm resolves and installs the packages listed in `package.json` and exits with status `0`. If a lockfile is present, `npm ci` may be used instead for a reproducible install.

2. Run the development server:
   ```bash
   npm run dev
   ```
   Expected output: Next.js starts and prints a local URL such as `http://localhost:3000`. The default port is `3000`.

3. Open [http://localhost:3000](http://localhost:3000) in your browser.

## Build and Lint

- Build the production bundle:
  ```bash
  npm run build
  ```
  Expected output: Next.js compiles the application and reports a successful build, exiting with status `0`.

- Run the linter:
  ```bash
  npm run lint
  ```
  Expected output: the configured linter runs over the project and exits with status `0` when there are no findings.

## Sample Hash

Use this hash to see a full execution trace:
`sample-tx-hash-12345`

## Troubleshooting

- **Running the dev server before installing dependencies**: If you run `npm run dev` before `npm install`, the command fails because the required packages (including Next.js) are not present. Install dependencies first, then re-run `npm run dev`.

  Successful path:
  ```bash
  npm install
  npm run dev
  ```
  The dev server starts and prints a local URL.

  Failure path:
  ```bash
  npm run dev
  ```
  The command exits with a non-zero status and an error indicating that the `next` binary or required modules could not be found. Run `npm install` and try again.