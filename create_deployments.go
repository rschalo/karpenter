/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/samber/lo"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"sigs.k8s.io/karpenter/pkg/apis"
	v1 "sigs.k8s.io/karpenter/pkg/apis/v1"
)

func main() {
	// Define command-line flags
	var numDeployments, numReplicas int
	var namespace, namePrefix string
	var deleteDeployments, halveReplicas, createNodePools, deleteNodePools bool
	var nodePoolPrefix string
	var customLabels string
	var instanceType, architecture string

	flag.IntVar(&numDeployments, "deployments", 1, "Number of deployments to create/delete")
	flag.IntVar(&numReplicas, "replicas", 3, "Number of replicas per deployment (only for creation)")
	flag.StringVar(&namespace, "namespace", "default", "Kubernetes namespace to create/delete deployments in")
	flag.StringVar(&namePrefix, "prefix", "antiaffinity", "Prefix for deployment names")
	flag.BoolVar(&deleteDeployments, "delete", false, "Delete deployments instead of creating them")
	flag.BoolVar(&halveReplicas, "halve", false, "Find all deployments with the given prefix and halve their replicas (rounding up)")
	flag.BoolVar(&createNodePools, "create-nodepools", false, "Create NodePools with diverse instance types and architectures")
	flag.BoolVar(&deleteNodePools, "delete-nodepools", false, "Delete NodePools with the given prefix")
	flag.StringVar(&nodePoolPrefix, "nodepool-prefix", "exclusive", "Prefix for NodePool names")
	flag.StringVar(&instanceType, "instance-type", "", "Instance type for NodePool (e.g., m5.large)")
	flag.StringVar(&architecture, "arch", "amd64", "Architecture for NodePool (amd64 or arm64)")
	flag.StringVar(&customLabels, "labels", "", "Custom labels for NodePool in format 'key1=value1,key2=value2'")

	// No need to handle kubeconfig explicitly when using GetConfigOrDie
	flag.Parse()

	// Validate inputs
	if numDeployments < 1 && !halveReplicas && !deleteNodePools {
		log.Fatal("Number of deployments must be at least 1")
	}
	if numReplicas < 1 && !deleteDeployments && !halveReplicas && !deleteNodePools {
		log.Fatal("Number of replicas must be at least 1")
	}

	// Create context
	ctx := context.Background()

	// Register Karpenter types with the global scheme
	gv := schema.GroupVersion{Group: apis.Group, Version: "v1"}
	clientgoscheme.Scheme.AddKnownTypes(gv,
		&v1.NodePool{},
		&v1.NodePoolList{},
		&v1.NodeClaim{},
		&v1.NodeClaimList{})
	metav1.AddToGroupVersion(clientgoscheme.Scheme, gv)

	// Get Kubernetes config and create client using lo.Must
	runtimeClient := lo.Must(client.New(config.GetConfigOrDie(), client.Options{Scheme: clientgoscheme.Scheme}))

	if deleteNodePools {
		// Delete NodePools
		deleteNodePoolsWithPrefix(ctx, runtimeClient, nodePoolPrefix, numDeployments)
	} else if createNodePools {
		// Create NodePools and matching deployments
		createNodePoolsWithDeployments(ctx, runtimeClient, namespace, namePrefix, nodePoolPrefix,
			numDeployments, numReplicas, instanceType, architecture, customLabels)
	} else if halveReplicas {
		// Halve replicas for all deployments with the given prefix
		halveDeploymentReplicas(ctx, runtimeClient, namespace, namePrefix)
	} else if deleteDeployments {
		// Delete deployments
		deleteDeploymentsWithPrefix(ctx, runtimeClient, namespace, namePrefix, numDeployments)
	} else {
		// Create namespace if it doesn't exist
		ensureNamespaceExists(ctx, runtimeClient, namespace)

		// Create deployments
		createDeploymentsWithAntiAffinity(ctx, runtimeClient, namespace, namePrefix, numDeployments, numReplicas)
	}
}

// ensureNamespaceExists checks if a namespace exists and creates it if it doesn't
func ensureNamespaceExists(ctx context.Context, c client.Client, namespace string) {
	ns := &corev1.Namespace{}
	err := c.Get(ctx, types.NamespacedName{Name: namespace}, ns)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create namespace if it doesn't exist
			newNs := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			err = c.Create(ctx, newNs)
			if err != nil {
				log.Fatalf("Error creating namespace %s: %v", namespace, err)
			}
			fmt.Printf("Created namespace %s\n", namespace)
		} else {
			log.Fatalf("Error checking namespace %s: %v", namespace, err)
		}
	}
}

