package client

const (
	OPTION_ECHO                        byte = 1
	OPTION_TERMINAL_TYPE               byte = 24
	OPTION_NEGOTIATE_ABOUT_WINDOW_SIZE byte = 31
	OPTION_TERMINAL_SPEED              byte = 32
	OPTION_X_DISPLAY_LOCATION          byte = 35
	OPTION_NEW_ENVIRONMENT_OPTION      byte = 39
)

func IsSupportOption(option byte) bool {
	for _, v := range []byte{OPTION_ECHO, OPTION_NEGOTIATE_ABOUT_WINDOW_SIZE} {
		if option == v {
			return true
		}
	}
	return false
}
