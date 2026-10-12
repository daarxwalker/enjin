package session

func createCacheKey(token string) string {
	return "session:" + token
}