// createDeploymentsWithAntiAffinity creates deployments with pod anti-affinity
func createDeploymentsWithAntiAffinity(ctx context.Context, c client.Client, namespace, namePrefix string, numDeployments, numReplicas int) {
	// Create namespace if it doesn't exist
	ensureNamespaceExists(ctx, c, namespace)

	for i := 1; i <= numDeployments; i++ {
		name := fmt.Sprintf("%s-%d", namePrefix, i)

		// Check for int32 overflow
		if numReplicas > math.MaxInt32 {
			log.Fatalf("Number of replicas %d exceeds maximum value for int32", numReplicas)
		}

		// Create deployment with pod anti-affinity
		deployment := createDeploymentWithAntiAffinity(name, namespace, int32(numReplicas), nil)

		// Apply the deployment
		err := c.Create(ctx, deployment)
		if err != nil {
			log.Printf("Error creating deployment %s: %v", name, err)
			continue
		}

		fmt.Printf("Created deployment %s with %d replicas\n", name, numReplicas)
	}
}

// deleteDeploymentsWithPrefix deletes deployments with the given prefix
func deleteDeploymentsWithPrefix(ctx context.Context, c client.Client, namespace, namePrefix string, numDeployments int) {
	for i := 1; i <= numDeployments; i++ {
		name := fmt.Sprintf("%s-%d", namePrefix, i)

		// Delete the deployment
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}

		err := c.Delete(ctx, deployment)
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Printf("Error deleting deployment %s: %v", name, err)
			} else {
				fmt.Printf("Deployment %s not found\n", name)
			}
			continue
		}

		fmt.Printf("Deleted deployment %s\n", name)
	}
}

// halveDeploymentReplicas finds all deployments with the given prefix and halves their replicas
func halveDeploymentReplicas(ctx context.Context, c client.Client, namespace, namePrefix string) {
	// List all deployments in the namespace
	deploymentList := &appsv1.DeploymentList{}
	err := c.List(ctx, deploymentList, client.InNamespace(namespace))
	if err != nil {
		log.Fatalf("Error listing deployments: %v", err)
	}

	modifiedCount := 0

	// Iterate through deployments and modify those with the given prefix
	for _, deployment := range deploymentList.Items {
		if strings.HasPrefix(deployment.Name, namePrefix) {
			// Get current replicas
			currentReplicas := *deployment.Spec.Replicas

			// Calculate new replicas (half, rounded up)
			newReplicas := int32(math.Ceil(float64(currentReplicas) / 2.0))

			// Update the deployment
			deployment.Spec.Replicas = &newReplicas

			err := c.Update(ctx, &deployment)
			if err != nil {
				log.Printf("Error updating deployment %s: %v", deployment.Name, err)
				continue
			}

			fmt.Printf("Updated deployment %s: replicas %d -> %d\n", deployment.Name, currentReplicas, newReplicas)
			modifiedCount++
		}
	}

	if modifiedCount == 0 {
		fmt.Printf("No deployments with prefix '%s' found in namespace '%s'\n", namePrefix, namespace)
	} else {
		fmt.Printf("Successfully modified %d deployment(s)\n", modifiedCount)
	}
}

// createNodePoolsWithDeployments creates NodePools with exclusive requirements and matching deployments
func createNodePoolsWithDeployments(ctx context.Context, c client.Client,
	namespace, deploymentPrefix, nodePoolPrefix string, numNodePools, numReplicas int,
	instanceType, architecture string, customLabelsStr string) {

	// Create namespace if it doesn't exist
	ensureNamespaceExists(ctx, c, namespace)

	// Parse custom labels
	customLabels := parseLabels(customLabelsStr)

	// Define sample instance types and architectures if not specified
	instanceTypes := []string{instanceType}
	architectures := []string{architecture}

	if instanceType == "" {
		instanceTypes = []string{
			"m5.large", "m5.xlarge", "c5.large", "c5.xlarge",
			"r5.large", "r5.xlarge", "t3.medium", "t3.large",
			"m6g.large", "c6g.large", "r6g.large", "t4g.medium",
		}
	}

	if architecture == "" {
		architectures = []string{"amd64", "arm64"}
	}

	// Calculate how many NodePools to create for each combination
	totalCombinations := len(instanceTypes) * len(architectures)
	poolsPerCombination := numNodePools / totalCombinations
	if poolsPerCombination < 1 {
		poolsPerCombination = 1
	}

	nodePoolCount := 0

	// Loop through combinations of instance types and architectures
	for _, instanceType := range instanceTypes {
		for _, architecture := range architectures {
			// Skip arm64 for non-graviton instances
			if architecture == "arm64" && !strings.Contains(instanceType, "g.") {
				continue
			}

			// Skip amd64 for graviton instances
			if architecture == "amd64" && strings.Contains(instanceType, "g.") {
				continue
			}

			for i := 1; i <= poolsPerCombination; i++ {
				nodePoolCount++
				if nodePoolCount > numNodePools {
					return
				}

				nodePoolName := fmt.Sprintf("%s-%s-%s-%d", nodePoolPrefix, strings.Replace(instanceType, ".", "-", -1), architecture, i)
				deploymentName := fmt.Sprintf("%s-%s-%s-%d", deploymentPrefix, strings.Replace(instanceType, ".", "-", -1), architecture, i)

				// Create NodePool with exclusive requirements
				nodePool := createNodePool(nodePoolName, instanceType, architecture, customLabels)

				// Create the NodePool
				err := c.Create(ctx, nodePool)
				if err != nil {
					log.Printf("Error creating NodePool %s: %v", nodePoolName, err)
					continue
				}

				fmt.Printf("Created NodePool %s with instance type %s, architecture %s\n",
					nodePoolName, instanceType, architecture)

				// Create matching deployment with node affinity to target this NodePool
				nodeSelector := createNodeSelectorForNodePool(nodePool)

				// Check for int32 overflow
				if numReplicas > math.MaxInt32 {
					log.Fatalf("Number of replicas %d exceeds maximum value for int32", numReplicas)
				}

				// Create deployment with pod anti-affinity and node affinity for the NodePool
				deployment := createDeploymentWithAntiAffinity(deploymentName, namespace, int32(numReplicas), nodeSelector)

				// Apply the deployment
				err = c.Create(ctx, deployment)
				if err != nil {
					log.Printf("Error creating deployment %s: %v", deploymentName, err)
					continue
				}

				fmt.Printf("Created deployment %s with %d replicas targeting NodePool %s\n",
					deployment.Name, numReplicas, nodePoolName)
			}
		}
	}
}

