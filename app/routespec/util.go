package routespec

func firstDescription(description []string) string {
	if len(description) == 0 {
		return ""
	}
	return description[0]
}
