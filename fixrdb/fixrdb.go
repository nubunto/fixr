/*
  Redis data structures:

   * users :: SET
   * user:%d :: MAP
   * rates:%d :: SET

   These are the data structures FixrBot uses to keep track of:
    * Users names and currency base
    * Registered users
    * The rates preferences of a given user.
*/
package fixrdb

import (
	"errors"
	"fixr/fixerio"
	"fmt"
	"github.com/mediocregopher/radix.v2/redis"
)

// The FixrDB access the backend, that is Redis.
type FixrDB struct {
	client *redis.Client
}

// Throws ErrInvalidBase when the base is invalid.
var ErrInvalidBase = errors.New("This base is invalid.")

// Throws ErrAlreadySubscribed when it tries to subscribe an ID but he already is subscribed.
var ErrAlreadySubscribed = errors.New("You are already subscribed")

// Throws ErroNotSubscribed when user should be subscribed but isn't.
var ErrNotSubscribed = errors.New("You are not subscribed.")

// Returns an generic RedisError when appropriate.
// Today, this doesn't do anything, but we can type match on this if we have to.
type RedisError struct {
	msg string
}

// Implement error interface.
func (r RedisError) Error() string {
	return "Redis error: " + r.msg
}

// New creates a *FixrDB and dials a connection to Redis.
// If redis isn't available on given transport and port, it panics.
func New(transport, port string) *FixrDB {
	client, err := redis.Dial(transport, port)
	if err != nil {
		panic(err)
	}
	return &FixrDB{client}
}

// Closes underlying connection with Redis.
func (fa *FixrDB) Close() {
	fa.client.Close()
}

// Subscribes an ID, if he isn't already subscribed.
// Returns ErrAlreadySubscribed if already subscribed
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

// Returns true when user is already subscribed.
// and a redis error, if there is one.
func (fa *FixrDB) isSubscribed(ID int) (bool, error) {
	isSubscribed, err := fa.client.Cmd("SISMEMBER", "users", ID).Int()
	if err != nil {
		return false, RedisError{err.Error()}
	}
	return isSubscribed == 1, nil
}

// Unsubscribes a user, if he isn't already.
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

// Returns all registered users
// Redis error otherwise
func (fa *FixrDB) GetRegistered() ([]string, error) {
	members, err := fa.client.Cmd("SMEMBERS", "users").List()
	if err != nil {
		return nil, RedisError{err.Error()}
	}
	return members, nil
}

// Returns all rates for a given ID
// Redis error otherwise
func (fa *FixrDB) GetRates(ID int) ([]string, error) {
	rates, err := fa.client.Cmd("SMEMBERS", fmt.Sprintf("rates:%d", ID)).List()
	if err != nil {
		return rates, RedisError{err.Error()}
	}
	return rates, nil
}

// Sets the Rates for given ID.
// Returns a error if something goes wrong.
func (fa *FixrDB) SetRates(ID int, rates []string) error {
	err := fa.client.Cmd("SADD", fmt.Sprintf("rates:%d", ID), rates).Err
	if err != nil {
		return RedisError{err.Error()}
	}
	return nil
}

// Removes a given rate of an ID.
// Returns an error if something goes wrong.
func (fa *FixrDB) RemoveRate(ID int, rate string) error {
	err := fa.client.Cmd("SREM", fmt.Sprintf("rates:%d", ID), rate).Err
	if err != nil {
		return RedisError{err.Error()}
	}
	return nil
}

// Sets the base for a given ID
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

// Clear all the rates of a given ID.
func (fa *FixrDB) ClearRates(ID int) error {
	err := fa.client.Cmd("DEL", fmt.Sprintf("rates:%d", ID)).Err
	if err != nil {
		return RedisError{err.Error()}
	}
	return nil
}

// Gets a property from the `user:%id` map. Returns an error if there's something wrong.
func (fa *FixrDB) GetSetting(ID int, prop string) (string, error) {
	prop, err := fa.client.Cmd("HGET", fmt.Sprintf("user:%d", ID), prop).Str()
	if err != nil {
		return "", RedisError{err.Error()}
	}
	return prop, nil
}
