-- KEYS: [Channel, Ready, Waiting]
-- ARGV: [Owner, Cost, MaxDebtSecs, NowMicros]
local channelKey, readyKey, waitingKey = KEYS[1], KEYS[2], KEYS[3]
local owner, cost, maxDebt = ARGV[1], math.max(0, tonumber(ARGV[2])), tonumber(ARGV[3])

local now = tonumber(ARGV[4]) / 1000000

local inflight = redis.call("HINCRBY", channelKey, "inflight", -1)
if inflight < 0 then
    inflight = 0
    redis.call("HSET", channelKey, "inflight", 0)
end

-- swap the estimated cost charged by pop for the actual cost, capping the debt so that a very expensive send can't
-- block the owner for long
local st = msgs2ThrottleState(channelKey, now)
if st.tps > 0 then
    msgs2ChargeDebt(channelKey, st, cost - 1, now, maxDebt)

    -- a send that cost less than estimated may mean that a waiting owner can send sooner
    if cost < 1 and redis.call("ZSCORE", waitingKey, owner) then
        local pausedUntil = tonumber(redis.call("HGET", channelKey, "paused_until")) or 0
        if pausedUntil <= now then
            redis.call("ZADD", waitingKey, "LT", math.max(msgs2ThrottledUntil(st, now), now), owner)
        end
    end
end

if redis.call("ZSCORE", readyKey, owner) then
    redis.call("ZADD", readyKey, inflight, owner)

-- an owner that's neither ready nor waiting has nothing queued, so once nothing is in flight let its state expire
elseif inflight == 0 and not redis.call("ZSCORE", waitingKey, owner) then
    redis.call("EXPIRE", channelKey, 60)
end
