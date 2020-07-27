// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"html/template"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gobuffalo/envy"
	log "github.com/sirupsen/logrus"
)

var clientID string
var clientSecret string
var callbackURL string
var token string
var mqttClient mqtt.Client
var MQTT_BROKER_URL = envy.Get("MQTT_BROKER_URL", "mqtt://user1:password1@blynk.bstiot.com:1883")

func main() {
	mqttClientID := "9a49ea0e-b2b3-4aac-b2bd-35a0bc095a1b"
	mqttURI, err := url.Parse(MQTT_BROKER_URL)
	if err != nil {
		log.Panic(err)
	} else {
		mqttClient, err = MQTTconnect(mqttClientID, mqttURI)
		if err != nil {
			log.Panic(err)
		}
	}
	go mqttSubscribePLCPayload("cmd/TEST1/Group1/1234")

	http.HandleFunc("/callback", callbackHandler)
	http.HandleFunc("/notify", notifyHandler)
	http.HandleFunc("/auth", authHandler)
	clientID = os.Getenv("ClientID")
	clientSecret = os.Getenv("ClientSecret")
	callbackURL = os.Getenv("CallbackURL")
	port := os.Getenv("PORT")
	fmt.Printf("ENV port:%s, cid:%s csecret:%s\n", port, clientID, clientSecret)
	addr := fmt.Sprintf(":%s", port)
	http.ListenAndServe(addr, nil)
}

func notifyHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() // Populates request.Form
	msg := r.Form.Get("msg")
	fmt.Printf("Get msg=%s\n", msg)

	data := url.Values{}
	data.Add("message", msg)

	byt, err := apiCall("POST", apiNotify, data, token)
	fmt.Println("ret:", string(byt), " err:", err)

	res := newTokenResponse(byt)
	fmt.Println("result:", res)
	token = res.AccessToken
	w.Write(byt)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() // Populates request.Form
	code := r.Form.Get("code")
	state := r.Form.Get("state")
	fmt.Printf("Get code=%s, state=%s \n", code, state)

	data := url.Values{}
	data.Add("grant_type", "authorization_code")
	data.Add("code", code)
	data.Add("redirect_uri", callbackURL)
	data.Add("client_id", clientID)
	data.Add("client_secret", clientSecret)

	byt, err := apiCall("POST", apiToken, data, "")
	fmt.Println("ret:", string(byt), " err:", err)

	res := newTokenResponse(byt)
	fmt.Println("result:", res)
	token = res.AccessToken
	w.Write(byt)
}
func authHandler(w http.ResponseWriter, r *http.Request) {
	check := func(err error) {
		if err != nil {
			log.Fatal(err)
		}
	}
	t, err := template.New("webpage").Parse(authTmpl)
	check(err)
	noItems := struct {
		ClientID    string
		CallbackURL string
	}{
		ClientID:    clientID,
		CallbackURL: callbackURL,
	}

	err = t.Execute(w, noItems)
	check(err)
}

func MQTTconnect(clientId string, uri *url.URL) (mqtt.Client, error) {
	opts := MQTTcreateClientOptions(clientId, uri)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	for !token.WaitTimeout(3 * time.Second) {

	}
	if err := token.Error(); err != nil {
		log.Fatal(err)
		return client, err
	}
	log.Info("MQTT Connected.")
	return client, nil
}

func MQTTcreateClientOptions(clientId string, uri *url.URL) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", uri.Host))
	opts.SetCleanSession(false)
	opts.SetAutoReconnect(true)
	if len(uri.User.Username()) > 0 {
		opts.SetUsername(uri.User.Username())

		password, ok := uri.User.Password()
		if ok {
			opts.SetPassword(password)
		}
	}
	opts.SetClientID(clientId)
	return opts
}

func mqttSubscribePLCPayload(topic string) {
	log.Infof("MQTT subscribed topic '%s'.", topic)
	mqttClient.Subscribe(topic, 1, func(client mqtt.Client, msg mqtt.Message) {
		log.Infof("* [%s] %s\n", msg.Topic(), string(msg.Payload()))
		// err := json.Unmarshal(msg.Payload(), &boxEvent)
		// if err != nil {
		// 	log.Error(err)
		// 	return
		// }
		data := url.Values{}
		data.Add("message", string(msg.Payload()))

		byt, err := apiCall("POST", apiNotify, data, token)
		log.Println("ret:", string(byt), " err:", err)

		res := newTokenResponse(byt)
		log.Println("result:", res)
		token = res.AccessToken
	})
}
