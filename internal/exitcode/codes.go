// Package exitcode defines stable Reinstate process exit codes.
package exitcode

const (
	OK            = 0
	Runtime       = 1
	Usage         = 2
	Config        = 3
	AuthStorage   = 4
	Compatibility = 5
	Conflict      = 6
	Safety        = 7
)
