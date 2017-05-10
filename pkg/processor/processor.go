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
	"os"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/upmc-enterprises/kong-operator/pkg/k8sutil"
	"github.com/upmc-enterprises/kong-operator/pkg/pg"
	myspec "github.com/upmc-enterprises/kong-operator/pkg/spec"
)

// processorLock ensures that reconciliation and event processing does
// not happen at the same time.
var (
	processorLock = &sync.Mutex{}
	namespace     = os.Getenv("NAMESPACE")
)

// Processor object
type Processor struct {
	k8sclient *k8sutil.K8sutil
	baseImage string
	clusters  map[string]*myspec.KongCluster
}

// New creates new instance of Processor
func New(kclient *k8sutil.K8sutil, baseImage string) (*Processor, error) {
	p := &Processor{
		k8sclient: kclient,
		baseImage: baseImage,
		clusters:  make(map[string]*myspec.KongCluster),
	}

	return p, nil
}

// Run starts the processor
func (p *Processor) Run() error {

	p.refreshClusters()
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

	//Reset
	p.clusters = make(map[string]*myspec.KongCluster)

	// Get existing clusters
	currentClusters, err := p.k8sclient.GetKongClusters()

	if err != nil {
		logrus.Error("Could not get list of clusters: ", err)
		return err
	}

	for _, cluster := range currentClusters {
		logrus.Infof("Found cluster: %s", cluster.Metadata.Name)

		p.clusters[cluster.Metadata.Name] = &myspec.KongCluster{
			Spec: myspec.ClusterSpec{
				Replicas:          cluster.Spec.Replicas,
				BaseImage:         cluster.Spec.BaseImage,
				UseSamplePostgres: cluster.Spec.UseSamplePostgres,
			},
		}
	}

	return nil
}

func (p *Processor) processKongEvent(c *myspec.KongCluster) error {
	processorLock.Lock()
	defer processorLock.Unlock()

	switch {
	case c.Type == "ADDED" || c.Type == "MODIFIED":
		return p.processKong(c)
	case c.Type == "DELETED":
		return p.deleteKong(c)
	}
	return nil
}

func (p *Processor) processKong(c *myspec.KongCluster) error {
	logrus.Println("--------> Kong Event!")

	// Refresh
	p.refreshClusters()

	// Deploy sample postgres deployments?
	if c.Spec.UseSamplePostgres {
		pg.SimplePostgresService(p.k8sclient, namespace)
		pg.SimplePostgresDeployment(p.k8sclient, namespace)
	} else {
		pg.DeleteSimplePostgres(p.k8sclient, namespace)
	}

	// Is a base image defined in the custom cluster?
	var baseImage = p.calcBaseImage(p.baseImage, c.Spec.BaseImage)

	logrus.Infof("Using [%s] as image for es cluster", baseImage)

	// Create Services
	p.k8sclient.CreateKongAdminService()
	p.k8sclient.CreateKongProxyService()

	// Create deployment
	p.k8sclient.CreateKongDeployment(baseImage, &c.Spec.Replicas)

	return nil
}

func (p *Processor) deleteKong(c *myspec.KongCluster) error {
	logrus.Println("--------> Kong Cluster deleted...removing all components...")

	err := p.k8sclient.DeleteKongDeployment()
	if err != nil {
		logrus.Error("Could not delete kong deployment:", err)
	}

	err = p.k8sclient.DeleteAdminService()
	if err != nil {
		logrus.Error("Could not delete admin service:", err)
	}

	err = p.k8sclient.DeleteProxyService()
	if err != nil {
		logrus.Error("Could not delete proxy service:", err)
	}

	if c.Spec.UseSamplePostgres {
		pg.DeleteSimplePostgres(p.k8sclient, namespace)
	}

	return nil
}

func (p *Processor) calcBaseImage(baseImage, customImage string) string {
	if len(customImage) > 0 {
		baseImage = customImage
	}

	return baseImage
}
