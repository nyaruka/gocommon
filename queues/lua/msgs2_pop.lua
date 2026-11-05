-- KEYS: [Ready, Waiting]
-- ARGV: [KeyBase, WakeLimit, NowMicros]
local readyKey, waitingKey = KEYS[1], KEYS[2]
local keyBase, wakeLimit = ARGV[1], tonumber(ARGV[2])

local now = tonumber(ARGV[3]) / 1000000
local nowSec = math.floor(now)

-- channel keys share our hash tag so are safe to construct here even in cluster mode
local function channelKey(owner)
    return keyBase .. ":ch:" .. owner
end

-- make ready any owners whose wait is over
for _, owner in ipairs(redis.call("ZRANGEBYSCORE", waitingKey, "-inf", now, "LIMIT", 0, wakeLimit)) do
    redis.call("ZREM", waitingKey, owner)
    redis.call("ZADD", readyKey, tonumber(redis.call("HGET", channelKey(owner), "inflight")) or 0, owner)
end

-- take the ready owner with the fewest sends in flight
local owner = redis.call("ZRANGE", readyKey, 0, 0)[1]
if not owner then
    return {"empty", ""}
end

local chKey = channelKey(owner)

local function park(untilTime)
    redis.call("ZREM", readyKey, owner)
    redis.call("ZADD", waitingKey, untilTime, owner)
    return {"retry", ""}
end

local state = redis.call("HMGET", chKey, "tps", "sec", "used")
local tps = tonumber(state[1]) or 0
local used = 0
if tonumber(state[2]) == nowSec then
    used = tonumber(state[3]) or 0
end

if tps > 0 and used >= tps then
    return park(nowSec + 1)
end

local function head(key)
    local result = redis.call("ZRANGE", key, 0, 0, "WITHSCORES")
    if result[1] then
        return result[1], tonumber(result[2])
    end
    return nil, nil
end

-- high priority items first, then bulk
local highKey, bulkKey = keyBase .. ":q:" .. owner .. "/1", keyBase .. ":q:" .. owner .. "/0"
local highValue, highScore = head(highKey)
local bulkValue, bulkScore = head(bulkKey)
local fromKey, value

if highValue and highScore <= now then
    fromKey, value = highKey, highValue
elseif bulkValue and bulkScore <= now then
    fromKey, value = bulkKey, bulkValue
elseif highValue or bulkValue then
    -- only items scheduled for later, so wait until the first of those
    return park(math.min(highScore or math.huge, bulkScore or math.huge))
else
    redis.call("ZREM", readyKey, owner)

    -- nothing queued so let the owner's state expire, but not before this second's usage is done. Sends in flight
    -- are forgotten so that any never marked done, e.g. by a worker that died, don't count against the owner
    -- forever - those that are still in flight just complete to zero.
    redis.call("HSET", chKey, "inflight", 0)
    redis.call("EXPIRE", chKey, 60)
    return {"retry", ""}
end

redis.call("ZREM", fromKey, value)
redis.call("ZADD", readyKey, redis.call("HINCRBY", chKey, "inflight", 1), owner)

-- items are batches of messages for a single contact, of which we take the first
local batch = cjson.decode(value)
local popped = table.remove(batch, 1)

if tps > 0 then
    local cost = 1
    if type(popped) == "table" and type(popped["tps_cost"]) == "number" then
        cost = math.max(1, math.floor(popped["tps_cost"]))
    end
    redis.call("HSET", chKey, "sec", nowSec, "used", used + cost)
end

-- and schedule the rest a few seconds out so they're sent in order
if #batch > 0 then
    redis.call("ZADD", highKey, now + 3, cjson.encode(batch))
end

return {owner, cjson.encode(popped)}
