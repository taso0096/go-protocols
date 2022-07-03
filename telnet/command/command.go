package command

const (
	SE byte = 240 + iota
	NOP
	DATA_MARK
	BREAK
	INTERRUPT_PROCESS
	ABORT_OUTPUT
	ARE_YOU_THERE
	ERASE_CHARACTER
	ERASE_LINE
	GO_AHEAD
	SB

	WILL
	WONT
	DO
	DONT

	IAC

	OPTION_ECHO                   byte = 1
	OPTION_TERMINAL_TYPE          byte = 24
	OPTION_TERMINAL_SPEED         byte = 32
	OPTION_X_DISPLAY_LOCATION     byte = 35
	OPTION_NEW_ENVIRONMENT_OPTION byte = 39
)

func IsNeedOption(cmd byte) bool {
	return cmd >= WILL
}
