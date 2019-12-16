# nifcloud-cloud-controller-manager

**nifcloud-cloud-controller-manager** is the [Kubernetes Cloud Controller Manager (CCM)](https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/) implementation for [NIFCLOUD](https://pfs.nifcloud.com/).

## Features

- Node Controller
- Node Lifecycle Controller
- Service Controller

## Requirements

* Set `--cloud-provider=external` to all kubelet in your cluster. **DO NOT** set `--cloud-provider` option to kube-apiserver and kube-controller-manager. (More information: https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/#running-cloud-controller-manager)
* Node name must be match the instance id.

## Installation

1. Edit `access_key_id` and `secret_access_key` Secret resource in `manifests/nifcloud-cloud-controller-manager.yaml`
1. `kubectl apply -f manifests/nifcloud-cloud-controller-manager.yaml`

## Example
### LoadBalancer

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:alpine
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: nginx
  labels:
    app: nginx
spec:
  type: LoadBalancer
  ports:
    - port: 80
      protocol: TCP
      targetPort: 80
  selector:
    app: nginx
```
