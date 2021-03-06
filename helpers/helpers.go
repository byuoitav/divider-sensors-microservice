package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/byuoitav/central-event-system/messenger"
	"github.com/byuoitav/common/log"

	"net/http"
	"sync"

	"github.com/byuoitav/common/v2/events"
)

// CONNECTED represents the signal sent by the sensors when the rooms are connected.
const CONNECTED = 1

// DISCONNECTED represents the signal sent by the sensors when the rooms are disconnected.
const DISCONNECTED = 0

// Messenger is the EventNode object used to publish events.
var Messenger *messenger.Messenger

// DC is the divider config for this pi
var DC DividerConfig

// StartReading sets up which pins to read from, and begins reading.
func StartReading(wg *sync.WaitGroup) {
	var err error
	DC, err = ReadConfig()
	pinList := DC.Pins
	if err != nil {
		log.L.Error("Ah dang, I couldn't get the pins...")
		return
	}

	log.L.Debugf("Divider Configuration: %+v", DC)

	wg.Add(len(pinList))
	for i := range pinList {
		go readSensors(pinList[i], wg)
	}
}

// Connect processes all changes that need to happen when the rooms are connected.
func Connect(p Pin) {
	log.L.Infof("Sensors connected")
	go ConnectedEvent(p)
	go DSPChange(p, CONNECTED)

	for _, req := range DC.Connect {
		go MakeRequest(req)
	}

	for _, event := range DC.ConnectEvents {
		go SendEvent(event)
	}
}

// Disconnect processes all changes that need to happen when the rooms are disconnected.
func Disconnect(p Pin) {
	log.L.Infof("Sensors disconnected")
	go DisconnectedEvent(p)
	go DSPChange(p, DISCONNECTED)

	for _, req := range DC.Disconnect {
		go MakeRequest(req)
	}

	for _, event := range DC.DisconnectEvents {
		go SendEvent(event)
	}
}

// MakeRequest makes a request, WHAAAA????
func MakeRequest(r Request) error {
	log.L.Infof("Making request: %+s", r)

	client := &http.Client{}
	url := fmt.Sprintf("http://%v:%v/%v/", r.Host, r.Port, r.Endpoint)

	body, err := json.Marshal(r.Body)
	if err != nil {
		log.L.Errorf("Failed to marshal body. ERROR: %s", err.Error())
		return err
	}

	req, err := http.NewRequest(r.Method, url, bytes.NewReader(body))
	if err != nil {
		log.L.Errorf("Failed to make request. ERROR: %s", err.Error())
		return err
	}
	req.Header.Add("content-type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.L.Errorf("Failed to send request. ERROR: %s", err.Error())
		return err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.L.Errorf("failed to read response body: %s", err)
		return err
	}

	if resp.StatusCode/100 != 2 {
		log.L.Errorf("NON 200 RESPONSE!!!. response code: %v. response body: %s", resp.StatusCode, respBody)
		return err
	}

	log.L.Debugf("response body: %s", respBody)
	return nil
}

// SendEvent sends an arbitrary event info
func SendEvent(e events.Event) error {
	log.L.Infof("Sending event: %+s", e)

	// send the event
	Messenger.SendEvent(e)

	return nil
}
