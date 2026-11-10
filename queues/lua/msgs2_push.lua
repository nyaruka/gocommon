-- KEYS: [Queue, Channel, Ready, Waiting]
-- ARGV: [Owner, TPS, Batch, NowMicros]
local queueKey, channelKey, readyKey, waitingKey = KEYS[1], KEYS[2], KEYS[3], KEYS[4]
local owner, tps, batch = ARGV[1], tonumber(ARGV[2]), ARGV[3]

local now = tonumber(ARGV[4]) / 1000000

redis.call("ZADD", queueKey, now, batch)

-- the latest push sets the TPS for everything queued for this owner
redis.call("HSET", channelKey, "tps", tps)
redis.call("PERSIST", channelKey)

local state = redis.call("HMGET", channelKey, "inflight", "paused_until")

-- a paused owner waits until the pause ends rather than being made ready
local pausedUntil = tonumber(state[2]) or 0
if pausedUntil > now then
    redis.call("ZREM", readyKey, owner)
    redis.call("ZADD", waitingKey, pausedUntil, owner)
    return 0
end

-- an owner in debt waits until it can send, otherwise make it ready so that this batch doesn't wait on whatever the
-- owner was waiting for, e.g. the rest of a batch
local throttledUntil = msgs2ThrottledUntil(msgs2ThrottleState(channelKey, now), now)
if throttledUntil > 0 then
    if not redis.call("ZSCORE", readyKey, owner) then
        redis.call("ZADD", waitingKey, "LT", throttledUntil, owner)
    end
    return 0
end

redis.call("ZREM", waitingKey, owner)
redis.call("ZADD", readyKey, tonumber(state[1]) or 0, owner)
return 1
