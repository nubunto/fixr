package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/mediocregopher/radix.v2/redis"
	"github.com/robfig/cron"
	"github.com/tucnak/telebot"
	"net/http"
	"strconv"
	"time"
)

var fixerAPI = "http://api.fixer.io/latest"

type fixer struct {
	Base  string             `json:"base"`
	Rates map[string]float64 `json:"rates"`
}

func (f fixer) String() string {
	s := fmt.Sprintf("Your base currency is %s\n", f.Base)
	s += "These are todays rates:\n"
	for currency, value := range f.Rates {
		s += fmt.Sprintf("Currency: %s, value: %.3f\n", currency, value)
	}
	return s
}

func getFixerData(base string) (fixer, error) {
	if len(base) == 0 {
		base = "EUR"
	}
	resp, err := http.Get(fixerAPI + "?base=" + base)
	defer resp.Body.Close()
	if err != nil {
		return fixer{}, err
	}
	var fixerData fixer
	err = json.NewDecoder(resp.Body).Decode(&fixerData)
	if err != nil {
		return fixer{}, err
	}
	return fixerData, nil
}

func sendCurrencies(bot *telebot.Bot) {
	client, err := redis.Dial("tcp", "localhost:6379")
	defer client.Close()
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	members, err := client.Cmd("SMEMBERS", "users").Array()
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	for _, member := range members {
		id, err := member.Str()
		user, _ := strconv.Atoi(id)
		if err != nil {
			fmt.Printf("error: %v", err)
		}
		base, _ := client.Cmd("HGET", fmt.Sprintf("users:%d", user), "base").Str()
		fixerData, err := getFixerData(base)
		if err != nil {
			fmt.Printf("error: %v", err)
		}
		bot.SendMessage(telebot.User{ID: user}, fixerData.String(), nil)
	}
}

func sendTo(ID int, base string, bot *telebot.Bot) {
	fixerData, err := getFixerData(base)
	if err != nil {
		fmt.Errorf("Error querying the Fixer API: %v", err)
	}
	bot.SendMessage(telebot.User{ID: ID}, fixerData.String(), nil)
}

func startSched(bot *telebot.Bot) {
	sched := cron.New()
	sched.AddFunc("@every 1m", func() {
		sendCurrencies(bot)
	})
	sched.Start()
}

func main() {
	var token = flag.String("token", "The bot token", "")
	flag.Parse()
	bot, err := telebot.NewBot(*token)
	if err != nil {
		fmt.Errorf("somethings wrong, goodbye. %v", err)
	}
	redis_cli, err := redis.Dial("tcp", "localhost:6379")
	defer redis_cli.Close()
	if err != nil {
		fmt.Errorf("redis_cli died.")
	}
	startSched(bot)
	messages := make(chan telebot.Message)
	bot.Listen(messages, 1*time.Second)
	for message := range messages {
		if message.Text == "/start" || message.Text == "/subscribe" {
			handleSubscription(message.Sender.ID, message.Sender.FirstName, bot, redis_cli)
		}
		if message.Text == "/unsubscribe" {
			handleUnsubscription(message.Sender.ID, bot, redis_cli)
		}
		if message.Text == "/currencies" {
			sendTo(message.Sender.ID, "USD", bot)
		}
		if message.Text == "/set" {
			// TODO: The user can select custom currencies. This involves a little more design on the DB part.
		}
	}
}

func handleSubscription(ID int, firstName string, bot *telebot.Bot, redis_cli *redis.Client) {
	subscribed, err := isSubscribed(ID, redis_cli)
	user := telebot.User{ID: ID}
	if err != nil {
		fmt.Errorf("somethings wrong, goodbye. %v", err)
	}
	if subscribed {
		bot.SendMessage(user, "You are already subscribed.", nil)
	} else {
		err := subscribe(ID, redis_cli)
		if err != nil {
			fmt.Printf("Something's wrong: %v", err)
		}
		bot.SendMessage(user, "Welcome aboard, "+firstName+"! We're glad to have you here.", nil)
	}
}

func handleUnsubscription(ID int, bot *telebot.Bot, redis_cli *redis.Client) {
	subscribed, err := isSubscribed(ID, redis_cli)
	user := telebot.User{ID: ID}
	if err != nil {
		fmt.Errorf("Error on unsubscription: %v", err)
	}
	if !subscribed {
		bot.SendMessage(user, "You aren't subscribed.", nil)
	} else {
		unsubscribed, err := unsubscribe(ID, redis_cli)
		if unsubscribed {
			bot.SendMessage(user, "You are now no longer subscribed. Send /subscribe if you want to get back on notifications.", nil)
		} else if err != nil {
			fmt.Errorf("Got error: %v", err)
		}
	}
}

func isSubscribed(ID int, redis_cli *redis.Client) (bool, error) {
	isSubscribed, err := redis_cli.Cmd("SISMEMBER", "users", ID).Int()
	if err != nil {
		return false, err
	}
	return isSubscribed == 1, nil
}

func subscribe(ID int, redis_cli *redis.Client) error {
	err := redis_cli.Cmd("SADD", "users", ID).Err
	if err != nil {
		return err
	}
	err = redis_cli.Cmd("HMSET", fmt.Sprintf("users:%d", ID), "base", "USD").Err
	if err != nil {
		return err
	}
	return nil
}

func unsubscribe(ID int, redis_cli *redis.Client) (bool, error) {
	err := redis_cli.Cmd("SREM", "users", ID).Err
	if err != nil {
		return false, err
	}
	return true, nil
}
