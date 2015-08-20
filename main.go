package main

import (
	"time"
	"github.com/tucnak/telebot"
)

func main() {
	bot, err := telebot.NewBot("114377233:AAGi4pyLGYInLyJOabQUIsFyeV8NM0As42E");
	if err != nil {
		return
	}

	messages := make(chan telebot.Message)
	bot.Listen(messages, 1 * time.Second)

	for message := range messages {
		if message.Text == "/hi" {
			bot.SendMessage(message.Chat, "Hello, " + message.Sender.FirstName + "!", &telebot.SendOptions{
				ReplyMarkup: telebot.ReplyMarkup{
					CustomKeyboard: [][]string{ {"how are you", "I'm fine"}, },
				},
			})
		}
	}
}
