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

-- items are batches of messages for a single contact, of which we take the first
local ok, batch = pcall(cjson.decode, value)
if not ok or type(batch) ~= "table" or batch[1] == nil then
    -- not a non-empty array so there's nothing to send
    return {"retry", ""}
end

redis.call("ZADD", readyKey, redis.call("HINCRBY", chKey, "inflight", 1), owner)

if tps > 0 then
    local cost = 1
    if type(batch[1]) == "table" and type(batch[1]["tps_cost"]) == "number" then
        cost = math.max(1, math.floor(batch[1]["tps_cost"]))
    end
    redis.call("HSET", chKey, "sec", nowSec, "used", used + cost)
end

-- cjson can't re-encode messages as they were, e.g. empty arrays become objects, so we split the batch's JSON at the
-- end of its first element, i.e. the first comma or bracket outside of a nested value or string
local first = string.find(value, "%[") + 1
local pos, depth = first, 0
local split
while true do
    local i = string.find(value, '[%[%]{}",]', pos)
    local c = string.sub(value, i, i)
    if c == '"' then
        repeat
            i = string.find(value, '["\\]', i + 1)
            local escape = string.sub(value, i, i) == "\\"
            if escape then
                i = i + 1
            end
        until not escape
    elseif c == "[" or c == "{" then
        depth = depth + 1
    elseif depth > 0 and (c == "]" or c == "}") then
        depth = depth - 1
    elseif depth == 0 then
        split = i
        break
    end
    pos = i + 1
end

-- and schedule the rest a few seconds out so they're sent in order
if batch[2] ~= nil then
    redis.call("ZADD", highKey, now + 3, "[" .. string.sub(value, split + 1))
end

return {owner, string.match(string.sub(value, first, split - 1), "^%s*(.-)%s*$")}
