package fixerio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// The url of the Fixer API we're issuing
var fixerAPI = "http://api.fixer.io/latest?"

// The response of the Fixer API
type Fixer struct {
	// Base is the currency base for calculations
	Base string `json:"base"`

	// Rates are the values, along with the ISO codes of each currency, and it's value, giving the base.
	Rates map[string]float64 `json:"rates"`
}

// By lack of a public documentation, I got these out from a request to Fixer.
// Thse are all (I hope) the valid ISO codes of currencies Fixer supports.
var Currencies = map[string]string{
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

// Implement Stringer, so we can print it to messages in telegram
func (f Fixer) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Your base currency is %s\n", Currencies[f.Base]))
	buffer.WriteString("These are todays rates:\n")
	for currency, value := range f.Rates {
		if name, ok := Currencies[currency]; ok {
			buffer.WriteString(fmt.Sprintf("%s is %.3f\n", name, value))
		} else {
			buffer.WriteString(fmt.Sprintf("%s is %.3f\n", currency, value))
		}
	}
	return buffer.String()
}

// Get's the names of ISO codes, and sorts them alphabetically
func GetIsoNames() string {
	var buf bytes.Buffer
	keys, i := make([]string, len(Currencies)), 0
	for currency, _ := range Currencies {
		keys[i] = currency
		i += 1
	}
	sort.Strings(keys)
	for _, key := range keys {
		buf.WriteString(key)
		buf.WriteString(", ")
		buf.WriteString(Currencies[key])
		buf.WriteString("\n")
	}
	return buf.String()
}

// Get's the ISO codes as a string
func GetIsoCodes() []string {
	keys, i := make([]string, len(Currencies)), 0
	for code, _ := range Currencies {
		keys[i] = code
		i += 1
	}
	return keys
}

// See's if Fixer can handle this base.
func IsValidBase(base string) bool {
	_, ok := Currencies[base]
	return ok
}

// Returns an JSON object from the Fixer API.
func GetFixerData(base string, rates []string) (Fixer, error) {
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
		return Fixer{}, err
	}
	var FixerData Fixer
	err = json.NewDecoder(resp.Body).Decode(&FixerData)
	if err != nil {
		return Fixer{}, err
	}
	return FixerData, nil
}
