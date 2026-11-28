package main

import "os"

// remoteSSH reports whether the process is running inside an SSH session.
func remoteSSH() bool {
	for _, k := range []string{"SSH_CLIENT", "SSH_CONNECTION", "SSH_TTY"} {
		if os.Getenv(k) != "" {
			return true
		}
	}
	return false
}
