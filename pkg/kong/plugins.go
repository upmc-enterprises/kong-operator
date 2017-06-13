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

var authPlugins = []string{"basic-auth", "key-auth", "jwt", "oauth2", "hmac-auth", "ldap-auth"}

// Plugin identifies a kong plugin
type Plugin struct {
	// Name of the plugin
	Name string `json:"name"`

	// Array of config items to setup plugin
	Config map[string]string `json:"config"`

	// Apis to apply the plugin
	Apis []string `json:"apis"`

	// Consumers define what users can access the API
	Consumers []Consumer `json:"consumers"`
}

// APIPlugins represents response from /apis/{name}/plugins/
type APIPlugins struct {
	Total int             `json:"total"`
	Data  []APIPluginData `json:"data"`
}

// APIPluginData represents data response from /apis/{name}/plugins/
type APIPluginData struct {
	ID        string `json:"id"`
	APIId     string `json:"api_id"`
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	CreatedAt int    `json:"created_at"`
}

func buildJSON(plugin Plugin) string {
	s := fmt.Sprintf(`{"name":"%s"`, plugin.Name)
	for k, v := range plugin.Config {
		s += fmt.Sprintf(`,"config.%s":"%s"`, k, v)
	}
	s += `}`

	return s
}

// EnablePlugin enables a plugin
func (k *Kong) EnablePlugin(plugin Plugin) {
	// Create the api object
	createPluginJSON := []byte(buildJSON(plugin))

	for _, api := range plugin.Apis {
		// Setup URL
		url := fmt.Sprintf("%s/apis/%s/plugins", kongAdminService, api)
		resp, err := k.client.Post(url, "application/json", bytes.NewBuffer(createPluginJSON))

		if err != nil {
			logrus.Error("Could not enable kong plugin: ", err)
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode != 201 {
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			logrus.Errorf("Enable plugin returned: %d Response: %s", resp.StatusCode, string(bodyBytes))
			continue
		}
		logrus.Infof("Enabled plugin: %s", plugin.Name)
	}
}

// UpdatePlugin updates a plugin
func (k *Kong) UpdatePlugin(plugin Plugin, ID string) {
	// Create the api object
	updatePluginJSON := []byte(buildJSON(plugin))

	for _, api := range plugin.Apis {
		// Setup URL
		url := fmt.Sprintf("%s/apis/%s/plugins/%s", kongAdminService, api, ID)

		req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(updatePluginJSON))
		if err != nil {
			logrus.Error("Could not update api: ", err)
			return
		}

		req.Header.Add("Content-Type", "application/json")
		resp, err := k.client.Do(req)

		if err != nil {
			logrus.Error("Could not update kong plugin: ", err)
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			logrus.Errorf("Enable plugin returned: %d Response: %s", resp.StatusCode, string(bodyBytes))
			continue
		}
		logrus.Infof("Updated plugin: %s", plugin.Name)
	}
}

// GetPlugins gets list of plugins enabled by api
func (k *Kong) GetPlugins() APIPlugins {
	var plugins APIPlugins

	url := fmt.Sprintf("%s/plugins", kongAdminService)
	resp, err := k.client.Get(url)

	if err != nil {
		logrus.Error("Could not get plugins: ", err)
		return plugins
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		logrus.Errorf("Get plugins returned: %d Response: %s", resp.StatusCode, string(bodyBytes))
		return plugins
	}

	// Parse from json
	json.NewDecoder(resp.Body).Decode(&plugins)

	return plugins
}

// DeletePlugin deletes Kong plugin
func (k *Kong) DeletePlugin(apiName, pluginID string) error {
	// Setup URL
	url := fmt.Sprintf("%s/apis/%s/plugins/%s", kongAdminService, apiName, pluginID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		logrus.Error("Could not create delete request: ", err)
		return err
	}

	resp, err := k.client.Do(req)
	if err != nil {
		logrus.Error("Could not delete kong plugin: ", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		logrus.Errorf("Delete plufin returned: %d Response: %s", resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("Delete plugin returned: %d", resp.StatusCode)
	}

	logrus.Infof("Deleted plugin: %s for API: %s", pluginID, apiName)

	return nil
}

// IsPluginExisting determines if a plugin is already existing
func (k *Kong) IsPluginExisting(plugin Plugin) (bool, APIPluginData) {

	for _, api := range plugin.Apis {
		url := fmt.Sprintf("%s/apis/%s/plugins", kongAdminService, api)
		resp, err := k.client.Get(url)

		if err != nil {
			logrus.Error("Could not check for kong plugin: ", err)
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			logrus.Errorf("IsPluginExisting plugin returned: %d Response: %s", resp.StatusCode, string(bodyBytes))
			return false, APIPluginData{}
		}

		// Parse from json
		var plugins APIPlugins
		json.NewDecoder(resp.Body).Decode(&plugins)

		// Check if matching
		for _, p := range plugins.Data {
			if p.Name == plugin.Name {
				return true, p
			}
		}
	}

	return false, APIPluginData{}
}

// RemovePlugin takes an item out of the data array
func RemovePlugin(s []APIPluginData, id string) []APIPluginData {
	var index = -1
	for i, p := range s {
		if p.ID == id {
			index = i
			break
		}
	}

	s[len(s)-1], s[index] = s[index], s[len(s)-1]
	return s[:len(s)-1]
}
