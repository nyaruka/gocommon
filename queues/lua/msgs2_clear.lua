-- KEYS: [HighQueue, BulkQueue, Channel, Ready, Waiting]
-- ARGV: [Owner, NowMicros]
local highKey, bulkKey, channelKey, readyKey, waitingKey = KEYS[1], KEYS[2], KEYS[3], KEYS[4], KEYS[5]
local owner = ARGV[1]

local now = tonumber(ARGV[2]) / 1000000

redis.call("DEL", highKey, bulkKey)
redis.call("ZREM", readyKey, owner)

-- a paused owner stays parked so that its state, and so the pause, is kept until the pause ends, when a pop will find
-- it has nothing queued and let the state expire
local pausedUntil = tonumber(redis.call("HGET", channelKey, "paused_until")) or 0
if pausedUntil > now then
    redis.call("PERSIST", channelKey)
    redis.call("ZADD", waitingKey, pausedUntil, owner)
    return
end

redis.call("ZREM", waitingKey, owner)

-- the owner's state is kept for sends still in flight but with nothing queued it can expire
redis.call("EXPIRE", channelKey, 60)
