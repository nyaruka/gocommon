local queuedKey = KEYS[1]
local activeKey = KEYS[2]
local pausedKey = KEYS[3]
local tempKey = KEYS[4]
local maxActivePerOwner = ARGV[1]

-- create a new set which is union of queued and active owners, with scores from active
redis.call("ZUNIONSTORE", tempKey, 2, queuedKey, activeKey, "WEIGHTS", 0, 1)

-- intersect with queued owners again to remove any active owners that have no queued tasks
redis.call("ZINTERSTORE", tempKey, 2, tempKey, queuedKey, "WEIGHTS", 1, 0)

-- substract paused owners from this set
redis.call("ZDIFFSTORE", tempKey, 2, tempKey, pausedKey)

-- never leave anything without an expiry...
redis.call("EXPIRE", tempKey, 60)

-- get the owner with the least active tasks
local result = redis.call("ZRANGEBYSCORE", tempKey, "-inf", "(" .. maxActivePerOwner, "LIMIT", 0, 1)

-- nothing? return nothing
local owner = result[1]
if not owner then
    return ""
end

-- decrement queued count for this owner
local queuedCount = tonumber(redis.call("ZINCRBY", queuedKey, -1, owner))
if queuedCount <= 0 then
    redis.call("ZREM", queuedKey, owner)
end

-- increment active count for this owner to prevent races
redis.call("ZINCRBY", activeKey, 1, owner)

return owner