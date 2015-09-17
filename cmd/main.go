package main

import (
	"fixr/fixrdb"
	"fixr/fixrtelegram"
	"fixr/logging"
	"flag"
	"fmt"
	"github.com/robfig/cron"
	"github.com/tucnak/telebot"
	"os"
	"time"
)

// startSched starts the cron on which to send currencies every day.
func startSched(bot *telebot.Bot, fa *fixrdb.FixrDB, cronString string) {
	logging.Info("Scheduling cron job for " + cronString)
	sched := cron.New()
	sched.AddFunc(cronString, func() {
		logging.Info("Sending currencies to all registered users")
		// send to all registered users.
		fixrtelegram.SendAll(bot, fa)
	})
	sched.Start()
}

func main() {
	// the token on which telebot will send/receive stuff
	var token = flag.String("token", "", "The bot token")

	// the address and port on which redis will listen
	var redis = flag.String("redis", "localhost:6379", "The address to bind to redis, e.g. \"localhost:5555\"")

	// the cron string on which to send all the currencies (why I did this?)
	var cronString = flag.String("cron", "0 0 9 * * *", "The cron string on which to send currencies.")

	// the filename of the log when on the server (this is good)
	var output = flag.String("output", "", "The file where to create the log")

	flag.Parse()

	bot, err := telebot.NewBot(*token)

	// if telegram is not there, bail.
	if err != nil {
		panic(err)
	}

	fixrAccessor := fixrdb.New("tcp", *redis)

	defer fixrAccessor.Close()

	// if there's a file, create it and use it. Else, assume debugging: pipe to stdout.
	if len(*output) > 0 {
		output, err := os.Create(*output)
		if err != nil {
			panic(err)
		}
		logging.Start(output)
	} else {
		logging.Start(os.Stdout)
	}

	// this doesn't actually work... but whatever.
	defer func() {
		logging.Info("Stopping the program")
	}()

	logging.Info("Started Redis instance")

	// start the scheduler
	startSched(bot, fixrAccessor, *cronString)
	messages := make(chan telebot.Message)
	done := make(chan bool)

	// listen to updates
	bot.Listen(messages, 1*time.Second)

	// handle messages on a different goroutine, so we don't mess this up.
	go handleMessages(messages, bot, fixrAccessor, done)

	// wait for the inevitable end (that never comes)
	<-done
}

func handleMessages(messages chan telebot.Message, bot *telebot.Bot, fixrAccessor *fixrdb.FixrDB, done chan bool) {

	// range over the channel of messages, handle the message sent, log any error.
	for message := range messages {
		logging.Info(fmt.Sprintf("Got message \"%s\" from %v", message.Text, message.Chat))
		err := fixrtelegram.HandleAction(message, bot, fixrAccessor)
		if err != nil {
			logging.Error(err)
		}
	}

	// we're done, notify other goroutine
	done <- true
}