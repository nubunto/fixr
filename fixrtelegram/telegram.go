package fixrtelegram

import (
	"bytes"
	"fixr/fixerio"
	"fixr/fixrdb"
	"fmt"
	"github.com/tucnak/telebot"
	"fixr/logging"
	"sort"
	"strconv"
	"strings"
)

// all implemented commands.
var commands = map[string]string{
	"/setbase":    "Sets the base currency for calculations",
	"/add":        "Prompts rates and adds them to daily notifications and requests",
	"/clear":      "Clears all your rates",
	"/del [rate]": "Removes given rate from daily notifications and requests",
	"/start":      "Subscribes to daily notifications",
	"/stop":       "Unsubscribes from daily notifications",
	"/help":       "See this message",
	"/iso":        "See the ISO codes of currencies in order to select which one to pass to /setbase or /add",
	"/cancel":     "Cancel the /add command and don't change anything.",
}

// handles adding commands via a lonely /add command.
// TODO: refactor this. It's ugly.
var (
	lastCommand int = noop
	rates       []string
)

// consts that manage state of commands.
const (
	noop int = 1 << iota
	addingCommand
)

// Send a Fixer message to all registered users.
func SendAll(bot *telebot.Bot, fa *fixrdb.FixrDB) error {
	members, err := fa.GetRegistered()
	if err != nil {
		return err
	}
	for _, member := range members {
		var err error
		user, _ := strconv.Atoi(member)
		err = sendTo(user, bot, fa)
		if err != nil {
			logging.Error(err)
		}
	}
	return nil
}

// sends a fixer message to a single ID.
func sendTo(ID int, bot *telebot.Bot, fa *fixrdb.FixrDB) error {
	var err error
	rates, err := fa.GetRates(ID)
	base, err := fa.GetSetting(ID, "base")
	FixerData, err := fixerio.GetFixerData(base, rates)
	if err != nil {
		return err
	}
	bot.SendMessage(telebot.User{ID: ID}, FixerData.String(), nil)
	return nil
}

// Gets the commands as a string.
// TODO: repeating map ordering. Refactor.
func GetCommands() string {
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

// Handles the message currently received.
// TODO: refactor.
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
		if altered, err := fixrAccessor.SetBase(message.Chat.ID, base); altered {
			bot.SendMessage(message.Chat, "Base altered to "+fixerio.Currencies[base], nil)
		} else if err != nil {
			if err == fixrdb.ErrInvalidBase {
				bot.SendMessage(message.Chat, "Base \""+base+"\" is not recognized.", nil)
			}
		}
	}

	if message.Text == "/iso" {
		bot.SendMessage(message.Chat, fixerio.GetIsoNames(), nil)
	}

	if message.Text == "/clear" {
		fixrAccessor.ClearRates(message.Chat.ID)
		bot.SendMessage(message.Chat, "Cleared all rates. Showing everything now.", nil)
	}

	// if we receive a lonely /add command
	if message.Text == "/add" {
		lastCommand = addingCommand
		rates = []string{}
		bot.SendMessage(message.Chat, "Okay, let's add some currencies. Tell me which are they. You can search some of them with /iso before /add'ing them.", nil)
	} else if message.Text == "/done" || message.Text == "/cancel" {
		// if we're done, set the rates, clear commands and hide any custom keyboards.
		if len(rates) > 0 && message.Text != "/cancel" {
			fixrAccessor.SetRates(message.Chat.ID, rates)
		}

		if lastCommand != addingCommand && message.Text == "/cancel" {
			bot.SendMessage(message.Chat, "I wasn't really doing anything...", nil)
		}

		lastCommand = noop
		rates = []string{}
		bot.SendMessage(message.Chat, "Done adding rates.", &telebot.SendOptions{
			ReplyMarkup: telebot.ReplyMarkup{
				HideCustomKeyboard: true,
			},
		})
	} else if lastCommand == addingCommand {
		// if we're adding, append to the rates if it is a valid
		if validBase := fixerio.IsValidBase(message.Text); validBase {
			rates = append(rates, message.Text)
			bot.SendMessage(message.Chat, fmt.Sprintf("Rates added so far: %s\nWhen you're done adding currencies, send me /done to save them.", strings.Join(rates, ",")), nil)
		} else {
			bot.SendMessage(message.Chat, "Invalid ISO code. Please, see /iso to inspect valid currencies and add them here. Go check, I'll wait. I'll only finish this up when you send me /done", nil)
		}
	}

	// if we receive a /add command with stuff after it AND we are not adding anything,
	if msgs := strings.Split(message.Text, " "); len(msgs) > 0 && msgs[0] == "/add" && lastCommand == noop {
		// strip them off the command, and insert them.
		rates = msgs[1:]
		fixrAccessor.SetRates(message.Chat.ID, rates)
		bot.SendMessage(message.Chat, fmt.Sprintf("Added the following rates: %s", strings.Join(rates, ",")), nil)
		rates = []string{}
	}

	// if there was any error, return it.
	if err != nil {
		return err
	}

	return nil
}
