/*
Copyright (c) 2017, UPMC Enterprises
All rights reserved.
Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:
    * Redistributions of source code must retain the above copyright
      notice, this list of conditions and the following disclaimer.
    * Redistributions in binary form must reproduce the above copyright
      notice, this list of conditions and the following disclaimer in the
      documentation and/or other materials provided with the distribution.
    * Neither the name UPMC Enterprises nor the
      names of its contributors may be used to endorse or promote products
      derived from this software without specific prior written permission.
THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL UPMC ENTERPRISES BE LIABLE FOR ANY
DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
*/

package kong

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Sirupsen/logrus"
)

// ConsumerTPR represents a user of the API
type ConsumerTPR struct {
	Username    string `json:"username"`
	CustomID    string `json:"custom_id"`
	MaxNumCreds int    `json:"maxNumCreds"`
}

// Consumer represents a user of the API
type Consumer struct {
	Username string `json:"username"`
	CustomID string `json:"custom_id"`
}

// Consumers is the call to /consumers
type Consumers struct {
	Data  []Consumer `json:"data"`
	Total int        `json:"total"`
}

// CreateConsumer creates a consumer
func (k *Kong) CreateConsumer(consumer ConsumerTPR) error {
	c := Consumer{Username: consumer.Username, CustomID: consumer.CustomID}

	// Create JSON
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(c)

	// Setup URL
	url := fmt.Sprintf("%s/consumers/", k.KongAdminURL)

	resp, err := k.client.Post(url, "application/json", buf)

	if err != nil {
		logrus.Error("Could not create consumer: ", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		logrus.Errorf("Create consumer returned: %d Response: %s", resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("Create consumer returned: %d", resp.StatusCode)
	}

	logrus.Infof("Created consumer: %s", consumer.Username)

	return nil
}

// DeleteConsumer deletes a consumer
func (k *Kong) DeleteConsumer(consumer Consumer) error {
	// Setup URL
	url := fmt.Sprintf("%s/consumers/%s", k.KongAdminURL, consumer.Username)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		logrus.Error("Could not delete consumer: ", err)
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	resp, err := k.client.Do(req)

	if err != nil {
		logrus.Error("Could not delete consumer: ", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		logrus.Errorf("Delete consumer returned: %d Response: %s", resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("Delete consumer returned: %d", resp.StatusCode)
	}

	logrus.Infof("Deleted consumer: %s", consumer.Username)

	return nil
}

// ConsumerExists checks if a consumer exists already
func (k *Kong) ConsumerExists(username string) bool {
	// Setup URL
	url := fmt.Sprintf("%s/consumers/%s", k.KongAdminURL, username)

	resp, err := k.client.Get(url)

	if err != nil {
		logrus.Error("Could not create consumer: ", err)
		return false
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		logrus.Errorf("Create consumer returned: %d Response: %s", resp.StatusCode, string(bodyBytes))
		return false
	}

	return true
}

// GetConsumers gets list of consumers
func (k *Kong) GetConsumers() Consumers {
	var consumers Consumers

	// Setup URL
	url := fmt.Sprintf("%s/consumers/", k.KongAdminURL)

	resp, err := k.client.Get(url)

	if err != nil {
		logrus.Error("Could not get kong consumers: ", err)
		return consumers
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		logrus.Errorf("Get consumers returned: %d Response: %s", resp.StatusCode, string(bodyBytes))
		return consumers
	}

	json.NewDecoder(resp.Body).Decode(&consumers)

	return consumers
}

// GetJWTPluginCreds gets creds for
func (k *Kong) GetJWTPluginCreds(username string) JWTCreds {
	// Setup URL
	url := fmt.Sprintf("%s/consumers/%s/jwt", k.KongAdminURL, username)

	resp, err := k.client.Get(url)

	if err != nil {
		logrus.Error("Could get jwt-auth for consumer: ", err)
		return JWTCreds{}
	}

	defer resp.Body.Close()

	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {

		logrus.Errorf("Get consumer auth returned: %d Response: %s", resp.StatusCode, string(bodyBytes))
		return JWTCreds{}
	}

	// parse to object
	var creds JWTCreds
	err = json.Unmarshal(bodyBytes, &creds)
	if err != nil {
		logrus.Error("Error parsing jwt-creds: ", err)
	}

	return creds
}

// GetKeyPluginCreds gets creds for
func (k *Kong) GetKeyPluginCreds(username string) KeyCreds {
	// Setup URL
	url := fmt.Sprintf("%s/consumers/%s/key-auth", k.KongAdminURL, username)

	resp, err := k.client.Get(url)

	if err != nil {
		logrus.Error("Could get key-auth for consumer: ", err)
		return KeyCreds{}
	}

	defer resp.Body.Close()

	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {

		logrus.Errorf("Get consumer auth returned: %d Response: %s", resp.StatusCode, string(bodyBytes))
		return KeyCreds{}
	}

	// parse to object
	var creds KeyCreds
	err = json.Unmarshal(bodyBytes, &creds)
	if err != nil {
		logrus.Error("Error parsing key-creds: ", err)
	}

	return creds
}

// FindConsumer finds item in existing consumers
func FindConsumer(a string, list []ConsumerTPR) (bool, ConsumerTPR) {
	for i, b := range list {
		if b.Username == a {
			return true, list[i]
		}
	}
	return false, ConsumerTPR{}
}

// RemoveConsumer takes an item out of the data array
func RemoveConsumer(s []Consumer, username string) []Consumer {
	if len(s) == 0 {
		return []Consumer{}
	}

	var index = -1
	for i, p := range s {
		if p.Username == username {
			index = i
			break
		}
	}

	s[len(s)-1], s[index] = s[index], s[len(s)-1]
	return s[:len(s)-1]
}
