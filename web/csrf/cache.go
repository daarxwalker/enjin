package csrf

func createCacheKey(token string) string {
	return "csrf" + ":" + token
}
