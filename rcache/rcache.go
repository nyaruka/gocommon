package rcache

import (
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
)

const (
	keyPattern   = "%s:%s"
	datePattern  = "2006_01_02"
	secondsInDay = 60 * 60 * 24
)

var hasTask = redis.NewScript(3,
	`-- KEYS: [TodayKey, YesterdayKey, Key]
    local value = redis.call("hget", KEYS[1], KEYS[3])
    if (value ~= nil) then
	  return value
    end
    return redis.call("hget", KEYS[2], KEYS[3])
`)

// Get returns the cached value for key in the passed in group or nil
func Get(rc redis.Conn, group string, key string) (string, error) {
	todayKey := fmt.Sprintf(keyPattern, group, time.Now().UTC().Format(datePattern))
	yesterdayKey := fmt.Sprintf(keyPattern, group, time.Now().Add(time.Hour*-24).UTC().Format(datePattern))
	value, err := redis.String(hasTask.Do(rc, todayKey, yesterdayKey, key))
	if err != nil && err != redis.ErrNil {
		return "", errors.Wrapf(err, "error getting value for group: %s and key: %s", group, key)
	}
	return value, nil
}

// Set sets the cached value for key for the passed in group. It will be cached for at least 24 hours but no longer than 48 hours
func Set(rc redis.Conn, group string, key string, value string) error {
	dateKey := fmt.Sprintf(keyPattern, group, time.Now().UTC().Format(datePattern))
	rc.Send("hset", dateKey, key, value)
	rc.Send("expire", dateKey, secondsInDay)
	_, err := rc.Do("")
	if err != nil {
		return errors.Wrapf(err, "error setting value for group: %s, key: %s, value: %s", group, key, value)
	}
	return nil
}

// Delete removes the value with the passed in key
func Delete(rc redis.Conn, group string, key string) error {
	todayKey := fmt.Sprintf(keyPattern, group, time.Now().UTC().Format(datePattern))
	yesterdayKey := fmt.Sprintf(keyPattern, group, time.Now().Add(time.Hour*-24).UTC().Format(datePattern))
	rc.Send("hdel", todayKey, key)
	rc.Send("hdel", yesterdayKey, key)
	_, err := rc.Do("")
	if err != nil {
		return errors.Wrapf(err, "error deleting value for group: %s and key: %s", group, key)
	}
	return nil
}

// Clear removes all values for the passed in group
func Clear(rc redis.Conn, group string) error {
	todayKey := fmt.Sprintf(keyPattern, group, time.Now().UTC().Format(datePattern))
	yesterdayKey := fmt.Sprintf(keyPattern, group, time.Now().Add(time.Hour*-24).UTC().Format(datePattern))
	rc.Send("del", todayKey)
	rc.Send("del", yesterdayKey)
	_, err := rc.Do("")
	return err
}
