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
	"fmt"
	"io/ioutil"

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

func buildJSON(plugin Plugin) string {
	s := fmt.Sprintf(`{"name":"%s"`, plugin.Name)
	for k, v := range plugin.Config {
		s += fmt.Sprintf(`,"config.%s":"%s"`, k, v)
	}
	s += `}`

	return s
}

// EnablePlugin enables a plugin
func (k *Kong) EnablePlugin(plugin Plugin, consumers []ConsumerTPR) {
	// Create the api object
	createPluginJSON := []byte(buildJSON(plugin))

	// --- Step1: Enable plugin
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
