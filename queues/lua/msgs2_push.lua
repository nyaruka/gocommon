-- KEYS: [Queue, Channel, Ready, Waiting]
-- ARGV: [Owner, TPS, Batch, NowMicros]
local queueKey, channelKey, readyKey, waitingKey = KEYS[1], KEYS[2], KEYS[3], KEYS[4]
local owner, tps, batch = ARGV[1], tonumber(ARGV[2]), ARGV[3]

local now = tonumber(ARGV[4]) / 1000000
local nowSec = math.floor(now)

redis.call("ZADD", queueKey, now, batch)

-- the latest push sets the TPS for everything queued for this owner
redis.call("HSET", channelKey, "tps", tps)
redis.call("PERSIST", channelKey)

local state = redis.call("HMGET", channelKey, "sec", "used", "inflight", "paused_until")

-- a paused owner waits until the pause ends rather than being made ready
local pausedUntil = tonumber(state[4]) or 0
if pausedUntil > now then
    redis.call("ZREM", readyKey, owner)
    redis.call("ZADD", waitingKey, pausedUntil, owner)
    return 0
end

-- an owner that's used up this second's TPS waits until the next, otherwise make it ready so that this batch doesn't
-- wait on whatever the owner was waiting for, e.g. the rest of a batch
if tps > 0 and tonumber(state[1]) == nowSec and (tonumber(state[2]) or 0) >= tps then
    if not redis.call("ZSCORE", readyKey, owner) then
        redis.call("ZADD", waitingKey, "LT", nowSec + 1, owner)
    end
    return 0
end

redis.call("ZREM", waitingKey, owner)
redis.call("ZADD", readyKey, tonumber(state[3]) or 0, owner)
return 1
