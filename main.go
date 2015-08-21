package main

import (
	"time"
	"github.com/tucnak/telebot"
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
		Markup: telebot.ReplyMarkup{ CustomKeyboard: keyboard },
	}
}

var token = flag.String("token", "The bot token", "")

func main() {
	flag.Parse()
	bot, err := telebot.NewBot(*token)
	if err != nil {
		fmt.Printf("Error: %v", err)
		return
	}

	messages := make(chan telebot.Message)
	bot.Listen(messages, 1 * time.Second)

	for message := range messages {
		if message.Text == "/hi" {
			c := NewCommand("Hello, " + message.Sender.FirstName, [][]string{ { "fuck", "you" } })
			c.Send(message, bot)
		}
	}
}

