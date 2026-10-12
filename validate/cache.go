package validate

func createCacheKey(token string) string {
	return "validation" + ":" + token
}
