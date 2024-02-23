# Spin k8s plugin

A [Spin plugin](https://github.com/fermyon/spin-plugins) for interacting with Kubernetes.

## Install

Install the stable release:

```sh
spin plugins update
spin plugins install k8s
```

### Canary release

For the canary release:

```sh
spin plugins install --url https://github.com/spinkube/spin-plugin-k8s/releases/download/canary/k8s.json
```

The canary release may not be stable, with some features still in progress.

### Compiling from source

As an alternative to the plugin manager, you can download and manually install the plugin. Manual installation is
commonly used to test in-flight changes. For a user, it's better to install the plugin using Spin's plugin manager.

Ensure the `pluginify` plugin is installed:

```sh
spin plugins update
spin plugins install pluginify --yes
```

Compile and install the plugin:

```sh
make
make install
```

## Prerequisites

Ensure spin-operator is installed in your Kubernetes cluster. See the [spin-operator Quickstart
Guide](https://github.com/spinkube/spin-operator/blob/main/documentation/content/quickstart.md).

## Usage

Install the wasm32-wasi target for Rust:

```sh
rustup target add wasm32-wasi
```

Create a Spin application:

```sh
spin new --accept-defaults -t http-rust hello-rust
cd hello-rust
```

Compile the application:

```sh
spin build
```

Publish your application:

```sh
docker login
spin registry push bacongobbler/hello-rust:latest
```

Deploy to Kubernetes:

```sh
spin k8s scaffold --from bacongobbler/hello-rust:latest | kubectl create -f -
```

View your application:

```sh
kubectl get spinapps
```

`spin k8s scaffold` deploys two replicas by default. You can change this with the `--replicas` flag:

```sh
spin k8s scaffold --from bacongobbler/hello-rust:latest --replicas 3 | kubectl apply -f -
```

Delete the app:

```sh
kubectl delete spinapp hello-rust
```

### Autoscaler support

Autoscaler support can be enabled by setting `--autoscaler` and by setting a CPU limit and a memory limit.

```sh
spin k8s scaffold --from bacongobbler/hello-rust:latest --autoscaler hpa --cpu-limit 100m --memory-limit 128Mi
```

Setting min/max replicas:

```sh
spin k8s scaffold --from bacongobbler/hello-rust:latest --autoscaler hpa --cpu-limit 100m --memory-limit 128Mi --min-replicas 1 --max-replicas 10
```

CPU/memory limits and CPU/memory requests can be set together:

```sh
spin k8s scaffold --from bacongobbler/hello-rust:latest --autoscaler hpa --cpu-limit 100m --memory-limit 128Mi --cpu-request 50m --memory-request 64Mi
```

```text
IMPORTANT!
    CPU/memory requests are optional and will default to the CPU/memory limit if not set.
    CPU/memory requests must be lower than their respective CPU/memory limit.
```

Setting the target CPU utilization:

```sh
spin k8s scaffold --from bacongobbler/hello-rust:latest --autoscaler hpa --cpu-limit 100m --memory-limit 128Mi --autoscaler-target-cpu-utilization 50
```

Setting the target memory utilization:

```sh
spin k8s scaffold --from bacongobbler/hello-rust:latest --autoscaler hpa --cpu-limit 100m --memory-limit 128Mi --autoscaler-target-memory-utilization 50
```

KEDA support:

```sh
spin k8s scaffold --from bacongobbler/hello-rust:latest --autoscaler keda --cpu-limit 100m --memory-limit 128Mi
```
