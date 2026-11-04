-- renames live to flushing unless a flushing hash is left over from a failed flush, returning whether it did
local liveKey, flushingKey = KEYS[1], KEYS[2]

if redis.call("EXISTS", flushingKey) == 0 and redis.call("EXISTS", liveKey) == 1 then
    redis.call("RENAME", liveKey, flushingKey)
    return 1
end
return 0
