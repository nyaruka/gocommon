-- deletes the batch if it's still the one with the given id
local batchKey, batchIDKey, id = KEYS[1], KEYS[2], ARGV[1]

if redis.call("GET", batchIDKey) == id then
    redis.call("DEL", batchKey)
end
