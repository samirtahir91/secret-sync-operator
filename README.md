[![Unit tests](https://github.com/samirtahir91/secret-sync-operator/actions/workflows/tests.yaml/badge.svg)](https://github.com/samirtahir91/secret-sync-operator/actions/workflows/tests.yaml)
[![Coverage Status](https://coveralls.io/repos/github/samirtahir91/secret-sync-operator/badge.svg)](https://coveralls.io/github/samirtahir91/secret-sync-operator)

# Secret-Sync-Operator
This is a Kubernetes operator that will sync secrets from a defined source namespace to any number of destination namespaces.

## Description
Key features:
- Uses a custom resource `SecretSync` in your destination namespace.
- Reads secrets defined in a `SecretSync` resource in a tenant namespace and syncs them from a source namespace - configured in the controller deployment env spec `SOURCE_NAMESPACE`.
    - `SOURCE_NAMESPACE` is set to `default`. It can be updated in the [manager deployment](config/manager/manager.yaml#L104)
    - The idea is for admins to configure the global source namespace via this ENV var.
- Allows for centralised secrets to by synced accross any tenant namespace, i.e. global credentials or certificates.
- Deleting the `SecretSync` object will also delete the secrets it owns.
- The operator will reconcile secrets that are defined in the `SecretSync.spec.secrets` list only, when:
    - Modifications are made to a source secret in the source namespace, that are referenced by any SecretSync objects in other namespaces.
    - Modifications are made to the secrets owned by the `SecretSync` object (i.e. a user manually update or deletes a secret owned by the `SecretSync` object)
    - Modifications are made to the `SecretSync` object (removing a secret from the list will force a delete of the secret in the destination namespace and vice versa)
- It will skip a secret syncing if the data already matches a source secret data.

## Example SecretSync object
Below example will setup a sync for `secret1` and `secret2` in the namespace `team-1`, assuming the controller has been configured with a source namespace `default`, the controller will reconcile the secrets into the `team-1` namespace (copying from the `default` namespace). \
Any changes to the secrets will trigger a sync/reconile to keep the secrets in sync with the `default` namespace.
```sh
kubectl apply -f - <<EOF
apiVersion: sync.samir.io/v1
kind: SecretSync
metadata:
  name: secretsync-sample
  namespace: team-1
spec:
  secrets:
    - secret1
    - secret2
EOF
```

> :warning: **Do not have duplicate secrets for `SecretSync` objects created in the same namespace**: It is recommended to have a single `SecretSync` object per namespace. \
> The controller will **not** check for this and can cause undesired behaviour if multiple `SecretSync` objects are attempting to sync the same secrets in the same namespace. \
> You could limit the count for SecretSync objects to 1 using a Resource Quota with the object quota, i.e. for namespace `team-1`
> ```yaml
> apiVersion: v1
>kind: ResourceQuota
>metadata:
>  name: secretsync-object-quota
>  namespace: team-1
>spec:
>  hard:
>    count/secretsyncs.sync.samir.io: 1
> ```

## Getting Started

### Prerequisites
- go version v1.20.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/secret-sync-operator:tag
```

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/secret-sync-operator:tag
```

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

### Testing

Current integration tests cover the scenarios:
- Creating `SecretSync` objects and checking the secrets are synced to a namespace
- Removing a secret from a `SecretSync` object and checking the secret has been deleted from a namespace
- Deleting a secret owned by a `SecretSync` object causes the reconilation of the secret to be re-created.
- Modifying a secret's data that is owned by a `SecretSync` object causes the reconcilation of the secret to be updated with the source secret's data.
- Modifying a source namespace secret referenced by one or more `SecretSync` object causes the reconcilation of the secret to be updated in affected namespaces.

**Run the controller in the foreground for testing:**
```sh
make run
```

**Run integration tests:**
```sh
make test
```

**Generate coverage html report:**
```sh
go tool cover -html=cover.out -o coverage.html
```

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

**NOTE:** Run `make --help` for more information on all potential `make` targets

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

