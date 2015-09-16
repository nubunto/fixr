package main

import (
	"flag"
	"github.com/robfig/cron"
	"github.com/tucnak/telebot"
	"time"
	"fmt"
	"fixr/fixrdb"
	"fixr/fixrtelegram"
	"fixr/logging"
	"os"
)

func startSched(bot *telebot.Bot, fa *fixrdb.FixrDB, cronString string) {
	logging.Info("Scheduling cron job for " + cronString)
	sched := cron.New()
	sched.AddFunc(cronString, func() {
		logging.Info("Sending currencies to all registered users")
		fixrtelegram.SendAll(bot, fa)
	})
	sched.Start()
}


func main() {
	var token = flag.String("token", "", "The bot token")
	var redis = flag.String("redis", "localhost:6379", "The address to bind to redis, e.g. \"localhost:5555\"")
	var cronString = flag.String("cron", "0 0 9 * * *", "The cron string on which to send currencies.")
	var output = flag.String("output", "", "The file where to create the log")
	flag.Parse()

	bot, err := telebot.NewBot(*token)

	if err != nil {
		panic(err)
	}

	fixrAccessor := fixrdb.New("tcp", *redis)

	defer fixrAccessor.Close()

	if len(*output) > 0 {
		output, err := os.Create(*output)
		if err != nil { panic (err) }
		logging.Start(output)
	} else {
		logging.Start(os.Stdout)
	}

	defer func() {
		logging.Info("Stopping the program")
	}()

	logging.Info("Started Redis instance")

	startSched(bot, fixrAccessor, *cronString)
	messages := make(chan telebot.Message)
	done := make(chan bool)
	bot.Listen(messages, 1*time.Second)
	go handleMessages(messages, bot, fixrAccessor, done)
	<-done
}

func handleMessages(messages chan telebot.Message, bot *telebot.Bot, fixrAccessor *fixrdb.FixrDB, done chan bool) {
	for message := range messages {
		logging.Info(fmt.Sprintf("Got message \"%s\" from %v", message.Text, message.Chat))
		err := fixrtelegram.HandleAction(message, bot, fixrAccessor)
		if err != nil {
			logging.Error(err)
		}
	}
	done <- true
}
