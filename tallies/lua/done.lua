-- deletes the flushing hashes and, unless more increments have arrived since they were moved, removes the owner from
-- pending
local liveKey, flushingKey, flushingExpiriesKey, pendingKey, owner = KEYS[1], KEYS[2], KEYS[3], KEYS[4], ARGV[1]

redis.call("DEL", flushingKey, flushingExpiriesKey)

if redis.call("EXISTS", liveKey) == 0 then
    redis.call("SREM", pendingKey, owner)
end
