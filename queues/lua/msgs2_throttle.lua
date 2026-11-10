-- Shared by the scripts that throttle an owner by its TPS, so that they all agree on when it can next send.

-- the most debt an owner can be in, in seconds, so that one very expensive send can't block it for long
local msgs2MaxDebt = 10

-- Reads the state of an owner that its throttling depends on.
local function msgs2ThrottleState(chKey, now)
    local s = redis.call("HMGET", chKey, "tps", "debt_until", "sec", "used", "debt_sec", "debt_used")
    local st = {tps = tonumber(s[1]) or 0, debtUntil = math.max(tonumber(s[2]) or 0, now), used = 0, debtUsed = 0}

    -- LEGACY WINDOW: this second's usage by all versions, and how much of that is already charged as debt
    local nowSec = math.floor(now)
    if tonumber(s[3]) == nowSec then
        st.used = tonumber(s[4]) or 0
        if tonumber(s[5]) == nowSec then
            st.debtUsed = tonumber(s[6]) or 0
        end
    end

    -- LEGACY WINDOW: charge as debt any usage by older versions, so that our debt covers what they send too. Only
    -- usage seen within its second is charged, so this is approximate.
    if st.tps > 0 and st.used > st.debtUsed then
        st.debtUntil = math.min(st.debtUntil + (st.used - st.debtUsed) / st.tps, now + msgs2MaxDebt)
        st.debtUsed = st.used
        redis.call("HSET", chKey, "debt_until", st.debtUntil, "debt_sec", nowSec, "debt_used", st.debtUsed)
    end

    return st
end

-- Returns when an owner can next send, or 0 if it can send now. An owner can send while its debt, with the estimated
-- cost of the send, is no more than a second's worth. The returned time is computed the same way each time so that an
-- owner woken at it is always found able to send.
local function msgs2ThrottledUntil(st, now)
    if st.tps > 0 then
        local wakeAt = st.debtUntil + 1 / st.tps - 1
        if wakeAt > now then
            return wakeAt
        end
    end
    return 0
end

-- Charges the given cost, which may be negative, against the debt of an owner with a TPS.
local function msgs2ChargeDebt(chKey, st, cost, now)
    st.debtUntil = math.max(math.min(st.debtUntil + cost / st.tps, now + msgs2MaxDebt), now)
    redis.call("HSET", chKey, "debt_until", st.debtUntil)
end
