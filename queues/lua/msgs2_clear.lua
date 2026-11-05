-- KEYS: [HighQueue, BulkQueue, Channel, Ready, Waiting]
-- ARGV: [Owner]
local highKey, bulkKey, channelKey, readyKey, waitingKey = KEYS[1], KEYS[2], KEYS[3], KEYS[4], KEYS[5]
local owner = ARGV[1]

redis.call("DEL", highKey, bulkKey)
redis.call("ZREM", readyKey, owner)
redis.call("ZREM", waitingKey, owner)

-- the owner's state is kept for sends still in flight but with nothing queued it can expire
redis.call("EXPIRE", channelKey, 60)
