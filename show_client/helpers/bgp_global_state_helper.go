package helpers

func StateBoolToStr(state string) string {
	switch state {
	case "true":
		return "enabled"
	case "false":
		return "disabled"
	default:
		return state
	}
}
