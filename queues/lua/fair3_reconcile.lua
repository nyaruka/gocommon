local activeKey = KEYS[1]
local inflightKey = KEYS[2]

-- count in-flight tasks per owner
local counts = {}
for _, record in ipairs(redis.call("HVALS", inflightKey)) do
    local sep = string.find(record, "|", 1, true)
    local owner = string.sub(record, 1, sep - 1)
    counts[owner] = (counts[owner] or 0) + 1
end

-- rebuild active counts from those, healing any drift from consumers which died holding a task
redis.call("DEL", activeKey)
for owner, count in pairs(counts) do
    redis.call("ZADD", activeKey, count, owner)
end
