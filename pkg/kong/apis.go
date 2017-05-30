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
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
)

// APIS represents a result from GET /apis
type APIS struct {
	Data  []Data `json:"data"`
	Total int    `json:"total"`
}

// Data represents an api
type Data struct {
	ID                     string        `json:"id"`
	Name                   string        `json:"name"`
	CreatedAt              time.Duration `json:"created_at"`
	Hosts                  []string      `json:"hosts"`
	HTTPIfTerminated       bool          `json:"http_if_terminated"`
	HTTPSOnly              bool          `json:"https_only"`
	PreserveHost           bool          `json:"preserve_host"`
	Retries                int           `json:"retries"`
	StripURI               bool          `json:"strip_uri"`
	UpstreamConnectTimeout int           `json:"upstream_connect_timeout"`
	UpstreamReadTimeout    int           `json:"upstream_read_timeout"`
	UpstreamSendTimeout    int           `json:"upstream_send_timeout"`
	UpstreamURL            string        `json:"upstream_url"`
}

// API represents the object required to create an API in Kong
type API struct {
	Name     string `json:"name"`
	Upstream string `json:"upstream_url"`
	Hosts    string `json:"hosts"`
}

// GetAPIs creates a new Kong api
func (k *Kong) GetAPIs() APIS {
	var apis APIS

	// Setup URL
	url := fmt.Sprintf("%s/apis/", kongAdminService)

	resp, err := k.client.Get(url)

	if err != nil {
		logrus.Error("Could not get kong apis: ", err)
		return apis
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		logrus.Errorf("Create api returned: %d Response: %s", resp.StatusCode, string(bodyBytes))
		return apis
	}

	json.NewDecoder(resp.Body).Decode(&apis)

	return apis
}

// CreateAPI creates a new Kong api
func (k *Kong) CreateAPI(api Data) error {
	// Create the api object
	createAPI := &API{Name: api.Name, Upstream: api.UpstreamURL, Hosts: strings.Join(api.Hosts, ",")}

	// Create JSON
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(createAPI)

	// Setup URL
	url := fmt.Sprintf("%s/apis/", kongAdminService)

	resp, err := k.client.Post(url, "application/json", buf)

	if err != nil {
		logrus.Error("Could not create kong api: ", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		logrus.Errorf("Create api returned: %d Response: %s", resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("Create api returned: %d", resp.StatusCode)
	}

	logrus.Infof("Created api: %s", api.Name)

	return nil
}

// UpdateAPI creates a new Kong api
func (k *Kong) UpdateAPI(name, upstream string, hosts []string) error {
	// Create the api object
	updateAPI := &API{Name: name, Upstream: upstream, Hosts: strings.Join(hosts, ",")}

	// Create JSON
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(updateAPI)

	// Setup URL
	url := fmt.Sprintf("%s/apis/%s", kongAdminService, name)

	req, err := http.NewRequest("PATCH", url, buf)
	if err != nil {
		logrus.Error("Could not update api: ", err)
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	resp, err := k.client.Do(req)

	if err != nil {
		logrus.Error("Could not update kong api: ", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		logrus.Errorf("Update api returned: %d Response: %s", resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("Update api returned: %d", resp.StatusCode)
	}

	logrus.Infof("Update api: %s", name)

	return nil
}

// DeleteAPI creates a new Kong api
func (k *Kong) DeleteAPI(name string) error {
	// Setup URL
	url := fmt.Sprintf("%s/apis/%s", kongAdminService, name)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		logrus.Error("Could not create delete request: ", err)
		return err
	}

	resp, err := k.client.Do(req)
	if err != nil {
		logrus.Error("Could not delete kong api: ", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		logrus.Errorf("Delete api returned: %d Response: %s", resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("Delete api returned: %d", resp.StatusCode)
	}

	logrus.Infof("Deleted api: %s", name)

	return nil
}

// FindAPI finds item in existing apis
func FindAPI(a string, list []Data) (bool, int) {
	for i, b := range list {
		if b.Name == a {
			return true, i
		}
	}
	return false, -1
}

// Remove takes an item out of the data array
func Remove(s []Data, i int) []Data {
	s[len(s)-1], s[i] = s[i], s[len(s)-1]
	return s[:len(s)-1]
}
