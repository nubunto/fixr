package main

import (
	"net/http"
	"fmt"
	"encoding/json"
)

type fixerData struct {
	Base string `json:"base"`
	Date string `json:"date"`
	Rates map[string]float64 `json:"rates"`
}

type telegramUser struct {
	Id int `json:"id"`
	FirstName string `json:"first_name"`
	LastName string `json:"last_name"`
	Username string `json:"username"`
}

type telegramMessage struct {
	Id int `json:"message_id"`
	From telegramUser `json:"from"`
	Text string `json:"text"`
}

type telegramUpdate struct {
	Id int `json:"update_id"`
	Message telegramMessage `json:"message"`
}

type telegramUpdates struct {
	Ok bool `json:"ok"`
	Result []telegramUpdate `json:"result"`
}

const telegramApi = "https://api.telegram.org/bot114377233:AAFGI7taTWCcI_M_jGMI9Cxev37UfshqUH0/"
const fixerApi = "http://api.fixer.io/latest"


func getTelegramUpdates() (telegramUpdates, error) {
	resp, err := http.Get(telegramApi + "getUpdates")
	if err != nil {
		return telegramUpdates{}, err
	}
	defer resp.Body.Close()
	var updates telegramUpdates
	if err := json.NewDecoder(resp.Body).Decode(&updates); err != nil {
		return telegramUpdates{}, err
	}
	return updates, nil
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello world")
	})
	http.ListenAndServe(":8080", nil)
}
