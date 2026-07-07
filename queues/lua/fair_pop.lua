local queuedKey = KEYS[1]
local activeKey = KEYS[2]
local pausedKey = KEYS[3]
local tempKey = KEYS[4]
local inflightKey = KEYS[5]
local expiresKey = KEYS[6]
local deadKey = KEYS[7]
local keyBase = ARGV[1]
local maxActivePerOwner = ARGV[2]
local now = ARGV[3]
local deadline = ARGV[4]
local maxAttempts = tonumber(ARGV[5])

-- owner queue keys share our hash tag so are safe to construct here even in cluster mode
local function queueKeys(owner)
    return "{" .. keyBase .. "}:o:" .. owner .. "/0", "{" .. keyBase .. "}:o:" .. owner .. "/1"
end

local function decrActive(owner)
    local activeCount = tonumber(redis.call("ZINCRBY", activeKey, -1, owner))
    if activeCount <= 0 then
        redis.call("ZREM", activeKey, owner)
    end
end

-- sets an owner's queued score from their actual queue sizes
local function updateQueued(owner, q0Key, q1Key)
    local size = redis.call("LLEN", q0Key) + redis.call("LLEN", q1Key)
    if size > 0 then
        redis.call("ZADD", queuedKey, size, owner)
    else
        redis.call("ZREM", queuedKey, owner)
    end
end

-- first look for in-flight tasks whose leases have expired, i.e. tasks whose consumers died or ran past their leases
local expired = redis.call("ZRANGEBYSCORE", expiresKey, "-inf", now, "LIMIT", 0, 10)
for _, taskID in ipairs(expired) do
    local record = redis.call("HGET", inflightKey, taskID)
    if not record then
        -- orphaned expiry entry.. just remove it
        redis.call("ZREM", expiresKey, taskID)
    else
        local sep1 = string.find(record, "|", 1, true)
        local sep2 = string.find(record, "|", sep1 + 1, true)
        local sep3 = string.find(record, "|", sep2 + 1, true)
        local owner = string.sub(record, 1, sep1 - 1)
        local priority = string.sub(record, sep1 + 1, sep2 - 1)
        local attempts = tonumber(string.sub(record, sep2 + 1, sep3 - 1))
        local task = string.sub(record, sep3 + 1)

        if attempts >= maxAttempts then
            -- task has been delivered too many times.. move to the dead list
            redis.call("RPUSH", deadKey, taskID .. "|" .. record)
            redis.call("LTRIM", deadKey, -1000, -1)
            redis.call("HDEL", inflightKey, taskID)
            redis.call("ZREM", expiresKey, taskID)
            decrActive(owner)
        elseif redis.call("SISMEMBER", pausedKey, owner) == 1 then
            -- owner is paused so re-arm the lease.. the task will be redelivered after they're resumed
            redis.call("ZADD", expiresKey, deadline, taskID)
        else
            -- redeliver with a new lease.. active count is unchanged as the task still holds its slot
            attempts = attempts + 1
            redis.call("HSET", inflightKey, taskID, owner .. "|" .. priority .. "|" .. attempts .. "|" .. task)
            redis.call("ZADD", expiresKey, deadline, taskID)
            return {taskID, owner, attempts, task}
        end
    end
end

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
    return false
end

local q0Key, q1Key = queueKeys(owner)

-- pop off their queues (priority first)
local priority = "1"
local payload = redis.call("LPOP", q1Key)
if not payload then
    priority = "0"
    payload = redis.call("LPOP", q0Key)
end

if not payload then
    -- owner had no queued tasks after all.. fix their queued score and tell caller to try again
    updateQueued(owner, q0Key, q1Key)
    return {}
end

updateQueued(owner, q0Key, q1Key)

local sep = string.find(payload, "|", 1, true)
if not sep then
    return redis.error_reply("invalid task payload: " .. payload)
end
local taskID = string.sub(payload, 1, sep - 1)
local task = string.sub(payload, sep + 1)

-- record task as in-flight with a lease
redis.call("ZINCRBY", activeKey, 1, owner)
redis.call("HSET", inflightKey, taskID, owner .. "|" .. priority .. "|1|" .. task)
redis.call("ZADD", expiresKey, deadline, taskID)

return {taskID, owner, 1, task}
