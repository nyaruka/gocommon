
local function dumpZSet(key)
    local arr = redis.call("ZRANGE", key, 0, -1, "WITHSCORES")
    local table = {}
    for i = 1, #arr, 2 do
        table[arr[i]] = tonumber(arr[i + 1])
    end
    return table
end

-- LUA doesn't have a way to return an empty array as [] and JSON doesn't have sets.. 
-- so return a set like {"a", "b"} as an object like {"a": 1, "b": 1}
local function dumpSet(key)
    local arr = redis.call("SMEMBERS", key)
    local table = {}
    for _, v in ipairs(arr) do
        table[v] = 1
    end
    return table
end

local queuedKey = KEYS[1]
local activeKey = KEYS[2]
local pausedKey = KEYS[3]

local result = {}
result["queued"] = dumpZSet(queuedKey)
result["active"] = dumpZSet(activeKey)
result["paused"] = dumpSet(pausedKey) 

return cjson.encode(result)