// Copyright (c) glassbox Authors.
// SPDX-License-Identifier: Apache-2.0
//
// This declaration narrows the global `process` for source files that run in
// environments where the full Node.js process object may not be available.
// It is intentionally minimal — only the subset used by the commands package.
//
// In test files, @types/node provides the full NodeJS.Process type and takes
// precedence, so spying on process.exit and process.stdout works correctly.
export {};
declare global {
  // Only augment when @types/node is NOT present (i.e. non-test environments).
  // When @types/node is available its NodeJS.Process declaration wins.
}
