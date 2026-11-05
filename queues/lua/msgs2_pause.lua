-- KEYS: [Channel, Ready, Waiting]
-- ARGV: [Owner, UntilMicros, NowMicros]
local channelKey, readyKey, waitingKey = KEYS[1], KEYS[2], KEYS[3]
local owner = ARGV[1]

local untilTime = tonumber(ARGV[2]) / 1000000
local now = tonumber(ARGV[3]) / 1000000

-- a pause can only be extended, so that a shorter backoff reported later doesn't cut short a longer one
local pausedUntil = tonumber(redis.call("HGET", channelKey, "paused_until")) or 0
if untilTime <= now or untilTime <= pausedUntil then
    return 0
end

redis.call("HSET", channelKey, "paused_until", untilTime)

-- park the owner even if it has nothing queued so that its state, and so the pause, is kept until the pause ends,
-- when a pop will find it has nothing queued and let the state expire
redis.call("PERSIST", channelKey)
redis.call("ZREM", readyKey, owner)
redis.call("ZADD", waitingKey, "GT", untilTime, owner)
return 1
