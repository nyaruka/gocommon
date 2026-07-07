local activeKey = KEYS[1]
local inflightKey = KEYS[2]
local expiresKey = KEYS[3]
local taskID = ARGV[1]

local record = redis.call("HGET", inflightKey, taskID)
if not record then
    -- lease already expired and task was reclaimed.. nothing to do
    return 0
end

local sep = string.find(record, "|", 1, true)
local owner = string.sub(record, 1, sep - 1)

redis.call("HDEL", inflightKey, taskID)
redis.call("ZREM", expiresKey, taskID)

-- decrement our active task count for this owner, removing if zero (or somehow negative)
local activeCount = tonumber(redis.call("ZINCRBY", activeKey, -1, owner))
if activeCount <= 0 then
    redis.call("ZREM", activeKey, owner)
end

return 1
