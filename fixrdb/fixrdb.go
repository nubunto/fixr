package fixrdb

import (
	"errors"
	"fmt"
	"fixr/fixerio"
	"github.com/mediocregopher/radix.v2/redis"
)

type FixrDB struct {
	client *redis.Client
}

func New(transport, port string) *FixrDB {
	client, err := redis.Dial(transport, port)
	if err != nil {
		panic(err)
	}
	return &FixrDB{client}
}

func (fa *FixrDB) Close() {
	fa.client.Close()
}

func (fa *FixrDB) Subscribe(ID int) (bool, error) {
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

func (fa *FixrDB) isSubscribed(ID int) bool {
	isSubscribed, err := fa.client.Cmd("SISMEMBER", "users", ID).Int()
	if err != nil {
		panic(err)
	}
	return isSubscribed == 1
}

func (fa *FixrDB) Unsubscribe(ID int) (bool, error) {
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
   * rates:%d :: SET 

*/

func (fa *FixrDB) GetRegistered() []string {
	members, err := fa.client.Cmd("SMEMBERS", "users").List()
	if err != nil {
		panic(err)
	}
	return members
}

func (fa *FixrDB) GetRates(ID int) []string {
	rates, err := fa.client.Cmd("SMEMBERS", fmt.Sprintf("rates:%d", ID)).List()
	if err != nil {
		panic(err)
	}
	return rates
}

func (fa *FixrDB) SetRates(ID int, rates []string) {
	err := fa.client.Cmd("SADD", fmt.Sprintf("rates:%d", ID), rates).Err
	if err != nil {
		panic(err)
	}
}

func (fa *FixrDB) RemoveRate(ID int, rate string) {
	err := fa.client.Cmd("SREM", fmt.Sprintf("rates:%d", ID), rate).Err
	if err != nil {
		panic(err)
	}
}

func (fa *FixrDB) SetBase(ID int, base string) bool {
	if isValid := fixerio.IsValidBase(base); !isValid {
		return false
	}
	err := fa.client.Cmd("HSET", fmt.Sprintf("user:%d", ID), "base", base).Err
	if err != nil {
		panic(err)
	}
	return true
}

func (fa *FixrDB) ClearRates(ID int) {
	err := fa.client.Cmd("DEL", fmt.Sprintf("rates:%d", ID)).Err
	if err != nil {
		panic(err)
	}
}


func (fa *FixrDB) GetSetting(ID int, prop string) string {
	prop, err := fa.client.Cmd("HGET", fmt.Sprintf("user:%d", ID), prop).Str()
	if err != nil {
		panic(err)
	}
	return prop
}
