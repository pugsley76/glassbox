// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package session

import "syscall"

func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}
