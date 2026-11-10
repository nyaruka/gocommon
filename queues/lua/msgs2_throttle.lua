-- Shared by the scripts that throttle an owner by its TPS, so that they all agree on when it can next send.

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

    -- LEGACY WINDOW: charge as debt any usage by older versions, so that our debt covers what they send too
    if st.tps > 0 and st.used > st.debtUsed then
        st.debtUntil = st.debtUntil + (st.used - st.debtUsed) / st.tps
        st.debtUsed = st.used
        redis.call("HSET", chKey, "debt_until", st.debtUntil, "debt_sec", nowSec, "debt_used", st.debtUsed)
    end

    return st
end

-- Returns when an owner can next send, or 0 if it can send now. An owner can send while its debt is no more than a
-- second's worth. The returned time is computed the same way each time so that an owner woken at it is always found
-- able to send.
local function msgs2ThrottledUntil(st, now)
    if st.tps > 0 and st.debtUntil - 1 > now then
        return st.debtUntil - 1
    end
    return 0
end

-- Charges the given cost, which may be negative, against the debt of an owner with a TPS, capping it at the given
-- number of seconds.
local function msgs2ChargeDebt(chKey, st, cost, now, maxDebt)
    st.debtUntil = math.max(math.min(st.debtUntil + cost / st.tps, now + maxDebt), now)
    redis.call("HSET", chKey, "debt_until", st.debtUntil)
end
