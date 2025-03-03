[![Build Status](https://img.shields.io/github/actions/workflow/status/aws/karpenter-core/presubmit.yaml?branch=main)](https://github.com/aws/karpenter-core/actions/workflows/presubmit.yaml)
![GitHub stars](https://img.shields.io/github/stars/aws/karpenter-core)
![GitHub forks](https://img.shields.io/github/forks/aws/karpenter-core)
[![GitHub License](https://img.shields.io/badge/License-Apache%202.0-ff69b4.svg)](https://github.com/aws/karpenter-core/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/aws/karpenter-core)](https://goreportcard.com/report/github.com/aws/karpenter-core)
[![Coverage Status](https://coveralls.io/repos/github/aws/karpenter-core/badge.svg?branch=main)](https://coveralls.io/github/aws/karpenter-core?branch=main)
[![contributions welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)](https://github.com/aws/karpenter-core/issues)

# Karpenter

Karpenter improves the efficiency and cost of running workloads on Kubernetes clusters by:

* **Watching** for pods that the Kubernetes scheduler has marked as unschedulable
* **Evaluating** scheduling constraints (resource requests, nodeselectors, affinities, tolerations, and topology spread constraints) requested by the pods
* **Provisioning** nodes that meet the requirements of the pods
* **Removing** the nodes when the nodes are no longer needed

## Karpenter Implementations
Karpenter is a multi-cloud project with implementations by the following cloud providers:
- [AWS](https://github.com/aws/karpenter-provider-aws)
- [Azure](https://github.com/Azure/karpenter-provider-azure)
- [AlibabaCloud](https://github.com/cloudpilot-ai/karpenter-provider-alibabacloud)
- [Cluster API](https://github.com/kubernetes-sigs/karpenter-provider-cluster-api)
- [GCP](https://github.com/cloudpilot-ai/karpenter-provider-gcp)
- [Proxmox](https://github.com/sergelogvinov/karpenter-provider-proxmox)

## Community, discussion, contribution, and support

If you have any questions or want to get the latest project news, you can connect with us in the following ways:
- __Using and Deploying Karpenter?__ Reach out in the [#karpenter](https://kubernetes.slack.com/archives/C02SFFZSA2K) channel in the [Kubernetes slack](https://slack.k8s.io/) to ask questions about configuring or troubleshooting Karpenter.
- __Contributing to or Developing with Karpenter?__ Join the [#karpenter-dev](https://kubernetes.slack.com/archives/C04JW2J5J5P) channel in the [Kubernetes slack](https://slack.k8s.io/) to ask in-depth questions about contribution or to get involved in design discussions.

### Working Group Meetings
Bi-weekly meetings alternating between Thursdays @ 9:00 PT ([convert to your timezone](http://www.thetimezoneconverter.com/?t=9:00&tz=Seattle)) and Thursdays @ 15:00 PT ([convert to your timezone](http://www.thetimezoneconverter.com/?t=15:00&tz=Seattle))

### Issue Triage Meetings
Weekly meetings alternating between repositories and time slots. Please check the calendar invite for specific dates:

**kubernetes-sigs/karpenter**:
- Alternating Mondays @ 9:00 PT ([convert to your timezone](http://www.thetimezoneconverter.com/?t=9:00&tz=Seattle)) and @ 15:00 PT [convert to your timezone](http://www.thetimezoneconverter.com/?t=15:00&tz=Seattle) bi-weekly

**aws/karpenter-provider-aws**:
- Alternating Mondays @ 9:00 PT ([convert to your timezone](http://www.thetimezoneconverter.com/?t=9:00&tz=Seattle)) and @ 15:00 PT [convert to your timezone](http://www.thetimezoneconverter.com/?t=15:00&tz=Seattle) bi-weekly

#### Meeting Resources
- **Zoom Link**: [Join Meeting](https://zoom.us/j/95618088729) (password: 77777)
- **Calendar**: Subscribe to our [Google Calendar](https://calendar.google.com/calendar/u/0?cid=N3FmZGVvZjVoZWJkZjZpMnJrMmplZzVqYmtAZ3JvdXAuY2FsZW5kYXIuZ29vZ2xlLmNvbQ)
- **Meeting Notes**: View our [Working Group Log](https://docs.google.com/document/d/18BT0AIMugpNpiSPJNlcAL2rv69yAE6Z06gUVj7v_clg/edit?usp=sharing)

Pull Requests and feedback on issues are very welcome!
See the [issue tracker](https://github.com/aws/karpenter-core/issues) if you're unsure where to start, especially the [Good first issue](https://github.com/aws/karpenter-core/issues?q=is%3Aopen+is%3Aissue+label%3Agood-first-issue) and [Help wanted](https://github.com/aws/karpenter-core/issues?utf8=%E2%9C%93&q=is%3Aopen+is%3Aissue+label%3Ahelp-wanted) tags, and
also feel free to reach out to discuss.

See also our [contributor guide](CONTRIBUTING.md) and the Kubernetes [community page](https://kubernetes.io/community) for more details on how to get involved.

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

## Talks
- 09/08/2022 [Workload Consolidation with Karpenter](https://youtu.be/BnksdJ3oOEs)
- 05/19/2022 [Scaling K8s Nodes Without Breaking the Bank or Your Sanity](https://www.youtube.com/watch?v=UBb8wbfSc34)
- 03/25/2022 [Karpenter @ AWS Community Day 2022](https://youtu.be/sxDtmzbNHwE?t=3931)
- 12/20/2021 [How To Auto-Scale Kubernetes Clusters With Karpenter](https://youtu.be/C-2v7HT-uSA)
- 11/30/2021 [Karpenter vs Kubernetes Cluster Autoscaler](https://youtu.be/3QsVRHVdOnM)
- 11/19/2021 [Karpenter @ Container Day](https://youtu.be/qxWJRUF6JJc)
- 05/14/2021 [Groupless Autoscaling with Karpenter @ Kubecon](https://www.youtube.com/watch?v=43g8uPohTgc)
- 05/04/2021 [Karpenter @ Container Day](https://youtu.be/MZ-4HzOC_ac?t=7137)

# Karpenter Deployment Generator

This tool helps create and manage deployments and NodePools for testing Karpenter's node provisioning capabilities. It can create deployments with pod anti-affinity and topology spread constraints, as well as NodePools with exclusive requirements.

## Features

- Create deployments with pod anti-affinity to force pods onto different nodes
- Create deployments with topology spread constraints to distribute pods across availability zones
- Create NodePools with a diverse sampling of instance types and architectures
- Create deployments that target specific NodePools
- Delete deployments and NodePools
- Halve the number of replicas for existing deployments

## Prerequisites

- Kubernetes cluster with Karpenter installed
- `kubectl` configured to access your cluster
- Go 1.19+

## Usage

### Creating Deployments with Anti-Affinity

```bash
go run create_deployments.go --deployments=3 --replicas=5 --namespace=test
```

This creates 3 deployments in the "test" namespace, each with 5 replicas. The pods will have:
- Anti-affinity rules to prevent pods from the same deployment from being scheduled on the same node
- Topology spread constraints to ensure pods are distributed across at least 3 availability zones

### Creating NodePools with Diverse Instance Types and Architectures

```bash
go run create_deployments.go --create-nodepools --deployments=10 --replicas=3 --namespace=test --labels="disk-type=ssd,env=test"
```

This creates:
- Up to 10 NodePools with a diverse sampling of instance types and architectures
- The tool automatically selects from a variety of instance types (m5.large, c5.large, r5.large, m6g.large, etc.)
- Appropriate architecture is selected for each instance type (amd64 for x86 instances, arm64 for Graviton instances)
- Matching deployments that target these NodePools using node selectors

### Deleting Deployments

```bash
go run create_deployments.go --delete --deployments=3 --namespace=test
```

This deletes 3 deployments in the "test" namespace.

### Deleting NodePools

```bash
go run create_deployments.go --delete-nodepools --deployments=10
```

This deletes 10 NodePools.

### Halving Replicas

```bash
go run create_deployments.go --halve --namespace=test
```

This finds all deployments with the default prefix in the "test" namespace and halves their replicas (rounding up).

## Command-Line Options

| Option | Default | Description |
|--------|---------|-------------|
| `--deployments` | 1 | Number of deployments/NodePools to create or delete |
| `--replicas` | 3 | Number of replicas per deployment |
| `--namespace` | "default" | Kubernetes namespace to create/delete deployments in |
| `--prefix` | "antiaffinity" | Prefix for deployment names |
| `--delete` | false | Delete deployments instead of creating them |
| `--halve` | false | Find all deployments with the given prefix and halve their replicas |
| `--create-nodepools` | false | Create NodePools with diverse instance types and architectures |
| `--delete-nodepools` | false | Delete NodePools with the given prefix |
| `--nodepool-prefix` | "exclusive" | Prefix for NodePool names |
| `--labels` | "" | Custom labels for NodePool in format 'key1=value1,key2=value2' |
| `--kubeconfig` | "~/.kube/config" | Path to kubeconfig file |

## Testing Karpenter with this Tool

This tool is particularly useful for testing Karpenter's node provisioning capabilities:

1. **Testing Anti-Affinity**: By creating deployments with pod anti-affinity, you can force Karpenter to provision multiple nodes.

2. **Testing Topology Spread**: The topology spread constraints ensure pods are distributed across availability zones, testing Karpenter's ability to provision nodes in different zones.

3. **Testing Instance Type Diversity**: By creating NodePools with different instance types, you can test Karpenter's ability to provision a diverse set of node types.

4. **Testing Architecture Support**: The tool creates NodePools for both x86 (amd64) and ARM (arm64) architectures, testing Karpenter's multi-architecture support.

5. **Testing Scaling**: The ability to halve replicas allows testing of Karpenter's node consolidation capabilities.

## Example Workflow

1. Create a diverse set of NodePools:
   ```bash
   go run create_deployments.go --create-nodepools --deployments=10 --replicas=5 --namespace=test
   ```

2. Observe Karpenter provisioning different types of nodes for each NodePool.

3. Scale down deployments:
   ```bash
   go run create_deployments.go --halve --namespace=test
   ```

4. Observe Karpenter consolidating nodes.

5. Clean up:
   ```bash
   go run create_deployments.go --delete --deployments=10 --namespace=test --prefix=antiaffinity
   go run create_deployments.go --delete-nodepools --deployments=10 --nodepool-prefix=exclusive
   ```