// deleteNodePoolsWithPrefix deletes NodePools with the given prefix
func deleteNodePoolsWithPrefix(ctx context.Context, c client.Client, nodePoolPrefix string, numNodePools int) {
	for i := 1; i <= numNodePools; i++ {
		nodePoolName := fmt.Sprintf("%s-%d", nodePoolPrefix, i)

		// Create a NodePool object to delete
		nodePool := &v1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				Name: nodePoolName,
			},
		}

		// Delete the NodePool
		err := c.Delete(ctx, nodePool)
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Printf("Error deleting NodePool %s: %v", nodePoolName, err)
			} else {
				fmt.Printf("NodePool %s not found\n", nodePoolName)
			}
			continue
		}

		fmt.Printf("Deleted NodePool %s\n", nodePoolName)
	}
}

// createNodePool creates a NodePool with the specified requirements
func createNodePool(name, instanceType, architecture string, customLabels map[string]string) *v1.NodePool {
	requirements := []v1.NodeSelectorRequirementWithMinValues{}

	// Add instance type requirement if specified
	if instanceType != "" {
		requirements = append(requirements, v1.NodeSelectorRequirementWithMinValues{
			NodeSelectorRequirement: corev1.NodeSelectorRequirement{
				Key:      "node.kubernetes.io/instance-type",
				Operator: corev1.NodeSelectorOpIn,
				Values:   []string{instanceType},
			},
		})
	}

	// Add architecture requirement
	requirements = append(requirements, v1.NodeSelectorRequirementWithMinValues{
		NodeSelectorRequirement: corev1.NodeSelectorRequirement{
			Key:      "kubernetes.io/arch",
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{architecture},
		},
	})

	// Add custom requirements from labels
	for key, value := range customLabels {
		requirements = append(requirements, v1.NodeSelectorRequirementWithMinValues{
			NodeSelectorRequirement: corev1.NodeSelectorRequirement{
				Key:      key,
				Operator: corev1.NodeSelectorOpIn,
				Values:   []string{value},
			},
		})
	}

	return &v1.NodePool{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.NodePoolSpec{
			Template: v1.NodeClaimTemplate{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": name,
					},
				},
				Spec: v1.NodeClaimTemplateSpec{
					NodeClassRef: &v1.NodeClassReference{
						Kind:  "EC2NodeClass",
						Name:  "default",
						Group: "karpenter.k8s.aws",
					},
					Requirements: requirements,
				},
			},
		},
	}
}

// createNodeSelectorForNodePool creates a node selector to target a specific NodePool
func createNodeSelectorForNodePool(nodePool *v1.NodePool) map[string]string {
	// Create a node selector that targets nodes created by this NodePool
	return map[string]string{
		v1.NodePoolLabelKey: nodePool.Name,
	}
}

// parseLabels parses a comma-separated list of key=value pairs into a map
func parseLabels(labelsStr string) map[string]string {
	labels := make(map[string]string)
	if labelsStr == "" {
		return labels
	}

	pairs := strings.Split(labelsStr, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			if key != "" && value != "" {
				labels[key] = value
			}
		}
	}

	return labels
}

// createDeploymentWithAntiAffinity creates a deployment with pod anti-affinity
func createDeploymentWithAntiAffinity(name, namespace string, replicas int32, nodeSelector map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: corev1.PodSpec{
					NodeSelector: nodeSelector,
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
								{
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "app",
												Operator: metav1.LabelSelectorOpIn,
												Values:   []string{name},
											},
										},
									},
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
					TopologySpreadConstraints: []corev1.TopologySpreadConstraint{
						{
							MaxSkew:           1,
							TopologyKey:       "topology.kubernetes.io/zone",
							WhenUnsatisfiable: corev1.DoNotSchedule,
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app": name,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "pause",
							Image: "gcr.io/google_containers/pause:3.2",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("100Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("100Mi"),
								},
							},
						},
					},
				},
			},
		},
	}
}
