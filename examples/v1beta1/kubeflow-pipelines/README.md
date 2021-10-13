# Using Katib with Kubeflow Pipelines

The following examples show how to use Katib with
[Kubeflow Pipelines](https://github.com/kubeflow/pipelines).

You can find the Katib Component source code for the Kubeflow Pipelines
[here](https://github.com/kubeflow/pipelines/tree/master/components/kubeflow/katib-launcher).

## Prerequisites

You have to install the following Python SDK to run these examples:

- [`kfp`](https://pypi.org/project/kfp/) >= 1.8.4
- [`kubeflow-katib`](https://pypi.org/project/kubeflow-katib/) >= 0.12.0

## Multi-User Pipelines Setup

The Notebooks examples run Pipelines in multi-user mode and your Kubeflow Notebook
must have the appropriate `PodDefault` with the `pipelines.kubeflow.org` audience.

Please follow [this guide](https://www.kubeflow.org/docs/components/pipelines/sdk/connect-api/#multi-user-mode)
to give an access Kubeflow Notebook to run Kubeflow Pipelines.

## List of Examples

The following Pipelines are deployed from Kubeflow Notebook:

- [Kubeflow E2E MNIST](kubeflow-e2e-mnist.ipynb)

- [Katib Experiment with Early Stopping](early-stopping.ipynb)

The following Pipelines have to be compiled and uploaded to the Kubeflow Pipelines UI:

- [MPIJob Horovod](mpi-job-horovod.py)
