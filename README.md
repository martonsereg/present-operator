# Present operator

This is a sample operator for demonstrating some of the basic concepts of a Kubernetes Operator.

It manages a [go present](https://godoc.org/golang.org/x/tools/present) deployment by creating the necessary configmap, deployment and service.
The main CRD contains the contents of the slides that will be hosted by present, but the deployment itself picks it up from a configmap.
On an update of the content, the operator restarts (deletes) the pods in the deployment, so they will pick up the new contents from the configmap.

This is a sample Kubebuilder project, so generating code, creating CRD YAMLs and deploying code is done in the standard way.

## Why

Complex operators are too difficult to understand when someone just met the concept, but it's good to see things working instead of only reading about it.
Kubebuilder examples (CronJob, ReplicaSet) are great, but they are mostly controllers instead of an operator.
This is the smallest operator example that I could think of and that holds on its own but still has some basic ideas of why an operator could be helpful.

## Build

Just run `make`.

## Installing the CRD in a cluster

Point your Kubernetes context to the right cluster, and run `make install`

## Create a docker image and push it

```
IMG=<image-name>:<tag> make docker-build
IMG=<image-name>:<tag> make docker-push
```

## Deploy the operator

Once you have a docker image built and pushed to a repo that's available from the cluster, just run:

```
IMG=<image-name>:<tag> make deploy
```

## Create a CR

There is an example CR in the `samples` directory, just apply it directly.

```
kubectl apply -f config/samples/example_v1alpha1_presentation.yaml
```

## Access the present service

The service is of type `ClusterIP`, the easiest way to access is to port-forward the service, but you can also use an ingress, or change the type in the code to `LoadBalancer`, maybe create a CRD property out of it.

```
kubectl port-forward service/present  3999:80
```

## TODO

Add a second CRD for contents only. Every CR could represent a new *slide*, so the main CR would only hold parameters of the deployment.

Also I'm pretty sure that there are some bugs in the code, and there's always room for improvement so all contributions are welcome.