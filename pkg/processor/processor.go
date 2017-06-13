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

package processor

import (
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/upmc-enterprises/kong-operator/pkg/k8sutil"
	"github.com/upmc-enterprises/kong-operator/pkg/kong"
	"github.com/upmc-enterprises/kong-operator/pkg/pg"
	"github.com/upmc-enterprises/kong-operator/pkg/tpr"
)

// processorLock ensures that reconciliation and event processing does
// not happen at the same time.
var (
	processorLock = &sync.Mutex{}
)

// Processor object
type Processor struct {
	k8sclient *k8sutil.K8sutil
	baseImage string
	clusters  map[string]KongCluster
}

// KongCluster represents an instance of a cluster setup via TPR
type KongCluster struct {
	cluster *tpr.KongCluster
	kong    *kong.Kong
}

// New creates new instance of Processor
func New(kclient *k8sutil.K8sutil, baseImage string) (*Processor, error) {
	p := &Processor{
		k8sclient: kclient,
		baseImage: baseImage,
		clusters:  make(map[string]KongCluster),
	}

	return p, nil
}

// Run starts the processor
func (p *Processor) Run() error {

	logrus.Infof("Found %d existing clusters ", len(p.clusters))

	return nil
}

// WatchKongEvents watches for changes to tpr kong events
func (p *Processor) WatchKongEvents(done chan struct{}, wg *sync.WaitGroup) {
	events, watchErrs := p.k8sclient.MonitorKongEvents(done)
	go func() {
		for {
			select {
			case event := <-events:
				err := p.processKongEvent(event)
				if err != nil {
					logrus.Errorln(err)
				}
			case err := <-watchErrs:
				logrus.Errorln(err)
			case <-done:
				wg.Done()
				logrus.Println("Stopped Kong event watcher.")
				return
			}
		}
	}()
}

func (p *Processor) refreshClusters() error {

	// Get existing clusters
	currentClusters, err := p.k8sclient.GetKongClusters()

	if err != nil {
		logrus.Error("Could not get list of clusters: ", err)
		return err
	}

	for _, cluster := range currentClusters {
		logrus.Infof("Found cluster: %s", cluster.Spec.Name)
		p.addInstanceIfNotExisting(cluster)
	}

	return nil
}

func (p *Processor) addInstanceIfNotExisting(cluster tpr.KongCluster) {
	if _, exists := p.clusters[cluster.Spec.Name]; exists {
		// Cluster already existing
	} else {
		logrus.Infof("Found new cluster! [%s]", cluster.Spec.Name)
		p.clusters[cluster.Spec.Name] = p.createKongInstance(cluster)
	}
}

func (p *Processor) createKongInstance(cluster tpr.KongCluster) KongCluster {

	kong, err := kong.New(cluster.Metadata.Namespace)

	if err != nil {
		logrus.Error("Error creating kong instance: ", err)
	}

	kc := KongCluster{
		cluster: &tpr.KongCluster{
			Spec: tpr.ClusterSpec{
				Name:              cluster.Spec.Name,
				Replicas:          cluster.Spec.Replicas,
				BaseImage:         cluster.Spec.BaseImage,
				UseSamplePostgres: cluster.Spec.UseSamplePostgres,
				Apis:              cluster.Spec.Apis,
			},
		},
		kong: kong,
	}

	return kc
}

func (p *Processor) processKongEvent(c *tpr.KongCluster) error {
	processorLock.Lock()
	defer processorLock.Unlock()

	switch {
	case c.Type == "ADDED":
		return p.createKong(c)
	case c.Type == "MODIFIED":
		return p.modifyKong(c)
	case c.Type == "DELETED":
		return p.deleteKong(c)
	}
	return nil
}

func (p *Processor) modifyKong(c *tpr.KongCluster) error {
	logrus.Println("--------> Update Kong Event!")

	p.process(c)

	return nil
}

