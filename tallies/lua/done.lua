-- deletes the flushing hash and, unless more increments have arrived since it was moved, removes the owner from pending
local liveKey, flushingKey, pendingKey, owner = KEYS[1], KEYS[2], KEYS[3], ARGV[1]

redis.call("DEL", flushingKey)

if redis.call("EXISTS", liveKey) == 0 then
    redis.call("SREM", pendingKey, owner)
end
