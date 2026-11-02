-- merges the live hash into the flushing hash (which may be left over from a failed flush), giving it a generation if
-- it doesn't have one, and returns the flushing hash
local liveKey, flushingKey, gen = KEYS[1], KEYS[2], ARGV[1]

local live = redis.call("HGETALL", liveKey)
for i = 1, #live, 2 do
    redis.call("HINCRBY", flushingKey, live[i], live[i + 1])
end
redis.call("DEL", liveKey)

if redis.call("EXISTS", flushingKey) == 1 then
    redis.call("HSETNX", flushingKey, "_gen", gen)
end

return redis.call("HGETALL", flushingKey)
