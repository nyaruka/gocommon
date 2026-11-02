-- merges the live hash into the flushing hash (which may be left over from a failed flush), giving it the generation
-- after the last written one if it doesn't have one, and returns the flushing hash
local liveKey, flushingKey, lastGen = KEYS[1], KEYS[2], tonumber(ARGV[1])

-- a flushing hash left over from a failed flush which did commit can only be deleted, since merging new increments
-- into it would see them skipped as already written
local leftoverGen = redis.call("HGET", flushingKey, "_gen")
if leftoverGen and tonumber(leftoverGen) <= lastGen then
    redis.call("DEL", flushingKey)
end

local live = redis.call("HGETALL", liveKey)
for i = 1, #live, 2 do
    redis.call("HINCRBY", flushingKey, live[i], live[i + 1])
end
redis.call("DEL", liveKey)

if redis.call("EXISTS", flushingKey) == 1 then
    redis.call("HSETNX", flushingKey, "_gen", lastGen + 1)
end

return redis.call("HGETALL", flushingKey)
