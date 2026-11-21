package theme

func parseValue(s string) (Value, bool) {
	switch Value(s) {
	case Light, Dark, System:
		return Value(s), true
	default:
		return "", false
	}
}
