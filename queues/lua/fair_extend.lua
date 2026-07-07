local expiresKey = KEYS[1]
local inflightKey = KEYS[2]
local deadline = ARGV[1]
local taskID = ARGV[2]
local attempts = ARGV[3]

-- only extend if the task is still leased to this caller, i.e. hasn't been redelivered since
local record = redis.call("HGET", inflightKey, taskID)
if record then
    local sep1 = string.find(record, "|", 1, true)
    local sep2 = string.find(record, "|", sep1 + 1, true)
    local sep3 = string.find(record, "|", sep2 + 1, true)
    if string.sub(record, sep2 + 1, sep3 - 1) == attempts then
        redis.call("ZADD", expiresKey, deadline, taskID)
        return 1
    end
end

return 0
