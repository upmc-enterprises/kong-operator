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

	"github.com/Sirupsen/logrus"
)

// Consumer represents a user of the API
type Consumer struct {
	Username string `json:"username"`
	CustomID string `json:"custom_id"`
}

// CreateConsumer creates a consumer
func (k *Kong) CreateConsumer(consumer Consumer) error {
	// Create JSON
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(consumer)

	// Setup URL
	url := fmt.Sprintf("%s/consumers/", kongAdminService)

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
