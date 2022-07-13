package command

const (
	SE byte = 240 + iota //	End of subnegotiation parameters.
	NOP
	DATA_MARK
	BREAK
	INTERRUPT_PROCESS
	ABORT_OUTPUT
	ARE_YOU_THERE
	ERASE_CHARACTER
	ERASE_LINE
	GO_AHEAD
	SB // Indicates that what follows is subnegotiation of the indicated option.

	WILL
	WONT
	DO
	DONT

	IAC
)

func IsNeedOption(cmd byte) bool {
	return WILL <= cmd && cmd < IAC
}
