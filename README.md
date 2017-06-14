# kong-operator

Kong is a scalable, open source API Gateway which runs in front of any RESTful API and is extended through Plugins, which provide extra functionality and services beyond the core platform. 

The kong-operator automates kong so that your apis will be deployed and secured consistently to avoid human error in configuring kong. 

## Prerequisites

Kong needs a database (postgres or cassandra) to be running and with proper credentials before kong starts. A built-in sample Postgres database deployment is provided to get started quickly, but this can be swapped out based upon configuration. 

## Usage

The operator is built using the controller + third party resource model. Once the controller is deployed to your cluster, it will automatically create the ThirdPartyResource. Next create a Kubernetes object type elasticsearchCluster to deploy the elastic cluster based upon the TPR.

## ThirdPartyResource

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

## Create JWT creds

Currently, credentials are not
```
$ curl -X POST http://kong-admin.default.svc.cluster.local:8001/consumers/slokas/jwt -H "Content-Type: application/x-www-form-urlencoded"
```

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
`go run cmd/operator/main.go --kubecfg-file=${HOME}/.kube/config`