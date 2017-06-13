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
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
)

var (
	// TODO: This needs to be based upon the namespace the
	kongAdminService    = fmt.Sprintf("https://%s:%s", os.Getenv("KONG_ADMIN_PORT_8444_TCP_ADDR"), os.Getenv("KONG_ADMIN_PORT_8444_TCP_PORT"))
	kongAdminServiceDNS = "https://kong-admin:8444"
)

// Kong struct
type Kong struct {
	client *http.Client
}

// New creates an instance of Kong
func New() (*Kong, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: time.Second * 10}

	k := &Kong{
		client: client,
	}
	return k, nil
}

// Ready creates a new Kong api
func (k *Kong) Ready(timeout chan bool, ready chan bool) {
	if len(kongAdminService) < 10 {
		kongAdminService = kongAdminServiceDNS
	}

	logrus.Infof("Using [%s] as kong-admin url", kongAdminService)

	for i := 0; i < 20; i++ {
		resp, err := k.client.Get(kongAdminService)
		if err != nil {
			logrus.Error("Error getting kong admin status: ", err)
		} else {
			if resp.StatusCode == 200 {
				logrus.Info("Kong API is ready!")
				ready <- true
				return
			}
		}
		time.Sleep(2 * time.Second)
	}
	timeout <- true
}
