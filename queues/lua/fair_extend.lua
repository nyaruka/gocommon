local expiresKey = KEYS[1]
local deadline = ARGV[1]
local taskID = ARGV[2]

-- only extend if the task still has a lease
if redis.call("ZSCORE", expiresKey, taskID) then
    redis.call("ZADD", expiresKey, deadline, taskID)
    return 1
end

return 0
