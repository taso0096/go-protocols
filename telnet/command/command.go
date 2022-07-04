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
)

func IsNeedOption(cmd byte) bool {
	return WILL <= cmd && cmd < IAC
}
