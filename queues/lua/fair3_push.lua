local queuedKey = KEYS[1]
local activeKey = KEYS[2]
local queue0Key = KEYS[3]
local queue1Key = KEYS[4]
local owner = ARGV[1]
local priority = tonumber(ARGV[2])
local task = ARGV[3]

-- we could just increment queued count but counting the queue sizes makes it self-correcting
local queuedCount = 0
if priority == 0 then
    queuedCount = redis.call("RPUSH", queue0Key, task) + redis.call("LLEN", queue1Key)
else
    queuedCount = redis.call("RPUSH", queue1Key, task) + redis.call("LLEN", queue0Key)
end

redis.call("ZADD", queuedKey, queuedCount, owner)
