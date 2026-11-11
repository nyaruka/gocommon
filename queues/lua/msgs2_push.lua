-- KEYS: [Queue, Channel, Ready, Waiting]
-- ARGV: [Owner, TPS, Batch, NowMicros]
local queueKey, channelKey, readyKey, waitingKey = KEYS[1], KEYS[2], KEYS[3], KEYS[4]
local owner, tps, batch = ARGV[1], tonumber(ARGV[2]), ARGV[3]

local now = tonumber(ARGV[4]) / 1000000

redis.call("ZADD", queueKey, now, batch)

-- the latest push sets the TPS for everything queued for this owner
redis.call("HSET", channelKey, "tps", tps)

if msgs2Schedule(channelKey, readyKey, waitingKey, owner, now) then
    return 1
end
return 0
