//go:build client
// +build client

package main

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// Redis runtime context and database objects
var (
	ctx = context.Background()
	// We reinitialise this client if a startup config is present
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
)

func reinitRedis() (reinitialised bool) {
	reinitialised = false

	if ddrmAppConfig.RedisServer != rdb.Options().Addr {
		rdb = redis.NewClient(&redis.Options{
			Addr:     ddrmAppConfig.RedisServer,
			Password: ddrmAppConfig.RedisPassword,
			DB:       ddrmAppConfig.RedisDatabase,
		})

		reinitialised = true
	}

	return
}

func cacheKey(fqdn string, recordType DdrmRecordType) string {
	return ddrmAppConfig.RedisKeyPrefix + ":" + fqdn + ":" + string(recordType)
}

// check Redis for current cached data
func getCachedValues(fqdn string, recordType DdrmRecordType) (answer []string) {
	dbgf(ddrmDebugTryingCache, fqdn, string(recordType))

	answer = []string{}

	if stateUseRedis {

		cachedKey := cacheKey(fqdn, recordType)
		members, err := rdb.SMembers(ctx, cachedKey).Result()

		if err != nil {
			return answer
		}

		answer = members
		return
	}

	dbgf(ddrmDebugNoCache, fqdn, string(recordType))

	return
}

// save values in Redis as sets
func setCachedValues(fqdn string, recordType DdrmRecordType, answer []string) (success bool) {
	success = false

	if stateUseRedis {
		cachedKey := cacheKey(fqdn, recordType)

		// delete the key so we can reuse it
		err := rdb.Del(ctx, cachedKey).Err()

		if err != nil {
			return
		}

		// add each answer one at a time because SAdd appears to be bugged
		for _, a := range answer {
			err = rdb.SAdd(ctx, cachedKey, a).Err()
			if err != nil {
				return
			}
		}

		success = true
	}

	return
}
