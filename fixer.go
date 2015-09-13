package main

import (
	"github.com/tucnak/telebot"
	"strconv"
	"net/url"
	"strings"
	"bytes"
	"fmt"
	"net/http"
	"encoding/json"
	"sort"
)

var fixerAPI = "http://api.fixer.io/latest?"

type fixer struct {
	Base  string             `json:"base"`
	Rates map[string]float64 `json:"rates"`
}

var currencies = map[string]string {
	"EUR": "Euro",
	"AUD": "Australian Dollar",
	"BGN": "Bulgarian Lev",
	"BRL": "Brazilian Real",
	"CAD": "Canadian Dollar",
	"CHF": "Swiss Franc",
	"CNY": "Yuan Renminbi",
	"CZK": "Czech Koruna",
	"DKK": "Danish Krone",
	"GBP": "Pound Sterling",
	"HKD": "Hong Kong Dollar",
	"HRK": "Croatian Kuna",
	"HUF": "Forint",
	"IDR": "Rupiah",
	"IRL": "New Israeli Sheqel",
	"INR": "Indian Rupee",
	"JPY": "Yen",
	"KRW": "Won",
	"MXN": "Mexican Peso",
	"MYR": "Malaysian Ringgit",
	"NOK": "Norwegian Krone",
	"NZD": "New Zealand Dollar",
	"PHP": "Philippine Peso",
	"PLN": "Zloty",
	"RON": "New Romanian Leu",
	"RUB": "Russian Ruble",
	"SEK": "Swedish Krona",
	"SGD": "Singapore Dollar",
	"THB": "Baht",
	"TRY": "Turkish Lira",
	"USD": "US Dollar",
	"ZAR": "Rand",
}

func (f fixer) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Your base currency is %s\n", currencies[f.Base]))
	buffer.WriteString("These are todays rates:\n")
	for currency, value := range f.Rates {
		if name, ok := currencies[currency]; ok {
			buffer.WriteString(fmt.Sprintf("%s is %.3f\n", name, value))
		} else {
			buffer.WriteString(fmt.Sprintf("%s is %.3f\n", currency, value))
		}
	}
	return buffer.String()
}

func getIsoNames() string {
	var buf bytes.Buffer
	keys, i := make([]string, len(currencies)), 0
	for currency, _ := range currencies {
		keys[i] = currency
		i += 1
	}
	sort.Strings(keys)
	for _, key := range keys {
		buf.WriteString(key)
		buf.WriteString(", ")
		buf.WriteString(currencies[key])
		buf.WriteString("\n")
	}
	return buf.String()
}

func getFixerData(base string, rates []string) (fixer, error) {
	vals := url.Values{}
	if len(base) > 0 {
		vals.Set("base", base)
	}
	if len(rates) > 0 {
		vals.Set("symbols", strings.Join(rates, ","))
	}
	resp, err := http.Get(fixerAPI + vals.Encode())
	defer resp.Body.Close()
	if err != nil {
		return fixer{}, err
	}
	var fixerData fixer
	err = json.NewDecoder(resp.Body).Decode(&fixerData)
	if err != nil {
		return fixer{}, err
	}
	return fixerData, nil
}

func sendCurrencies(bot *telebot.Bot, fa *FixrAccessor) {
	members := fa.GetRegistered()
	for _, member := range members {
		user, _ := strconv.Atoi(member)
		base := fa.GetSetting(user, "base")
		rates := fa.GetRates(user)
		fixerData, err := getFixerData(base, rates)
		if err != nil {
			errorLogger.Println(err)
		}
		bot.SendMessage(telebot.User{ID: user}, fixerData.String(), nil)
	}
}

func sendTo(ID int, bot *telebot.Bot, fa *FixrAccessor) {
	rates := fa.GetRates(ID)
	base := fa.GetSetting(ID, "base")
	fixerData, err := getFixerData(base, rates)
	if err != nil {
		fmt.Errorf("Error querying the Fixer API: %v", err)
	}
	bot.SendMessage(telebot.User{ID: ID}, fixerData.String(), nil)
}
