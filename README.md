# nifcloud-cloud-controller-manager

**nifcloud-cloud-controller-manager** is the [Kubernetes Cloud Controller Manager (CCM)](https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/) implementation for [NIFCLOUD](https://pfs.nifcloud.com/) (**UNOFFICIAL**).

## Features

- Node Controller
- Node Lifecycle Controller
- Service Controller

## Requirements

- Set `--cloud-provider=external` to all kubelet in your cluster. **DO NOT** set `--cloud-provider` option to kube-apiserver and kube-controller-manager. (More information: https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/#running-cloud-controller-manager)
- Node name must be match the instance id.

## Installation

### Using helm

1. Create Secret resource with an NIFCLOUD access key id and secret access key.
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: nifcloud-cloud-controller-manager-secret
     namespace: kube-system
   stringData:
     access_key_id: ""
     secret_access_key: ""
   ```
2. Add helm repository.
   ```sh
   helm repo add nifcloud-cloud-controller-manager https://raw.githubusercontent.com/aokumasan/nifcloud-cloud-controller-manager/main/charts
   helm repo update
   ```
3. Install. (Please change the parameter `<REGION>` to your environment.)
   ```sh
   helm upgrade --install nifcloud-cloud-controller-manager nifcloud-cloud-controller-manager/nifcloud-cloud-controller-manager \
     --namespace kube-system \
     --set nifcloud.region=<REGION> \
     --set nifcloud.accessKeyId.secretName=nifcloud-cloud-controller-manager-secret \
     --set nifcloud.accessKeyId.key=access_key_id \
     --set nifcloud.secretAccessKey.secretName=nifcloud-cloud-controller-manager-secret \
     --set nifcloud.secretAccessKey.key=secret_access_key
   ```

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
