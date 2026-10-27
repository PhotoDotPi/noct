package player

import "fmt"

type IPCError struct {
	Command string
	Message string
}

func (e *IPCError) Error() string {
	return fmt.Sprintf("mpv command %q failed: %s", e.Command, e.Message)
}
