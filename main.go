package main

import (
	"fmt"
	"net/http"
	"github.com/nubunto/fixr/tg"
)

func main() {
	bot := tg.NewBot("114377233:AAFGI7taTWCcI_M_jGMI9Cxev37UfshqUH0")
	
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello world")
		
		myBot, _ := bot.GetMe()
		
		fmt.Printf("%v", myBot)
	})
	
	http.ListenAndServe(":8080", nil)
}
