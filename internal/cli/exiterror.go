package cli

import "fmt"

// ExitError carries a process exit code through cobra's RunE.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string { return fmt.Sprintf("exit code %d", e.Code) }