func (p *Processor) createKong(c *tpr.KongCluster) error {
	logrus.Println("--------> Create Kong Event!")

	// Create instance in local cluster list
	p.clusters[c.Spec.Name] = p.createKongInstance(*c)

	// Deploy sample postgres deployments?
	if c.Spec.UseSamplePostgres {
		pg.SimplePostgresSecret(p.k8sclient, c.Metadata.Namespace)
	}

	// Is a base image defined in the custom cluster?
	var baseImage = p.calcBaseImage(p.baseImage, c.Spec.BaseImage)

	logrus.Infof("Using [%s] as image for es cluster", baseImage)

	// Create Services
	p.k8sclient.CreateKongAdminService(c.Metadata.Namespace)
	p.k8sclient.CreateKongProxyService(c.Metadata.Namespace)

	// Create deployment
	p.k8sclient.CreateKongDeployment(baseImage, &c.Spec.Replicas, c.Metadata.Namespace)

	// Wait for kong to be ready
	timeout := make(chan bool, 1)
	ready := make(chan bool)

	logrus.Info("Waiting for Kong API to become ready....")
	go p.clusters[c.Spec.Name].kong.Ready(timeout, ready)

	select {
	case <-ready:
		p.process(c)
	case <-timeout:
		// the read from ready has timed out
		logrus.Error("Giving up waiting for kong-api to become ready...")
	}

	return nil
}

func (p *Processor) process(c *tpr.KongCluster) {
	// Lookup cluster
	p.addInstanceIfNotExisting(*c)

	// Get current APIs from Kong
	kongApis := p.clusters[c.Spec.Name].kong.GetAPIs()

	// Get current plugins from Kong
	kongPlugins := p.clusters[c.Spec.Name].kong.GetPlugins()

	logrus.Infof("Found %d apis existing in kong api...", kongApis.Total)

	// process apis
	for _, api := range c.Spec.Apis {

		found, position := kong.FindAPI(api.Name, kongApis.Data)

		// --- Apis
		if found {
			logrus.Infof("Existing API [%s] found, updating...", kongApis.Data[position].Name)
			p.clusters[c.Spec.Name].kong.UpdateAPI(api.Name, api.UpstreamURL, api.Hosts)

			// Clean up local list
			kongApis.Data = kong.RemoveAPI(kongApis.Data, position)
		} else {
			logrus.Infof("API [%s] not found, creating...", c.Spec.Name)
			p.clusters[c.Spec.Name].kong.CreateAPI(api)
		}

		// --- Consumers
		for _, consumer := range c.Spec.Consumers {
			logrus.Info("Processing consumer: ", consumer.Username)

			if !p.clusters[c.Spec.Name].kong.ConsumerExists(consumer.Username) {
				p.clusters[c.Spec.Name].kong.CreateConsumer(consumer)
			}
		}

		// --- Plugins
		for _, plugin := range c.Spec.Plugins {
			logrus.Info("Processing plugin: ", plugin.Name)

			existing, plug := p.clusters[c.Spec.Name].kong.IsPluginExisting(plugin)

			if existing {
				logrus.Infof("Plugin already existing [%s], updating...", plugin.Name)
				p.clusters[c.Spec.Name].kong.UpdatePlugin(plugin, plug.ID)

				// Clean up local list
				kongPlugins.Data = kong.RemovePlugin(kongPlugins.Data, plug.ID)
			} else {
				logrus.Infof("Plugin not found [%s], creating...", plugin.Name)
				p.clusters[c.Spec.Name].kong.EnablePlugin(plugin)
			}
		}
	}

	// Delete existing apis left
	for _, api := range kongApis.Data {
		logrus.Infof("Deleting api: %s", api.Name)
		p.clusters[c.Spec.Name].kong.DeleteAPI(api.Name)
	}

	// Delete existing plugins left
	for _, plug := range kongPlugins.Data {
		logrus.Infof("Deleting plugin: %s", plug.Name)
		p.clusters[c.Spec.Name].kong.DeletePlugin(plug.APIId, plug.ID)
	}
}

func (p *Processor) deleteKong(c *tpr.KongCluster) error {
	logrus.Println("--------> Kong Cluster deleted...removing all components...")

	err := p.k8sclient.DeleteKongDeployment(p.clusters[c.Spec.Name].kong.KongAdminURL)
	if err != nil {
		logrus.Error("Could not delete kong deployment:", err)
	}

	err = p.k8sclient.DeleteAdminService(p.clusters[c.Spec.Name].kong.KongAdminURL)
	if err != nil {
		logrus.Error("Could not delete admin service:", err)
	}

	err = p.k8sclient.DeleteProxyService(p.clusters[c.Spec.Name].kong.KongAdminURL)
	if err != nil {
		logrus.Error("Could not delete proxy service:", err)
	}

	if c.Spec.UseSamplePostgres {
		pg.DeleteSimplePostgres(p.k8sclient, c.Metadata.Namespace)
	}

	return nil
}

func (p *Processor) calcBaseImage(baseImage, customImage string) string {
	if len(customImage) > 0 {
		baseImage = customImage
	}

	return baseImage
}
