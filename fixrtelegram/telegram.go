package fixrtelegram

import (
	"bytes"
	"sort"
	"fmt"
	"strings"
	"strconv"
	"fixr/fixerio"
	"fixr/fixrdb"
	"github.com/tucnak/telebot"
)

var commands = map[string]string{
	"/setbase":                       "Sets the base currency for calculations",
	"/add": "Prompts rates and adds them to daily notifications and requests",
	"/clear":      "Clears all your rates",
	"/del [rate]": "Removes given rate from daily notifications and requests",
	"/start":      "Subscribes to daily notifications",
	"/stop":       "Unsubscribes from daily notifications",
	"/help":       "See this message",
	"/iso":        "See the ISO codes of currencies in order to select which one to pass to /setbase or /add",
}

var (
	lastCommand int = noop
	rates []string
)

const (
	noop int = 1 << iota
	addingCommand
)


func SendAll(bot *telebot.Bot, fa *fixrdb.FixrDB) error {
	members := fa.GetRegistered()
	for _, member := range members {
		user, _ := strconv.Atoi(member)
		base := fa.GetSetting(user, "base")
		rates := fa.GetRates(user)
		FixerData, err := fixerio.GetFixerData(base, rates)
		if err != nil {
			return err
		}
		bot.SendMessage(telebot.User{ID: user}, FixerData.String(), nil)
	}
	return nil
}

func sendTo(ID int, bot *telebot.Bot, fa *fixrdb.FixrDB) error {
	rates := fa.GetRates(ID)
	base := fa.GetSetting(ID, "base")
	FixerData, err := fixerio.GetFixerData(base, rates)
	if err != nil {
		return err
	}
	bot.SendMessage(telebot.User{ID: ID}, FixerData.String(), nil)
	return nil
}

func GetCommands() string {
	// go doesn't guarantee map ordering....
	var buf bytes.Buffer
	buf.WriteString("Hello! Welcome to FixrBot. This bot aims to help you with currency values from around the world.\nHere are my features:\n\n")
	keys, i := make([]string, len(commands)), 0
	for command, _ := range commands {
		keys[i] = command
		i += 1
	}
	sort.Strings(keys)
	for _, key := range keys {
		buf.WriteString(key)
		buf.WriteString(" - ")
		buf.WriteString(commands[key])
		buf.WriteString("\n")
	}
	return buf.String()
}

func HandleAction(message telebot.Message, bot *telebot.Bot, fixrAccessor *fixrdb.FixrDB) error {
	var err error
	if message.Text == "/start" {
		if subscribed, err := fixrAccessor.Subscribe(message.Chat.ID); err != nil && !subscribed {
			bot.SendMessage(message.Chat, "You are already subscribed.", nil)
		} else {
			bot.SendMessage(message.Chat, "You were subscribed to daily notifications. Send /stop if you don't want that.", nil)
		}
	}

	if message.Text == "/stop" {
		if unsubscribed, err := fixrAccessor.Unsubscribe(message.Chat.ID); err != nil && !unsubscribed {
			bot.SendMessage(message.Chat, "You are already unsubscribed.", nil)
		} else {
			bot.SendMessage(message.Chat, "You were unsubscribed from daily notifications. Send /start to subscribe again", nil)
		}
	}

	if message.Text == "/get" {
		err = sendTo(message.Chat.ID, bot, fixrAccessor)
	}

	if msgs := strings.Split(message.Text, " "); len(msgs) == 2 && msgs[0] == "/del" {
		fixrAccessor.RemoveRate(message.Chat.ID, msgs[1])
		bot.SendMessage(message.Chat, fmt.Sprintf("Removed currency %s", msgs[1]), nil)
	}

	if message.Text == "/help" {
		bot.SendMessage(message.Chat, GetCommands(), nil)
	}

	if len(message.Text) > len("/setbase") && message.Text[:len("/setbase")] == "/setbase" {
		base := message.Text[len("/setbase")+1:]
		if altered := fixrAccessor.SetBase(message.Chat.ID, base); altered {
			bot.SendMessage(message.Chat, "Base altered to "+fixerio.Currencies[base], nil)
		} else {
			bot.SendMessage(message.Chat, "Base \""+base+"\" is not recognized.", nil)
		}
	}

	if message.Text == "/iso" {
		bot.SendMessage(message.Chat, fixerio.GetIsoNames(), nil)
	}

	if message.Text == "/clear" {
		fixrAccessor.ClearRates(message.Chat.ID)
		bot.SendMessage(message.Chat, "Cleared all rates. Showing everything now.", nil)
	}


	if message.Text == "/add" {
		lastCommand = addingCommand
		rates = []string{}
		bot.SendMessage(message.Chat, "Okay, let's add some currencies. Tell me which are they. You can search some of them with /iso before /add'ing them.", nil)
	} else if message.Text == "/done" {
		fmt.Printf("what")
		if len(rates) > 0 {
			fixrAccessor.SetRates(message.Chat.ID, rates)
		}
		lastCommand = noop
		rates = []string{}
		bot.SendMessage(message.Chat, "Done adding rates.", &telebot.SendOptions{
			ReplyMarkup: telebot.ReplyMarkup{
				HideCustomKeyboard: true,
			},
		})
	} else if lastCommand == addingCommand {
		if validBase := fixerio.IsValidBase(message.Text); validBase {
			rates = append(rates, message.Text)
			bot.SendMessage(message.Chat, fmt.Sprintf("Rates added so far: %s", strings.Join(rates, ",")), nil)
		} else {
			bot.SendMessage(message.Chat, "Invalid ISO code. Please, see /iso to inspect valid currencies and add them here.", nil)
		}
	}
	if msgs := strings.Split(message.Text, " "); len(msgs) > 0 && msgs[0] == "/add" && lastCommand == noop {
		rates = msgs[1:]
		fixrAccessor.SetRates(message.Chat.ID, rates)
		bot.SendMessage(message.Chat, fmt.Sprintf("Added the following rates: %s", strings.Join(rates, ",")), nil)
		rates = []string{}
	}
	if err != nil {
		return err
	}
	return nil
}
