package flash

func createCacheKey(token string) string {
	return "flash:" + token
}
