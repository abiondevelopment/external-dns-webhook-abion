# external-dns-webhook-abion

[ExternalDNS] makes Kubernetes resources discoverable via public DNS servers which allows you to control DNS records dynamically via Kubernetes 
resources in a DNS provider-agnostic way.
The external-dns-webhook-abion allows integrating [ExternalDNS] with [Abion] managed zones via the [Abion API].
[ExternalDNS] is, by default, aware of the records it is managing, therefore it can manage non-empty hosted zones. 
We strongly encourage you to set `--txt-owner-id` to a unique value that doesn't change for the lifetime of your cluster.

To be able to use external-dns-webhook-abion and manage your zones, you *must* have an Abion account to retrieve an Abion API key.
Contact [Abion] for help how to create an account and API key

# Docker images
Prebuilt docker images can be found on [Docker Hub]

# Release Docker images with GitHub Actions

The repository includes a GitHub Actions workflow at [.github/workflows/release-dockerhub.yml](.github/workflows/release-dockerhub.yml).
It publishes the Docker image to [Docker Hub] when a release tag is pushed, for example `1.0.3`.
The workflow only accepts tags matching `X.Y.Z`.

The pushed Docker tag is the Git tag name. For a tag such as `1.0.3`, the workflow pushes:

- `abiondevelopment/external-dns-webhook-abion:1.0.3`

For local builds without an explicit tag override, the `Makefile` default is `dev`.

# Install external-dns-webhook-abion using Helm
Install external-dns and external-dns-webhook-abion using helm by following the guide below. The Helm cart values for 
external-dns-webhook-abion can be found [here](deploy/external-dns-abion-values.yaml). For additional configurable helm chart values,
please check the [kubernetes-sigs helm chart] config and for Abion webhook specific configuration: [Environment variables](#environment-variables). 

    # Create Abion API key secret
    kubectl create secret generic abion-credentials --from-literal=api-key='<EXAMPLE_PLEASE_REPLACE>'

    # Install external-dns and external-dns-webhook-abion using helm 
    # Configure external-dns-abion-values according to requirements
    helm repo add external-dns https://kubernetes-sigs.github.io/external-dns/
    helm install external-dns-abion external-dns/external-dns -f deploy/external-dns-abion-values.yaml --version 1.14.3

# Environment variables

The following environment variables are available for Abion External DNS Webhook:

| Variable             | Description                                                                                                                                    | Notes                |
|----------------------|------------------------------------------------------------------------------------------------------------------------------------------------|----------------------|
| ABION_API_KEY        | ABION API key. You *must* have an Abion account to retrieve an API key. Contact [Abion] for help how to create an account and API key.         | Mandatory            |
| DOMAIN_FILTER        | Comma-separated list of zones to manage (e.g. `example.com,other.com`). Supports exact zone names and wildcard patterns (`*.example.com` matches any subdomain such as `sub.example.com` or `deep.sub.example.com`, but not `example.com` itself). Exact entries use a fast path that skips listing all zones; wildcard entries require fetching all zones to match against. If unset, all accessible zones are fetched. | Default: (empty)     |
| DRY_RUN              | If set, changes won't be applied.                                                                                                              | Default: `false`     | 
| ABION_DEBUG          | Enables webhook debug messages.                                                                                                                | Default: `false`     |  
| LOG_FORMAT           | Specifies log format for webhook. Supported values are `text` or `json`                                                                        | Default: `text`      |  
| SERVER_HOST          | Webhook hostname or IP address.                                                                                                                | Default: `localhost` |
| SERVER_PORT          | Webhook port.                                                                                                                                  | Default: `8888`      |
| SERVER_READ_TIMEOUT  | Webhook ReadTimeout is the maximum duration for reading the entire request. A zero or negative value means there will be no timeout.           | Default: 0           |
| SERVER_WRITE_TIMEOUT | Webhook WriteTimeout is the maximum duration before timing out writes of the response. A zero or negative value means there will be no timeout | Default: 0           |
| ABION_API_TIMEOUT    | HTTP client timeout for calls from the webhook to the Abion API (e.g. `30s`, `1m`). A zero or negative value disables the timeout.                | Default: `5s`       |


# Test external-dns-webhook-abion in Minikube
    
    # Start minikube 
    minikube start --memory=4G
    
    # Build external-dns-webhook-abion docker image
    eval $(minikube docker-env)
    make build

    # Create Abion API KEY secret
    kubectl create secret generic abion-credentials --from-literal=api-key='<EXAMPLE_PLEASE_REPLACE>'

    # Install external-dns and external-dns-webhook-abion using helm 
    helm repo add external-dns https://kubernetes-sigs.github.io/external-dns/
    helm install external-dns-abion external-dns/external-dns -f deploy/external-dns-abion-values.yaml --version 1.14.3

    # Test adding a service and verify the DNS records are configured as desired. 
    # Create tunnel to create a route to service deployed with type LoadBalancer within minikube
    minikube tunnel 

    # Make sure to update the `example.test.` value in the `external-dns.alpha.kubernetes.io/hostname` field for the example-service and make sure the apiKey/user has access to that zone. Then apply the service:   
    kubectl apply -f example/example-service.yaml

    # Go to Abion Core (https://app.abion.com) and login to verify the new records has been added. 

    # Remove the service and verify the records are removed
    kubectl delete service example-service

    # Go to Abion Core and login and verify the records have been removed from the zone. 
    
    # Uninstall external dns
    helm uninstall external-dns-abion


[Abion]: https://abion.com/
[Abion API]: https://demo.abion.com/pmapi-doc
[Abion Core]: https://app.abion.com
[Docker Hub]: https://hub.docker.com/r/abiondevelopment/external-dns-webhook-abion
[ExternalDNS]: https://github.com/kubernetes-sigs/external-dns
[kubernetes-sigs helm chart]: https://github.com/kubernetes-sigs/external-dns/blob/master/charts/external-dns/README.md#values
