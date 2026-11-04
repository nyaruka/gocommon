-- moves up to size fields from flushing into a new batch, returning its id and its fields and amounts as a flat list.
-- Its id is one more than the greater of the store's last batch and the last id used here, so ids only go up even if
-- either side loses data. Fails if there's already a batch, which means another flush is running.
local flushingKey, batchKey, batchIDKey = KEYS[1], KEYS[2], KEYS[3]
local lastBatch, size = tonumber(ARGV[1]), tonumber(ARGV[2])

if redis.call("EXISTS", batchKey) == 1 then
    return redis.error_reply("a batch is already being flushed")
end

local kvs = redis.call("HRANDFIELD", flushingKey, size, "WITHVALUES")
if #kvs == 0 then
    return {0, {}}
end

-- in chunks to stay under the limit on how many values unpack can return
local chunk = 1000
for i = 1, #kvs, chunk do
    local kvChunk, fieldChunk = {}, {}
    for j = i, math.min(i + chunk - 1, #kvs), 2 do
        table.insert(kvChunk, kvs[j])
        table.insert(kvChunk, kvs[j + 1])
        table.insert(fieldChunk, kvs[j])
    end
    redis.call("HSET", batchKey, unpack(kvChunk))
    redis.call("HDEL", flushingKey, unpack(fieldChunk))
end

local id = math.max(lastBatch, tonumber(redis.call("GET", batchIDKey) or "0")) + 1
redis.call("SET", batchIDKey, id)
return {id, kvs}
