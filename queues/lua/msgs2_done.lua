-- KEYS: [Channel, Ready, Waiting, RequeueQueue]
-- ARGV: [Owner, Cost, NowMicros, Item?]
local channelKey, readyKey, waitingKey, requeueKey = KEYS[1], KEYS[2], KEYS[3], KEYS[4]
local owner, cost, item = ARGV[1], math.max(0, tonumber(ARGV[2])), ARGV[4]

local now = tonumber(ARGV[3]) / 1000000

local inflight = redis.call("HINCRBY", channelKey, "inflight", -1)
if inflight < 0 then
    inflight = 0
    redis.call("HSET", channelKey, "inflight", 0)
end

-- swap the estimated cost charged by pop for the actual cost. This assumes the estimate was charged at the current TPS,
-- so is approximate if the TPS has changed since.
local st = msgs2ThrottleState(channelKey, now)
if st.tps > 0 then
    msgs2ChargeDebt(channelKey, st, cost - 1, now)

    -- a send that cost less than estimated may mean that a waiting owner can send sooner. A requeue reschedules the
    -- owner below anyway.
    if cost < 1 and not item and redis.call("ZSCORE", waitingKey, owner) then
        local pausedUntil = tonumber(redis.call("HGET", channelKey, "paused_until")) or 0
        if pausedUntil <= now then
            redis.call("ZADD", waitingKey, "LT", math.max(msgs2ThrottledUntil(st, now), now), owner)
        end
    end
end

-- an item that was popped but never sent goes back to be popped before anything else, in the order put back
if item then
    redis.call("ZADD", requeueKey, now, item)
    msgs2Schedule(channelKey, readyKey, waitingKey, owner, now)
end

if redis.call("ZSCORE", readyKey, owner) then
    redis.call("ZADD", readyKey, msgs2ReadyScore(inflight, redis.call("HGET", channelKey, "last_pop")), owner)

-- an owner that's neither ready nor waiting has nothing queued, so once nothing is in flight let its state expire
elseif inflight == 0 and not redis.call("ZSCORE", waitingKey, owner) then
    redis.call("EXPIRE", channelKey, 60)
end
