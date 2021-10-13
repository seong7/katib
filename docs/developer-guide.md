# Developer Guide

This developer guide is for people who want to contribute to the Katib project.
If you're interesting in using Katib in your machine learning project,
see the following user guides:

- [Concepts](https://www.kubeflow.org/docs/components/katib/overview/)
  in Katib, hyperparameter tuning, and neural architecture search.
- [Getting started with Katib](https://kubeflow.org/docs/components/katib/hyperparameter/).
- Detailed guide to [configuring and running a Katib
  experiment](https://kubeflow.org/docs/components/katib/experiment/).

## Requirements

- [Go](https://golang.org/) (1.17 or later)
- [Docker](https://docs.docker.com/) (17.05 or later)
- [Java](https://docs.oracle.com/javase/8/docs/technotes/guides/install/install_overview.html) (8 or later)
- [Python](https://www.python.org/) (3.7 or later)
- [kustomize](https://kustomize.io/) (4.0.5 or later)

## Build from source code

Check source code as follows:

```bash
make build REGISTRY=<image-registry> TAG=<image-tag>
```

To use your custom images for the Katib components, modify
[Kustomization file](https://github.com/kubeflow/katib/blob/master/manifests/v1beta1/installs/katib-standalone/kustomization.yaml)
and [Katib Config](https://github.com/kubeflow/katib/blob/master/manifests/v1beta1/components/controller/katib-config.yaml)

You can deploy Katib v1beta1 manifests into a Kubernetes cluster as follows:

```bash
make deploy
```

You can undeploy Katib v1beta1 manifests from a Kubernetes cluster as follows:

```bash
make undeploy
```

## Modify controller APIs

If you want to modify Katib controller APIs, you have to
generate deepcopy, clientset, listers, informers, open-api and Python SDK with the changed APIs.
You can update the necessary files as follows:

```bash
make generate
```

## Controller Flags

Below is a list of command-line flags accepted by Katib controller:

| Name                            | Type                      | Default   | Description                                                                                                            |
| ------------------------------- | ------------------------- | --------- | ---------------------------------------------------------------------------------------------------------------------- |
| enable-grpc-probe-in-suggestion | bool                      | true      | Enable grpc probe in suggestions                                                                                       |
| experiment-suggestion-name      | string                    | "default" | The implementation of suggestion interface in experiment controller                                                    |
| metrics-addr                    | string                    | ":8080"   | The address the metric endpoint binds to                                                                               |
| trial-resources                 | []schema.GroupVersionKind | null      | The list of resources that can be used as trial template, in the form: Kind.version.group (e.g. TFJob.v1.kubeflow.org) |
| webhook-inject-securitycontext  | bool                      | false     | Inject the securityContext of container[0] in the sidecar                                                              |
| webhook-port                    | int                       | 8443      | The port number to be used for admission webhook server                                                                |

## Workflow design

Please see [workflow-design.md](./workflow-design.md).

## Katib admission webhooks

Katib uses three [Kubernetes admission webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/).

1. `validator.experiment.katib.kubeflow.org` -
   [Validating admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#validatingadmissionwebhook)
   to validate the Katib Experiment before the creation.

1. `defaulter.experiment.katib.kubeflow.org` -
   [Mutating admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook)
   to set the [default values](../pkg/apis/controller/experiments/v1beta1/experiment_defaults.go)
   in the Katib Experiment before the creation.

1. `mutator.pod.katib.kubeflow.org` - Mutating admission webhook to inject the metrics
   collector sidecar container to the training pod. Learn more about the Katib's
   metrics collector in the
   [Kubeflow documentation](https://www.kubeflow.org/docs/components/katib/experiment/#metrics-collector).

You can find the YAMLs for the Katib webhooks
[here](../manifests/v1beta1/components/webhook/webhooks.yaml).

**Note:** If you are using a private Kubernetes cluster, you have to allow traffic
via `TCP:8443` by specifying the firewall rule and you have to update the master
plane CIDR source range to use the Katib webhooks

### Katib cert generator

Katib uses the custom `cert-generator` [Kubernetes Job](https://kubernetes.io/docs/concepts/workloads/controllers/job/)
to generate certificates for the webhooks.

Once Katib is deployed in the Kubernetes cluster, the `cert-generator` Job follows these steps:

- Generate the self-signed CA certificate and private key.

- Generate public certificate and private key signed with the key generated in the previous step.

- Create a Kubernetes Secret with the signed certificate. Secret has
  the `katib-webhook-cert` name and `cert-generator` Job's `ownerReference` to
  clean-up resources once Katib is uninstalled.

  Once Secret is created, the Katib controller Deployment spawns the Pod,
  since the controller has the `katib-webhook-cert` Secret volume.

- Patch the webhooks with the `CABundle`.

You can find the `cert-generator` source code [here](../cmd/cert-generator/v1beta1).

## Implement a new algorithm and use it in Katib

Please see [new-algorithm-service.md](./new-algorithm-service.md).

## Algorithm settings documentation

Please see [algorithm-settings.md](./algorithm-settings.md).

## Katib UI documentation

Please see [Katib UI README](https://github.com/kubeflow/katib/tree/master/pkg/ui/v1beta1).

## Design proposals

Please see [proposals](./proposals).
