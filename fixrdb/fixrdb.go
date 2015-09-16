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

var ErrInvalidBase = errors.New("This base is invalid.")
var ErrAlreadySubscribed = errors.New("You are already subscribed")
var ErrNotSubscribed = errors.New("You are not subscribed.")

type RedisError struct {
	msg string
}

func (r RedisError) Error() string {
	return "Redis error: " + r.msg
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
	var err error
	alreadySubscribed, err := fa.isSubscribed(ID)
	if alreadySubscribed {
		return false, ErrAlreadySubscribed
	}
	err = fa.client.Cmd("SADD", "users", ID).Err
	err = fa.client.Cmd("HSET", fmt.Sprintf("user:%d", ID), "base", "USD").Err
	if err != nil {
		return false, RedisError{err.Error()}
	}
	return true, nil
}

func (fa *FixrDB) isSubscribed(ID int) (bool, error) {
	isSubscribed, err := fa.client.Cmd("SISMEMBER", "users", ID).Int()
	if err != nil {
		return false, RedisError{err.Error()}
	}
	return isSubscribed == 1, nil
}

func (fa *FixrDB) Unsubscribe(ID int) (bool, error) {
	var err error
	subscribed, err := fa.isSubscribed(ID)
	if !subscribed {
		return false, ErrNotSubscribed
	}
	err = fa.client.Cmd("SREM", "users", ID).Err
	if err != nil {
		return false, RedisError{err.Error()}
	}
	return true, nil
}

/*
  Redis data structures:

   * users :: SET
   * user:%d :: MAP
   * rates:%d :: SET 

*/

func (fa *FixrDB) GetRegistered() ([]string, error) {
	members, err := fa.client.Cmd("SMEMBERS", "users").List()
	if err != nil {
		return nil, RedisError{err.Error()}
	}
	return members, nil
}

func (fa *FixrDB) GetRates(ID int) ([]string, error) {
	rates, err := fa.client.Cmd("SMEMBERS", fmt.Sprintf("rates:%d", ID)).List()
	if err != nil {
		return rates, RedisError{err.Error()}
	}
	return rates, nil
}

func (fa *FixrDB) SetRates(ID int, rates []string) error {
	err := fa.client.Cmd("SADD", fmt.Sprintf("rates:%d", ID), rates).Err
	if err != nil {
		return RedisError{err.Error()}
	}
	return nil
}

func (fa *FixrDB) RemoveRate(ID int, rate string) error {
	err := fa.client.Cmd("SREM", fmt.Sprintf("rates:%d", ID), rate).Err
	if err != nil {
		return RedisError{err.Error()}
	}
	return nil
}

func (fa *FixrDB) SetBase(ID int, base string) (bool, error) {
	if isValid := fixerio.IsValidBase(base); !isValid {
		return false, ErrInvalidBase 
	}
	err := fa.client.Cmd("HSET", fmt.Sprintf("user:%d", ID), "base", base).Err
	if err != nil {
		return false, RedisError{err.Error()}
	}
	return true, nil
}

func (fa *FixrDB) ClearRates(ID int) error {
	err := fa.client.Cmd("DEL", fmt.Sprintf("rates:%d", ID)).Err
	if err != nil {
		return RedisError{err.Error()}
	}
	return nil
}


func (fa *FixrDB) GetSetting(ID int, prop string) (string, error) {
	prop, err := fa.client.Cmd("HGET", fmt.Sprintf("user:%d", ID), prop).Str()
	if err != nil {
		return "", RedisError{err.Error()}
	}
	return prop, nil
}
