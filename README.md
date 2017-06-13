# kong-operator

## Deploy sample apps

The example cluster in the `./examples/` directory utilizes some sample apps defined here:

```
# Go Service
kubectl create ns apps
kubectl create -f https://raw.githubusercontent.com/stevesloka/microservice-sample/master/k8s/deployment.yaml -n apps
kubectl create -f https://raw.githubusercontent.com/stevesloka/microservice-sample/master/k8s/service.yaml -n apps
```

## Create JWT creds
```
$ curl -X POST http://kong-admin.default.svc.cluster.local:8001/consumers/slokas/jwt -H "Content-Type: application/x-www-form-urlencoded"
```


## Update existing TPR

```
$ curl -H 'Content-Type: application/json' -X PUT --data @example/example-kong-cluster.json http://127.0.0.1:8001/apis/enterprises.upmc.com/v1/namespaces/default/k
ongclusters/example-kong-cluster
```

## Dev locally

Use the following command to run the operator locally:
`NAMESPACE=default  go run cmd/operator/main.go --kubecfg-file=${HOME}/.kube/config`