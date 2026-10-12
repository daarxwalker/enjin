package csrf

import "github.com/redis/go-redis/v9"

var addScript = redis.NewScript(`
local limit = math.max(tonumber(ARGV[4]), 1)
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', ARGV[3])
redis.call('ZADD', KEYS[1], ARGV[2], ARGV[1])
redis.call('ZREMRANGEBYRANK', KEYS[1], 0, -limit - 1)
local latest = redis.call('ZRANGE', KEYS[1], -1, -1, 'WITHSCORES')
redis.call('PEXPIREAT', KEYS[1], latest[2])
return 1
`)

var takeScript = redis.NewScript(`
local expiresAt = redis.call('ZSCORE', KEYS[1], ARGV[1])
if not expiresAt then
	return 0
end
redis.call('ZREM', KEYS[1], ARGV[1])
if tonumber(expiresAt) <= tonumber(ARGV[2]) then
	return 0
end
return 1
`)
