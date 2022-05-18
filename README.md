# Operator

[![Release](https://img.shields.io/github/release/ubogdan/operator.svg?style=flat-square)](https://github.com/ubogdan/operator/releases)

## Running the Operator
### Requirements

* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) (>= 1.20)
* Kubernetes distribution such as [kind](https://kind.sigs.k8s.io).

#### Setup

* Setup Kind with ingress controller enabled
```shell
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
EOF
```
 
* Setup Nginx ingress controller
```shell
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
```

* Setup Cert-Manager
```shell
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.8.0/cert-manager.yaml
```

* Edit email address in `samples/cert-manager.yaml` and apply configuration using the following command:
```shell
kubectl apply -f samples/cert-manager.yaml
```

* Deploy the operator
```shell
kubectl apply -f https://raw.githubusercontent.com/ubogdan/operator/main/deploy/static/provider/kind/deploy.yaml
```

## Setting up Your Development Environment
### Requirements

Before you start, install the following tools and packages:

* [go](https://golang.org/dl/) (>= 1.17)
* [golangci-lint](https://github.com/golangci/golangci-lint)
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) (>= 1.20)
* [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) (>= 3.4.0)
* [docker](https://docs.docker.com/) (>= 19.0.0)
* Kubernetes distribution such as [kind](https://kind.sigs.k8s.io).

#### Get sources

```bash
git clone https://github.com/ubogdan/operator.git
cd operator
```

#### Setup
* Setup Kind with ingress controller enabled
```shell
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
EOF
```
* Setup Nginx ingress controller
```shell
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
```

* Setup Cert-Manager
```shell
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.8.0/cert-manager.yaml
```
 
* Edit email address in `samples/cert-manager.yaml` and apply configuration using the following command:
```shell
kubectl apply -f samples/cert-manager.yaml
```
* Run the operator (development mode)
```shell
make install run
```

* Edit `samples/workloads_v1_container.yaml` and apply configuration using the following command:
```shell
kubectl apply -f samples/workloads_v1_container.yaml
```



### Recommended reading

* [Resources](https://book-v1.book.kubebuilder.io/basics/what_is_a_resource.html)
* [Controllers](https://book-v1.book.kubebuilder.io/basics/what_is_a_controller.html)
* [Controller Managers](https://book-v1.book.kubebuilder.io/basics/what_is_the_controller_manager.html)