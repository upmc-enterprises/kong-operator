# kong-operator

Kong is a scalable, open source API Gateway which runs in front of any RESTful API and is extended through Plugins, which provide extra functionality and services beyond the core platform. 

The kong-operator automates kong so that your apis will be deployed and secured consistently to avoid human error in configuring kong. 

## Prerequisites

Kong needs a database (postgres or cassandra) to be running and with proper credentials before kong starts. A built-in sample Postgres database deployment is provided to get started quickly, but this can be swapped out based upon configuration. 

## Usage

The operator is built using the controller + third party resource model. Once the controller is deployed to your cluster, it will automatically create the ThirdPartyResource. Next create a Kubernetes object type elasticsearchCluster to deploy the elastic cluster based upon the TPR.

## ThirdPartyResource

Use the following spec to customize your Kong cluster:

- Spec:
  - Name: Name of cluster
  - Replicas: Number of kong instances to run in the cluster
  - UseSamplePostgres: Set to `true` if using the provided Postgres DB
  - APIs[]: Array of API's to setup
    - Name: Name of service
    - Upstream URL: Where to proxy the request to inside the k8s cluster
    - Hosts[]: List of hosts to accept requests
  - Plugins[]: Array of plugins to enable
    - Name: Name of plugin ([See kong site](https://getkong.org/plugins/))
    - APIs[]: List of apis to enable plugin
    - Consumers[]: List of consumers to grant access to api
  - Consumers[]: List of consumers to pre-configure
    - UserName: Required by operator
    - CustomID: CustomID to setup
    - MaxNumCreds: Max number of creds that the operator is allowed to provision 


## Deploy sample apps

The example cluster in the `./examples/` directory utilizes some sample apps defined here:

```
# Go Service
kubectl create ns apps
kubectl create -f https://raw.githubusercontent.com/stevesloka/microservice-sample/master/k8s/deployment.yaml -n apps
kubectl create -f https://raw.githubusercontent.com/stevesloka/microservice-sample/master/k8s/service.yaml -n apps
```

### Output
 
The sample app prints the IP of the pod which the request was handled as well as the time the request was handled:

```
$ http http://go-microservice.apps:8080                                                                                   [k8s-minikube/ 9:43:09]
HTTP/1.1 200 OK
Content-Length: 59
Content-Type: text/plain; charset=utf-8
Date: Wed, 14 Jun 2017 04:48:18 GMT

Service: 172.17.0.4 
Request Time: Wed Jun 14 04:48:18 2017
```

## Deploy Operator

Use the following to deploy the operator to your cluster:

```
$ kubectl create -f https://raw.githubusercontent.com/upmc-enterprises/kong-operator/master/example/operator.yaml
```

## Quick Start

To get started quickly, setup the sample apps defined in previous section as well as deploy the operator to your cluster. Once those pieces are ready, then create the custom kong cluster.

### Deploy sample cluster

After running this create, a Kong cluster will be created and configured automatically. Any request to `service-go.k8s.com` will route to the k8s service `http://go-microservice.apps.svc.cluster.local:8080` inside the apps namespace. In addition the JWT plugin will be enabled forcing authentication and a consumer named `slokas` will be generated. 

```
$ kubectl create -f https://raw.githubusercontent.com/upmc-enterprises/kong-operator/master/example/example-kong-cluster.json
```

### Create JWT creds

Currently, credentials are not automatically created. In the case of JWT, a token needs to be created to from the creds that Kong generates. There are a number of ways to accomplish this, but one easy way is with this project (https://github.com/stevesloka/jwt-creator). Just pop in the secret and key from Kong and it will generate a sample JWT. After that, send a request to the API passing the token as a header:

```
$ curl -X POST https://kong-admin.default:8444/consumers/slokas/jwt -H "Content-Type: application/x-www-form-urlencoded"

# --- Generate token
$ git clone https://github.com/stevesloka/jwt-creator.git
$ (Update key / secret in code)
$ go run main.go

# --- Make request
$ curl -i https://kong-proxy \
    -H 'Authorization: Bearer <token>' \
    -H 'Host: service-go.k8s.com'
```

###

## Update existing TPR

Changes to an existing TPR can be updated by sending a `PUT` request to the API server:

```
$ curl -H 'Content-Type: application/json' -X PUT --data @example/example-kong-cluster.json http://127.0.0.1:8001/apis/enterprises.upmc.com/v1/namespaces/default/k
ongclusters/example-kong-cluster
```

_NOTE: Update the namespace above to match your deployment as well as the name of the cluster._

## Custom Postgres Database

To use your own Postgres db, just set the option `useSamplePostgres=false`, then create a secret named `kong-postgres` and set the following params:

| Key               | Value             | 
| ----------------- |:-----------------:| 
| KONG_PG_DATABASE  | Name of database  | 
| KONG_PG_HOST      | Server host       | 
| KONG_PG_PASSWORD  | Password for user | 
| KONG_PG_USER      | DB User           | 

## Dev locally

Use the following command to run the operator locally:
```
go run cmd/operator/main.go --kubecfg-file=${HOME}/.kube/config
```

## About

Built by UPMC Enterprises in Pittsburgh, PA. http://enterprises.upmc.com/