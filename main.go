package main

import (
	"time"
	"github.com/tucnak/telebot"
	"github.com/mediocregopher/radix.v2/redis"
	"fmt"
	"flag"
)

type command struct {
	Message string
	Markup telebot.ReplyMarkup
}

func (c command) Send(message telebot.Message, bot *telebot.Bot) {
	bot.SendMessage(message.Chat, c.Message, &telebot.SendOptions{
		ReplyMarkup: c.Markup,
	})
}

func NewCommand(message string, keyboard [][]string) *command {
	return &command {
		Message: message,
		Markup: telebot.ReplyMarkup{ OneTimeKeyboard: true, CustomKeyboard: keyboard, ResizeKeyboard: true, },
	}
}



func main() {
	flag.Parse()
	var token = flag.String("token", "The bot token", "")
	bot, err := telebot.NewBot(*token)
	if err != nil {
		fmt.Errorf("somethings wrong, goodbye. %v", err)
	}
	redis_cli, err := redis.Dial("tcp", "localhost:6379")
	if err != nil {
		fmt.Errorf("redis_cli died.")
	}
	messages := make(chan telebot.Message)
	bot.Listen(messages, 1 * time.Second)
	for message := range messages {
		if message.Text == "/start" {
			handleSubscription(message, bot, redis_cli)
		}
	}
}

func handleSubscription(message telebot.Message, bot *telebot.Bot, redis_cli *redis.Client) {
	isSubscribed, err := redis_cli.Cmd("SISMEMBER", "registered_users", message.Sender.ID).Int()
	if err != nil {
		fmt.Errorf("somethings wrong, goodbye. %v", err)
	}
	if isSubscribed == 1 {
		NewCommand("You are already subscribed", nil).Send(message, bot)
	} else {
		next_conn, err := redis_cli.Cmd("INCR", "user_id").Int()
		if err != nil {
			fmt.Errorf("somethings wrong, goodbye. %v", err)
		}
		err = redis_cli.Cmd("SADD", "registered_users", message.Sender.ID).Err
		if err != nil {
			fmt.Errorf("somethings wrong, goodbye. %v", err)
		}
		err = redis_cli.Cmd("HMSET", fmt.Sprintf("users:%d", next_conn), "username", message.Sender.Username, "id", message.Sender.ID).Err
		if err != nil {
			fmt.Errorf("somethings wrong, goodbye. %v", err)
		}
		NewCommand("Welcome aboard, " + message.Sender.FirstName + "! We're happy to have you here.", nil).Send(message, bot)
	}
}
