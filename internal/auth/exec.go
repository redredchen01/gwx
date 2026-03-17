package auth

import "os/exec"

// commandExec wraps exec.Command for testability.
var commandExec = exec.Command
