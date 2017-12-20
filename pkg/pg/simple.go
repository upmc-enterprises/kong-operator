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

package pg

import (
	"k8s.io/api/apps/v1beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/upmc-enterprises/kong-operator/pkg/k8sutil"
	"k8s.io/apimachinery/pkg/util/intstr"
	"github.com/Sirupsen/logrus"
	"fmt"
	"k8s.io/api/batch/v1"
)

// SimplePostgresDeployment returns a simple postgres deployment spec for testing purposes
func SimplePostgresDeployment(k *k8sutil.K8sutil, namespace string) error {
	var replicas int32
	replicas = int32(1)

	// Check if deployment exists
	deployment, err := k.KubernetesInterface.AppsV1beta2().Deployments(namespace).Get("postgres", metav1.GetOptions{})

	if len(deployment.Name) == 0 {
		logrus.Infof("%s not found, creating...", "postgres")

		deployment := &v1beta2.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "postgres",
				Labels: map[string]string{
					"name": "postgres",
				},
			},
			Spec: v1beta2.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"name": "postgres",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"name": "postgres",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Name:  "postgres",
								Image: "postgres:9.4",
								Env: []corev1.EnvVar{
									corev1.EnvVar{
										Name:  "POSTGRES_USER",
										Value: "kong",
									},
									corev1.EnvVar{
										Name:  "POSTGRES_PASSWORD",
										Value: "kong",
									},
									corev1.EnvVar{
										Name:  "POSTGRES_DB",
										Value: "kong",
									},
									corev1.EnvVar{
										Name:  "PGDATA",
										Value: "/var/lib/postgresql/data/pgdata",
									},
								},
								Ports: []corev1.ContainerPort{
									corev1.ContainerPort{
										Name:          "postgres",
										ContainerPort: 5432,
										Protocol:      corev1.ProtocolTCP,
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									corev1.VolumeMount{
										Name:      "pg-data",
										MountPath: "/var/lib/postgresql/data",
									},
								},
							},
						},
						Volumes: []corev1.Volume{
							corev1.Volume{
								Name: "pg-data",
								VolumeSource: corev1.VolumeSource{
									EmptyDir: &corev1.EmptyDirVolumeSource{},
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
	} else if err != nil {
		logrus.Error("Could not get admin service: ", err)
		return err
	}

	return nil
}

// SimplePostgresService creates the postgres service
func SimplePostgresService(k *k8sutil.K8sutil, namespace string) error {

	// Check if service exists
	svc, err := k.KubernetesInterface.CoreV1().Services(namespace).Get("postgres", metav1.GetOptions{})

	// Service missing, create
	if len(svc.Name) == 0 {
		logrus.Infof("%s not found, creating...", "postgres")

		clientSvc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: "postgres",
				Labels: map[string]string{
					"name": "postgres",
				},
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"name": "postgres",
				},
				Ports: []corev1.ServicePort{
					corev1.ServicePort{
						Name:       "pgql",
						Port:       5432,
						TargetPort: intstr.FromInt(5432),
						Protocol:   "TCP",
					},
				},
				Type: corev1.ServiceTypeClusterIP,
			},
		}

		_, err := k.KubernetesInterface.CoreV1().Services(namespace).Create(clientSvc)

		if err != nil {
			logrus.Error("Could not create postgres service: ", err)
			return err
		}
	} else if err != nil {
		logrus.Error("Could not get postgres service: ", err)
		return err
	}

	return nil
}

// SimplePostgresSecret creates the postgres secrets
func SimplePostgresSecret(k *k8sutil.K8sutil, namespace string) error {

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kong-postgres",
		},
		Data: map[string][]byte{
			"KONG_PG_USER":     []byte("kong"),
			"KONG_PG_PASSWORD": []byte("kong"),
			"KONG_PG_HOST":     []byte(fmt.Sprintf("postgres.%s.svc.cluster.local", namespace)),
			"KONG_PG_DATABASE": []byte("kong"),
		},
	}

	_, err := k.KubernetesInterface.CoreV1().Secrets(namespace).Create(secret)

	if err != nil {
		logrus.Error("Could not create postgres secret: ", err)
		return err
	}

	return nil
}

func SimpleKongMigrationJob(k *k8sutil.K8sutil, namespace string) error {

	job := v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kong-migration",
		},
		Spec: v1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kong-migration",
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name: "kong-migration",
							Image: "kong",
							Env: []corev1.EnvVar{
								{
									Name: "KONG_NGINX_DAEMON",
									Value: "off",
								},
								{
									Name: "KONG_PG_PASSWORD",
									Value: "kong",
								},
								{
									Name: "KONG_PG_HOST",
									Value: "postgres.default.svc.cluster.local",
								},
							},
							Command: []string{ "/bin/sh", "-c", "kong migrations up" },
						},
					},
				},
			},
		},
	}

	_, err := k.KubernetesInterface.BatchV1().Jobs(namespace).Create(&job)
	if err != nil {
		logrus.Error("Could not create job for kong migration", err)
		return err
	}

	return nil
}

// // DeleteSimplePostgresSecret deletes simple secret
// func DeleteSimplePostgresSecret(k *k8sutil.K8sutil, namespace string) error {
// }

// DeleteSimplePostgres cleans up deployment / service for postgres db
func DeleteSimplePostgres(k *k8sutil.K8sutil, namespace string) {

	//TODO DELETE SECRET!
}
