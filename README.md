# kubettlreaper
A Kubernetes operator that uses time-to-live to enable time-bound objects.

## Description
It uses a single configMap to configure what Kinds of objects to check TTL for and the check interval.

There are no CRDs to install.

At every interval, the operator will get all resources matching a TTL label (`kubettlreaper.samir.io/ttl`) for each Kind configured to watch, and delete the resources if their TTL has expired (using creation timestamp + TTL as the calculation)

## Example ConfigMap to configure Kinds to check for TTL
- Configure group/version/kinds (GVKs) under `gvk-list` (all valid GVKs are supported)
- Configure the check interval under `check-interval`
- The configMap name must match the arg in the controller Deployment spec, i.e. - `- --configuration-name=kube-ttl-reaper`
```sh
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: kube-ttl-reaper
  namespace: kubettlreaper-system
data:
  check-interval: 30s
  gvk-list: |
    - group: ""
      version: "v1"
      kind: "Pod"
    - group: "apps"
      version: "v1"
      kind: "Deployment"
    - group: "rbac.authorization.k8s.io"
      version: "v1"
      kind: "RoleBinding"
EOF
```

## Example to configure a TTL on a RoleBinding object
- Add the label or create the object with the label and time value `kubettlreaper.samir.io/ttl`
```sh
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: pod-reader-binding
  namespace: default
  labels:
    kubettlreaper.samir.io/ttl: 1m  # This label is all that is required
subjects:
  - kind: ServiceAccount
    name: default
    namespace: default
roleRef:
  kind: Role
  name: pod-reader
  apiGroup: rbac.authorization.k8s.io
EOF
```

## Development

### Prerequisites
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/kubettlreaper:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/kubettlreaper:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

### Testing

**Run the controller in the foreground for testing:**
```sh
export OPERATOR_NAMESPACE=kubettlreaper-system
# run
make run
```

**Run integration tests using env test (without a real cluster):**
```sh
export OPERATOR_NAMESPACE=kubettlreaper-system
make test
```

**Generate coverage html report:**
```sh
go tool cover -html=cover.out -o coverage.html
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/kubettlreaper:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/kubettlreaper/<tag or branch>/dist/install.yaml
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

