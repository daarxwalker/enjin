package cache

import "github.com/redis/go-redis/v9"

var incrementScript = redis.NewScript(`
if tonumber(ARGV[1]) < 0 and redis.call('EXISTS', KEYS[1]) == 0 then
	return 0
end
local value = redis.call('INCRBY', KEYS[1], ARGV[1])
if tonumber(ARGV[2]) > 0 and redis.call('PTTL', KEYS[1]) < 0 then
	redis.call('PEXPIRE', KEYS[1], ARGV[2])
end
return value
`)
