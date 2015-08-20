package tg

import (
	"net/http"
	"encoding/json"
	"fmt"
)

const telegramAPI = "https://api.telegram.org/bot%s/%s"

type TelegramBot struct {
	token string // the bot token of the API
}

func NewBot(token string) *TelegramBot {
	return &TelegramBot{token}
}

func (bot TelegramBot) GetUpdates() (TelegramUpdateResponse, error) {
	resp, err := http.Get(fmt.Sprintf(telegramAPI, bot.token, "getUpdates"))
	if err != nil {
		return TelegramUpdateResponse{}, err
	}
	defer resp.Body.Close()
	var updates TelegramUpdateResponse
	if err := json.NewDecoder(resp.Body).Decode(&updates); err != nil {
		return TelegramUpdateResponse{}, err
	}
	return updates, nil
}

func (bot TelegramBot) GetMe() (TelegramUserResponse, error) {
	resp, err := http.Get(fmt.Sprintf(telegramAPI, bot.token, "getMe"))
	if err != nil {
		return TelegramUserResponse{}, err
	}
	defer resp.Body.Close()
	var bot TelegramUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&bot); err != nil {
		return TelegramUserResponse{}, err
	}
	return bot, nil
}