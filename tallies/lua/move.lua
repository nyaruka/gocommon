-- merges the live hashes into the flushing hashes (which may be left over from a failed flush), giving the flushing
-- hash the generation after the last written one if it doesn't have one, and returns both flushing hashes
local liveKey, liveExpiriesKey, flushingKey, flushingExpiriesKey = KEYS[1], KEYS[2], KEYS[3], KEYS[4]
local lastGen = tonumber(ARGV[1])

-- flushing hashes left over from a failed flush which did commit can only be deleted, since merging new increments
-- into them would see them skipped as already written
local leftoverGen = redis.call("HGET", flushingKey, "_gen")
if leftoverGen and tonumber(leftoverGen) <= lastGen then
    redis.call("DEL", flushingKey, flushingExpiriesKey)
end

local live = redis.call("HGETALL", liveKey)
for i = 1, #live, 2 do
    redis.call("HINCRBY", flushingKey, live[i], live[i + 1])
end

-- leftover expiries were set first so they win
local liveExpiries = redis.call("HGETALL", liveExpiriesKey)
for i = 1, #liveExpiries, 2 do
    redis.call("HSETNX", flushingExpiriesKey, liveExpiries[i], liveExpiries[i + 1])
end

redis.call("DEL", liveKey, liveExpiriesKey)

if redis.call("EXISTS", flushingKey) == 1 then
    redis.call("HSETNX", flushingKey, "_gen", lastGen + 1)
end

return {redis.call("HGETALL", flushingKey), redis.call("HGETALL", flushingExpiriesKey)}
