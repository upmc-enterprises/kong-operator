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

package k8sutil

import (
	"time"

	"github.com/upmc-enterprises/kong-operator/pkg/apis/cr/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/api/apps/v1beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"github.com/upmc-enterprises/kong-operator/pkg/apis/cr"
	client "github.com/upmc-enterprises/kong-operator/pkg/client/clientset/versioned"
	"fmt"
	"github.com/Sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/intstr"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

const (
	kongProxyServiceName   = "kong-proxy"
	kongAdminServiceName   = "kong-admin"
	kongDeploymentName     = "kong"
	kongPostgresSecretName = "kong-postgres"
)

// K8sutil defines the kube object
type K8sutil struct {
	KubernetesInterface kubernetes.Interface
	ClusterInterface rest.Interface
	ExtenstionsInterface apiextensionsclient.Interface
	ClusterClient *client.Clientset
}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

// New creates a new instance of k8sutil
func New(kubeconfig string) (*K8sutil, error) {
	// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
	config, err := buildConfig(kubeconfig)
	if err != nil {
		panic(err)
	}

	kubeclient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	crclient, err := client.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	extensionsclient, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	k := &K8sutil{
		KubernetesInterface: kubeclient,
		ClusterInterface: crclient.Cr().RESTClient(),
		ExtenstionsInterface: extensionsclient,
		ClusterClient: crclient,
	}

	return k, nil
}

// CreateKubernetesThirdPartyResource checks if Kong TPR exists. If not, create
func (k *K8sutil) CreateKubernetesThirdPartyResource() error {
	apiResourceList, err := k.ClusterClient.Discovery().ServerResources()

	if err != nil {
		panic(err)
	}

	groupVersion := v1.SchemeGroupVersion.String()

	for _, apiresource := range apiResourceList {
		if apiresource.GroupVersion == groupVersion {
			for _, resource := range apiresource.APIResources {
				if resource.SingularName == "cluster" {
					return nil
				}
			}
		}
	}

	crd := v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("clusters.%s", cr.GroupName),
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group: cr.GroupName,
			Version: v1.SchemeGroupVersion.Version,
			Scope: v1beta1.NamespaceScoped,
			Names: v1beta1.CustomResourceDefinitionNames{
				Plural: "clusters",
				Singular: "cluster",
				Kind: "Cluster",
				ShortNames: []string{
					"kc",
				},
			},
		},
	}
	_, err = k.ExtenstionsInterface.Apiextensions().CustomResourceDefinitions().Create(&crd)
	if err != nil {
		panic(err)
	}

	return nil
}

// GetKongClusters returns a list of custom clusters defined
func (k *K8sutil) GetKongClusters() (*v1.ClusterList, error) {
	return k.ClusterClient.CrV1().Clusters("").List(metav1.ListOptions{})
}

// MonitorKongEvents watches for new or removed clusters
func (k *K8sutil) MonitorKongEvents(stopchan chan struct{}) (<-chan *v1.Cluster, <-chan error) {
	events := make(chan *v1.Cluster)
	errc := make(chan error, 1)

	source := cache.NewListWatchFromClient(
		k.ClusterInterface,
		"clusters",
		corev1.NamespaceAll,
		fields.Everything())


	createAddHandler := func(obj interface{}) {
		event := obj.(*v1.Cluster)
		event.Status.State = v1.ClusterStateAdded
		events <- event
	}
	createDeleteHandler := func(obj interface{}) {
		event := obj.(*v1.Cluster)
		event.Status.State = v1.ClusterStateDeleted
		events <- event
	}

	updateHandler := func(old interface{}, obj interface{}) {
		event := obj.(*v1.Cluster)
		event.Status.State = v1.ClusterStateModified
		events <- event
	}

	_, controller := cache.NewInformer(
		source,
		&v1.Cluster{},
		time.Minute*60,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    createAddHandler,
			UpdateFunc: updateHandler,
			DeleteFunc: createDeleteHandler,
		})

	go controller.Run(stopchan)

	return events, errc
}

// CreateKongProxyService creates the kong proxy service
func (k *K8sutil) CreateKongProxyService(namespace string) error {

	// Check if service exists
	svc, err := k.KubernetesInterface.CoreV1().Services(namespace).Get(kongProxyServiceName, metav1.GetOptions{})

	// Service missing, create
	if len(svc.Name) == 0 {
		logrus.Infof("%s not found, creating...", kongProxyServiceName)

		clientSvc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: kongProxyServiceName,
				Labels: map[string]string{
					"name": kongProxyServiceName,
				},
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"app": "kong",
				},
				Ports: []corev1.ServicePort{
					corev1.ServicePort{
						Name:       "kong-proxy",
						Port:       80,
						TargetPort: intstr.FromInt(8000),
						Protocol:   "TCP",
					},
					corev1.ServicePort{
						Name:       "kong-proxy-ssl",
						Port:       443,
						TargetPort: intstr.FromInt(8443),
						Protocol:   "TCP",
					},
				},
				Type: corev1.ServiceTypeLoadBalancer,
				LoadBalancerSourceRanges: []string{
					"0.0.0.0/0",
				},
			},
		}

		_, err := k.KubernetesInterface.CoreV1().Services(namespace).Create(clientSvc)

		if err != nil {
			logrus.Error("Could not create proxy service", err)
			return err
		}
	} else if err != nil {
		logrus.Error("Could not get proxy service! ", err)
		return err
	}

	return nil
}

