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
	"regexp"
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

type commandHandler func(*telebot.Bot, telebot.Message, *fixrdb.FixrDB) error

var commandDispatcher = map[string]commandHandler{
	"/setbase": func(bot *telebot.Bot, message telebot.Message, fixrAccessor *fixrdb.FixrDB) error {
			var err error
			base := message.Text[len("/setbase")+1:]
			if altered, err := fixrAccessor.SetBase(message.Chat.ID, base); altered {
				bot.SendMessage(message.Chat, "Base altered to "+fixerio.Currencies[base], nil)
			} else if err != nil {
				if err == fixrdb.ErrInvalidBase {
					bot.SendMessage(message.Chat, "Base \""+base+"\" is not recognized.", nil)
				}
			}
			if err != nil { return err }
			return nil
		},
	"/start": func(bot *telebot.Bot, message telebot.Message, fixrAccessor *fixrdb.FixrDB) error {
		var err error
		if subscribed, err := fixrAccessor.Subscribe(message.Chat.ID); err != nil && !subscribed {
			bot.SendMessage(message.Chat, "You are already subscribed.", nil)
		} else {
			bot.SendMessage(message.Chat, "You were subscribed to daily notifications. Send /stop if you don't want that.", nil)
		}
		if err != nil { return err }
		return nil
	},
	"/stop": func(bot *telebot.Bot, message telebot.Message, fixrAccessor *fixrdb.FixrDB) error {
		var err error
		if unsubscribed, err := fixrAccessor.Unsubscribe(message.Chat.ID); err != nil && !unsubscribed {
			bot.SendMessage(message.Chat, "You are already unsubscribed.", nil)
		} else {
			bot.SendMessage(message.Chat, "You were unsubscribed from daily notifications. Send /start to subscribe again", nil)
		}
		if err != nil { return err }
		return nil
	},
	"/get": func(bot *telebot.Bot, message telebot.Message, fixrAccessor *fixrdb.FixrDB) error {
		return sendTo(message.Chat.ID, bot, fixrAccessor)
	},
	"/del": func(bot *telebot.Bot, message telebot.Message, fixrAccessor *fixrdb.FixrDB) error {
		msgs := strings.Split(message.Text, " ")
		fixrAccessor.RemoveRate(message.Chat.ID, msgs[1])
		bot.SendMessage(message.Chat, fmt.Sprintf("Removed currency %s", msgs[1]), nil)
		return nil
	},
	"/iso": func(bot *telebot.Bot, message telebot.Message, fixrAccessor *fixrdb.FixrDB) error {
		bot.SendMessage(message.Chat, fixerio.GetIsoNames(), nil)
		return nil
	},
	"/clear": func(bot *telebot.Bot, message telebot.Message, fixrAccessor *fixrdb.FixrDB) error {
		var err error 
		err = fixrAccessor.ClearRates(message.Chat.ID)
		bot.SendMessage(message.Chat, "Cleared all rates. Showing everything now.", nil)
		if err != nil { return err }
		return nil
	},
	"/help": func(bot *telebot.Bot, message telebot.Message, fixrAccessor *fixrdb.FixrDB) error {
		bot.SendMessage(message.Chat, GetCommands(), nil)
		return nil
	},
	"/add": func(bot *telebot.Bot, message telebot.Message, fixrAccessor *fixrdb.FixrDB) error {
		var err error
		if msgs := strings.Split(message.Text, " "); len(msgs) > 1 && !addingHandler.active {
			err = fixrAccessor.SetRates(message.Chat.ID, msgs[1:])
			bot.SendMessage(message.Chat, fmt.Sprintf("Added the following rates: %s", strings.Join(addingHandler.rates, ",")), nil)
		}
		addingHandler.active = true
		bot.SendMessage(message.Chat, "Okay, let's add some currencies. Tell me which are they. You can search some of them with /iso before /add'ing them.", nil)
		if err != nil { return err }
		return nil
	},

}

type addingCommand struct {
	rates []string
	active bool
}

var addingHandler *addingCommand

func (ah *addingCommand) Do(bot *telebot.Bot, message telebot.Message, db *fixrdb.FixrDB) error {
	var err error
	msg := message.Text
	if ah.active {
		if msg == "/done" {
			err = db.SetRates(message.Chat.ID, ah.rates)
			bot.SendMessage(message.Chat, "Done adding rates.", nil)
			ah.active = false
		} else if msg == "/cancel" {
			ah.rates = []string{}
			ah.active = false
			bot.SendMessage(message.Chat, "Bailing out. Didn't add any rates.", nil)
		} else {
			if validBase := fixerio.IsValidBase(message.Text); validBase {
				ah.rates = append(ah.rates, message.Text)
				bot.SendMessage(message.Chat, fmt.Sprintf("Rates added so far: %s\nWhen you're done adding currencies, send me /done to save them.", strings.Join(ah.rates, ",")), nil)
			} else {
				bot.SendMessage(message.Chat, "Invalid ISO code. Please, see /iso to inspect valid currencies and add them here. Go check, I'll wait. I'll only finish this up when you send me /done", nil)
			}
		}
	} else {
		bot.SendMessage(message.Chat, "Okay, let's add some currencies. Tell me which are they. You can search some of them with /iso before /add'ing them.", nil)
		ah.active = true
	}
	if err != nil { return err }
	return nil
}

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
// TODO: repeating map ordering in somewhere else. Refactor.
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
func HandleAction(message telebot.Message, bot *telebot.Bot, fixrAccessor *fixrdb.FixrDB) error {
	var err error
	if addingHandler == nil {
		addingHandler = &addingCommand{make([]string, 0), false}
	}
	if addingHandler.active {
		addingHandler.Do(bot, message, fixrAccessor)
	} else {
		re, _ := regexp.Compile("/[a-z]+") 
		command := re.FindString(message.Text)
		if fn, ok := commandDispatcher[command]; ok {
			err = fn(bot, message, fixrAccessor)
		} else {
			commandDispatcher["/help"](bot, message, fixrAccessor)
		}
	}
	if err != nil { return err }
	return nil
}