package sms

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/luksbutz/vigilate/internal/config"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func SendTextTwilio(to, msg string, app *config.AppConfig) error {
	secret := app.PreferenceMap["twilio_auth_token"]
	key := app.PreferenceMap["twilio_sid"]
	from := app.PreferenceMap["twilio_phone_number"]

	urlString := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", key)

	msgData := url.Values{}
	msgData.Set("To", to)
	msgData.Set("From", from)
	msgData.Set("Body", msg)

	msgDataReader := *strings.NewReader(msgData.Encode())

	client := &http.Client{}
	req, _ := http.NewRequest("POST", urlString, &msgDataReader)

	req.SetBasicAuth(key, secret)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, _ := client.Do(req)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var data map[string]interface{}
		decoder := json.NewDecoder(resp.Body)

		err := decoder.Decode(&data)
		fmt.Println(data)
		if err != nil {
			log.Println(err)
			return err
		}
	} else {
		log.Println("Error sending sms", resp.Status)
		return errors.New("error sending sms")
	}

	return nil
}
