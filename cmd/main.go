package main

import (
	"flag"
	"github.com/robfig/cron"
	"github.com/tucnak/telebot"
	"io"
	"log"
	"os"
	"time"
	"fixr/fixrdb"
	"fixr/fixrtelegram"
)


var (
	infoLogger  *log.Logger
	errorLogger *log.Logger
)

func startSched(bot *telebot.Bot, fa *fixrdb.FixrDB) {
	cronString := "@every 1m"
	infoLogger.Printf("Scheduling cron job for %s\n", cronString)
	sched := cron.New()
	sched.AddFunc(cronString, func() {
		infoLogger.Println("Sending currencies to all registered users")
		fixrtelegram.SendAll(bot, fa)
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
	defer recoverMain(errorLogger)
	var token = flag.String("token", "", "The bot token")
	var redis = flag.String("redis", "localhost:6379", "The address to bind to redis, e.g. \"localhost:5555\"")
	flag.Parse()
	bot, err := telebot.NewBot(*token)

	if err != nil {
		panic(err)
	}

	fixrAccessor := fixrdb.New("tcp", *redis)

	defer fixrAccessor.Close()
	infoLogger.Println("Started Redis instance")
	startSched(bot, fixrAccessor)
	messages := make(chan telebot.Message)
	bot.Listen(messages, 1*time.Second)
	for message := range messages {
		infoLogger.Printf("Got message \"%s\" from %v\n", message.Text, message.Chat)
		err := fixrtelegram.HandleAction(message, bot, fixrAccessor)
		if err != nil {
			errorLogger.Println(err)
		}
	}
}

func recoverMain(errorLogger *log.Logger) {
	if err := recover(); err != nil {
		errorLogger.Printf("PANIC on error %v", err)
	}
}
