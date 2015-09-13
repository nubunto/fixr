package main

import (
	"errors"
	"github.com/mediocregopher/radix.v2/redis"
	"fmt"
)

type FixrAccessor struct {
	client *redis.Client
}

func NewBackend(transport, port string) *FixrAccessor {
	client, err := redis.Dial(transport, port)
	if err != nil {
		panic(err)
	}
	return &FixrAccessor{client}
}

func (fa *FixrAccessor) Close() {
	fa.client.Close()
}

func (fa *FixrAccessor) Subscribe(ID int) (bool, error) {
	alreadySubscribed := fa.isSubscribed(ID)
	if alreadySubscribed {
		return false, errors.New("ID is already subscribed.")
	}
	if err := fa.client.Cmd("SADD", "users", ID).Err; err != nil {
		panic(err)
	}
	if err := fa.client.Cmd("HSET", fmt.Sprintf("user:%d", ID), "base", "USD").Err; err != nil {
		panic(err)
	}
	return true, nil
}

func (fa *FixrAccessor) isSubscribed(ID int) bool {
	isSubscribed, err := fa.client.Cmd("SISMEMBER", "users", ID).Int()
	if err != nil {
		panic(err)
	}
	return isSubscribed == 1
}

func (fa *FixrAccessor) Unsubscribe(ID int) (bool, error) {
	subscribed := fa.isSubscribed(ID)
	if !subscribed {
		return false, errors.New("ID is not subscribed")
	}
	err := fa.client.Cmd("SREM", "users", ID).Err
	if err != nil {
		panic(err)
	} else {
		return true, nil
	}
}

/*
  Redis data structures:

   * users :: SET
   * user:%d :: MAP
   * rates:%d :: LIST

*/

func (fa *FixrAccessor) GetRegistered() []string {
	members, err := fa.client.Cmd("SMEMBERS", "users").List()
	if err != nil { panic(err) }
	return members
}

func (fa *FixrAccessor) GetRates(ID int) []string {
	rates, err := fa.client.Cmd("LRANGE", fmt.Sprintf("rates:%d", ID), 0, -1).List()
	if err != nil { panic(err) }
	return rates
}

func (fa *FixrAccessor) SetRates(ID int, rates []string) {
	err := fa.client.Cmd("LPUSH", fmt.Sprintf("rates:%d", ID), rates).Err
	if err != nil { panic(err) }
}

func (fa *FixrAccessor) SetBase(ID int, base string) bool {
	if isValid := isValidBase(base); !isValid {
		return false
	}
	err := fa.client.Cmd("HSET", fmt.Sprintf("user:%d", ID), "base", base).Err
	if err != nil { panic(err) }
	return true
}

func isValidBase(base string) bool {
	_, ok := currencies[base]
	return ok
}

func (fa *FixrAccessor) GetSetting(ID int, prop string) string {
	prop, err := fa.client.Cmd("HGET", fmt.Sprintf("user:%d", ID), prop).Str()
	if err != nil { panic(err) }
	return prop
}