// CreateKongAdminService creates the kong proxy service
func (k *K8sutil) CreateKongAdminService(namespace string) error {

	// Check if service exists
	svc, err := k.KubernetesInterface.CoreV1().Services(namespace).Get(kongAdminServiceName, metav1.GetOptions{})

	// Service missing, create
	if len(svc.Name) == 0 {
		logrus.Infof("%s not found, creating...", kongAdminServiceName)

		clientSvc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: kongAdminServiceName,
				Labels: map[string]string{
					"name": kongAdminServiceName,
				},
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"app": "kong",
				},
				Ports: []corev1.ServicePort{
					corev1.ServicePort{
						Name:       "kong-admin",
						Port:       8444,
						TargetPort: intstr.FromInt(8444),
						Protocol:   "TCP",
					},
				},
				Type: corev1.ServiceTypeClusterIP,
			},
		}

		_, err := k.KubernetesInterface.CoreV1().Services(namespace).Create(clientSvc)

		if err != nil {
			logrus.Error("Could not create admin service: ", err)
			return err
		}
	} else if err != nil {
		logrus.Error("Could not get admin service: ", err)
		return err
	}

	return nil
}

// DeleteProxyService creates the kong proxy service
func (k *K8sutil) DeleteProxyService(namespace string) error {
	err := k.KubernetesInterface.CoreV1().Services(namespace).Delete(kongProxyServiceName, &metav1.DeleteOptions{})
	if err != nil {
		logrus.Error("Could not delete service "+kongProxyServiceName+":", err)
	} else {
		logrus.Infof("Delete service: %s", kongProxyServiceName)
	}

	return err
}

// DeleteAdminService creates the kong admin service
func (k *K8sutil) DeleteAdminService(namespace string) error {
	err := k.KubernetesInterface.CoreV1().Services(namespace).Delete(kongAdminServiceName, &metav1.DeleteOptions{})
	if err != nil {
		logrus.Error("Could not delete service "+kongAdminServiceName+":", err)
	} else {
		logrus.Infof("Delete service: %s", kongAdminServiceName)
	}

	return err
}

