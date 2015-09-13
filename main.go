package main

import (
	"flag"
	"github.com/robfig/cron"
	"github.com/tucnak/telebot"
	"io"
	"log"
	"os"
	"strings"
	"time"
	"bytes"
	"sort"
)

var (
	infoLogger *log.Logger
	errorLogger *log.Logger
)

var commands = map[string]string {
	"/setbase": "Sets the base currency for calculations",
	"/add [rates] (comma separated)": "Adds a given rate to daily notifications and requests",
	"/clear": "Clears all your rates",
	"/del [rate]": "Removes given rate from daily notifications and requests",
	"/start": "Subscribes to daily notifications",
	"/stop": "Unsubscribes from daily notifications",
	"/help": "See this message",
	"/iso": "See the ISO codes of currencies in order to select which one to pass to /setbase or /add",
}

func showHelp() string {
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

func startSched(bot *telebot.Bot, fa *FixrAccessor) {
	cronString := "@every 1m"
	infoLogger.Printf("Scheduling cron job for %s\n", cronString)
	sched := cron.New()
	sched.AddFunc(cronString, func() {
		sendCurrencies(bot, fa)
	})
	sched.Start()
}

func createLoggers(info, err io.Writer) (*log.Logger, *log.Logger) {
	logType := log.Ldate | log.Ltime | log.Lshortfile
	return log.New(info, "INFO: ", logType), log.New(err, "ERROR :", logType)
}

func init() {
	infoLogger, errorLogger = createLoggers(os.Stdout, os.Stdout)
}

func main() {
	var token = flag.String("token", "The bot token", "")
	var redis = flag.String("redis", "The transport and address to bind to redis, comma separated", "tcp,localhost:6379")
	flag.Parse()
	bot, err := telebot.NewBot(*token)

	if err != nil {
		panic(err)
	}

	redisConfig := strings.Split(*redis, ",")
	transport, address := redisConfig[0], redisConfig[1]

	fixrAccessor := NewBackend(transport, address);

	defer fixrAccessor.Close()
	infoLogger.Println("Started Redis instance")
	startSched(bot, fixrAccessor)
	messages := make(chan telebot.Message)
	bot.Listen(messages, 1*time.Second)
	defer recoverMain(errorLogger)
	for message := range messages {
		infoLogger.Printf("Got %s message from %v\n", message.Text, message.Chat)
		if message.Text == "/start" {
			bot.SendMessage(message.Chat, showHelp(), nil)
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
			sendTo(message.Chat.ID, bot, fixrAccessor)
		}
		if message.Text == "/help" {
			bot.SendMessage(message.Chat, showHelp(), nil)
		}
		if len(message.Text) > len("/setbase") && message.Text[:len("/setbase")] == "/setbase" {
			base := message.Text[len("/setbase") + 1:]
			if altered := fixrAccessor.SetBase(message.Chat.ID, base); altered {
				bot.SendMessage(message.Chat, "Base altered to " + currencies[base], nil)
			} else {
				bot.SendMessage(message.Chat, "Base \"" + base + "\" is not recognized.", nil)
			}
		}
		if message.Text == "/iso" {
			bot.SendMessage(message.Chat, getIsoNames(), nil) 
		}
	}
}

func recoverMain(errorLogger *log.Logger) {
	if err := recover(); err != nil {
		errorLogger.Printf("PANIC on error %v", err)
	}
}