// CreateKongDeployment creates the kong deployment
func (k *K8sutil) CreateKongDeployment(baseImage string, replicas *int32, namespace string) error {

	// Check if deployment exists
	deployment, err := k.KubernetesInterface.AppsV1beta2().Deployments(namespace).Get(kongDeploymentName, metav1.GetOptions{})

	if len(deployment.Name) == 0 {
		logrus.Infof("%s not found, creating...", kongDeploymentName)

		deployment := &v1beta2.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: kongDeploymentName,
				Labels: map[string]string{
					"app": "kong",
					"name": kongDeploymentName,
				},
			},
			Spec: v1beta2.DeploymentSpec{
				Replicas: replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "kong",
						"name": kongDeploymentName,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app":  "kong",
							"name": kongDeploymentName,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Name:  kongDeploymentName,
								Image: baseImage,
								Env: []corev1.EnvVar{
									corev1.EnvVar{
										Name: "NAMESPACE",
										ValueFrom: &corev1.EnvVarSource{
											FieldRef: &corev1.ObjectFieldSelector{
												FieldPath: "metadata.namespace",
											},
										},
									},
									corev1.EnvVar{
										Name: "KONG_PG_USER",
										ValueFrom: &corev1.EnvVarSource{
											SecretKeyRef: &corev1.SecretKeySelector{
												Key: "KONG_PG_USER",
												LocalObjectReference: corev1.LocalObjectReference{
													Name: kongPostgresSecretName,
												},
											},
										},
									},
									corev1.EnvVar{
										Name: "KONG_PG_PASSWORD",
										ValueFrom: &corev1.EnvVarSource{
											SecretKeyRef: &corev1.SecretKeySelector{
												Key: "KONG_PG_PASSWORD",
												LocalObjectReference: corev1.LocalObjectReference{
													Name: kongPostgresSecretName,
												},
											},
										},
									},
									corev1.EnvVar{
										Name: "KONG_PG_HOST",
										ValueFrom: &corev1.EnvVarSource{
											SecretKeyRef: &corev1.SecretKeySelector{
												Key: "KONG_PG_HOST",
												LocalObjectReference: corev1.LocalObjectReference{
													Name: kongPostgresSecretName,
												},
											},
										},
									},
									corev1.EnvVar{
										Name: "KONG_PG_DATABASE",
										ValueFrom: &corev1.EnvVarSource{
											SecretKeyRef: &corev1.SecretKeySelector{
												Key: "KONG_PG_DATABASE",
												LocalObjectReference: corev1.LocalObjectReference{
													Name: kongPostgresSecretName,
												},
											},
										},
									},
									corev1.EnvVar{
										Name: "KONG_HOST_IP",
										ValueFrom: &corev1.EnvVarSource{
											FieldRef: &corev1.ObjectFieldSelector{
												APIVersion: "v1",
												FieldPath:  "status.podIP",
											},
										},
									},
									corev1.EnvVar{
										Name:  "KONG_ADMIN_LISTEN", // Disable non-tls
										Value: "127.0.0.1:8001",
									},
								},
								Command: []string{
									"/bin/sh", "-c",
									"KONG_CLUSTER_ADVERTISE=$(KONG_HOST_IP):7946 KONG_NGINX_DAEMON='off' kong start",
								},
								Ports: []corev1.ContainerPort{
									corev1.ContainerPort{
										Name:          "proxy",
										ContainerPort: 8000,
										Protocol:      corev1.ProtocolTCP,
									},
									corev1.ContainerPort{
										Name:          "proxy-ssl",
										ContainerPort: 8443,
										Protocol:      corev1.ProtocolTCP,
									},
									corev1.ContainerPort{
										Name:          "surf-tcp",
										ContainerPort: 7946,
										Protocol:      corev1.ProtocolTCP,
									},
									corev1.ContainerPort{
										Name:          "surf-udp",
										ContainerPort: 7946,
										Protocol:      corev1.ProtocolUDP,
									},
								},
							},
						},
					},
				},
			},
		}

		_, err := k.KubernetesInterface.AppsV1beta2().Deployments(namespace).Create(deployment)

		if err != nil {
			logrus.Error("Could not create kong deployment: ", err)
			return err
		}
	} else {
		if err != nil {
			logrus.Error("Could not get kong deployment! ", err)
			return err
		}

		//scale replicas?
		if deployment.Spec.Replicas != replicas {
			deployment.Spec.Replicas = replicas

			_, err := k.KubernetesInterface.AppsV1beta2().Deployments(namespace).Update(deployment)

			if err != nil {
				logrus.Error("Could not scale deployment: ", err)
			}
		}
	}

	return nil
}

// DeleteKongDeployment deletes kong deployment
func (k *K8sutil) DeleteKongDeployment(namespace string) error {

	// Get list of deployments
	deployment, err := k.KubernetesInterface.AppsV1beta2().Deployments(namespace).Get(kongDeploymentName, metav1.GetOptions{})

	if err != nil {
		logrus.Error("Could not get deployments! ", err)
		return err
	}

	//Scale the deployment down to zero (https://github.com/kubernetes/client-go/issues/91)
	deployment.Spec.Replicas = new(int32)
	_, err = k.KubernetesInterface.AppsV1beta2().Deployments(namespace).Update(deployment)

	if err != nil {
		logrus.Errorf("Could not scale deployment: %s ", deployment.Name)
	} else {
		logrus.Infof("Scaled deployment: %s to zero", deployment.Name)
	}

	err = k.KubernetesInterface.AppsV1beta2().Deployments(namespace).Delete(deployment.Name, &metav1.DeleteOptions{})

	if err != nil {
		logrus.Errorf("Could not delete deployments: %s ", deployment.Name)
	} else {
		logrus.Infof("Deleted deployment: %s", deployment.Name)
	}

	// ZZzzzzz...zzzzZZZzzz
	time.Sleep(2 * time.Second)

	// Get list of ReplicaSets
	replicaSets, err := k.KubernetesInterface.AppsV1().ReplicaSets(namespace).List(metav1.ListOptions{LabelSelector: "app=kong,name=kong"})

	if err != nil {
		logrus.Error("Could not get replica sets! ", err)
	}

	for _, replicaSet := range replicaSets.Items {
		err := k.KubernetesInterface.AppsV1().ReplicaSets(namespace).Delete(replicaSet.Name, &metav1.DeleteOptions{})

		if err != nil {
			logrus.Errorf("Could not delete replica sets: %s ", replicaSet.Name)
		} else {
			logrus.Infof("Deleted replica set: %s", replicaSet.Name)
		}
	}

	return nil
}
