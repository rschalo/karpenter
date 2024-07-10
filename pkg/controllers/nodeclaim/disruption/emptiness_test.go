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

package disruption_test

import (
	"time"

	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/test"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "sigs.k8s.io/karpenter/pkg/test/expectations"
)

var _ = Describe("Emptiness", func() {
	var nodePool *v1.NodePool
	var nodeClaim *v1.NodeClaim
	var node *corev1.Node
	BeforeEach(func() {
		nodePool = test.NodePool()
		nodePool.Spec.Disruption.ConsolidationPolicy = v1.ConsolidationPolicyWhenEmpty
		nodePool.Spec.Disruption.ConsolidateAfter = &v1.NillableDuration{Duration: lo.ToPtr(time.Second * 30)}
		nodeClaim, node = test.NodeClaimAndNode(v1.NodeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					v1.NodePoolLabelKey:            nodePool.Name,
					corev1.LabelInstanceTypeStable: "default-instance-type", // need the instance type for the cluster state update
				},
			},
		})
	})
	Context("Metrics", func() {
		It("should fire a karpenter_nodeclaims_disrupted metric when empty", func() {
			ExpectApplied(ctx, env.Client, nodePool, nodeClaim, node)
			ExpectMakeNodeClaimsInitialized(ctx, env.Client, nodeClaim)

			ExpectObjectReconciled(ctx, env.Client, nodeClaimDisruptionController, nodeClaim)

			nodeClaim = ExpectExists(ctx, env.Client, nodeClaim)
			Expect(nodeClaim.StatusConditions().Get(v1.ConditionTypeEmpty).IsTrue()).To(BeTrue())

			metric, found := FindMetricWithLabelValues("karpenter_nodeclaims_disrupted", map[string]string{
				"type":     "emptiness",
				"nodepool": nodePool.Name,
			})
			Expect(found).To(BeTrue())
			Expect(metric.GetCounter().GetValue()).To(BeNumerically("==", 1))
		})
	})
	It("should mark NodeClaims as empty", func() {
		ExpectApplied(ctx, env.Client, nodePool, nodeClaim, node)
		ExpectMakeNodeClaimsInitialized(ctx, env.Client, nodeClaim)

		ExpectObjectReconciled(ctx, env.Client, nodeClaimDisruptionController, nodeClaim)

		nodeClaim = ExpectExists(ctx, env.Client, nodeClaim)
		Expect(nodeClaim.StatusConditions().Get(v1.ConditionTypeEmpty).IsTrue()).To(BeTrue())
	})
	It("should mark NodeClaims as empty that have only pods in terminating state", func() {
		rs := test.ReplicaSet()
		ExpectApplied(ctx, env.Client, rs)

		ExpectApplied(ctx, env.Client, nodePool, nodeClaim, node)
		ExpectMakeNodeClaimsInitialized(ctx, env.Client, nodeClaim)

		// Pod owned by a Deployment
		pods := test.Pods(3, test.PodOptions{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "apps/v1",
						Kind:               "ReplicaSet",
						Name:               rs.Name,
						UID:                rs.UID,
						Controller:         lo.ToPtr(true),
						BlockOwnerDeletion: lo.ToPtr(true),
					},
				},
			},
			NodeName:   node.Name,
			Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
		})
		ExpectApplied(ctx, env.Client, lo.Map(pods, func(p *corev1.Pod, _ int) client.Object { return p })...)

		for _, p := range pods {
			// Trigger an eviction to set the deletion timestamp but not delete the pod
			ExpectEvicted(ctx, env.Client, p)
			ExpectExists(ctx, env.Client, p)
		}

		ExpectObjectReconciled(ctx, env.Client, nodeClaimDisruptionController, nodeClaim)

		nodeClaim = ExpectExists(ctx, env.Client, nodeClaim)
		Expect(nodeClaim.StatusConditions().Get(v1.ConditionTypeEmpty).IsTrue()).To(BeTrue())
	})
	It("should mark NodeClaims as empty that have only DaemonSet pods", func() {
		ds := test.DaemonSet()
		ExpectApplied(ctx, env.Client, ds)

		ExpectApplied(ctx, env.Client, nodePool, nodeClaim, node)
		ExpectMakeNodeClaimsInitialized(ctx, env.Client, nodeClaim)

		// Pod owned by a DaemonSet
		pod := test.Pod(test.PodOptions{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "apps/v1",
						Kind:               "DaemonSet",
						Name:               ds.Name,
						UID:                ds.UID,
						Controller:         lo.ToPtr(true),
						BlockOwnerDeletion: lo.ToPtr(true),
					},
				},
			},
			NodeName:   node.Name,
			Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
		})
		ExpectApplied(ctx, env.Client, pod)

		ExpectObjectReconciled(ctx, env.Client, nodeClaimDisruptionController, nodeClaim)

		nodeClaim = ExpectExists(ctx, env.Client, nodeClaim)
		Expect(nodeClaim.StatusConditions().Get(v1.ConditionTypeEmpty).IsTrue()).To(BeTrue())
	})
	It("should remove the status condition from the nodeClaim when emptiness is disabled", func() {
		nodePool.Spec.Disruption.ConsolidateAfter.Duration = nil
		nodeClaim.StatusConditions().SetTrue(v1.ConditionTypeEmpty)
		ExpectApplied(ctx, env.Client, nodePool, nodeClaim, node)
		ExpectMakeNodeClaimsInitialized(ctx, env.Client, nodeClaim)

		ExpectObjectReconciled(ctx, env.Client, nodeClaimDisruptionController, nodeClaim)

		nodeClaim = ExpectExists(ctx, env.Client, nodeClaim)
		Expect(nodeClaim.StatusConditions().Get(v1.ConditionTypeEmpty)).To(BeNil())
	})
	It("should remove the status condition from the nodeClaim when the nodeClaim initialization condition is unknown", func() {
		nodeClaim.StatusConditions().SetTrue(v1.ConditionTypeEmpty)
		ExpectApplied(ctx, env.Client, nodePool, nodeClaim, node)
		ExpectMakeNodeClaimsInitialized(ctx, env.Client, nodeClaim)
		nodeClaim.StatusConditions().SetUnknown(v1.ConditionTypeInitialized)
		ExpectApplied(ctx, env.Client, nodeClaim)

		ExpectObjectReconciled(ctx, env.Client, nodeClaimDisruptionController, nodeClaim)

		nodeClaim = ExpectExists(ctx, env.Client, nodeClaim)
		Expect(nodeClaim.StatusConditions().Get(v1.ConditionTypeEmpty)).To(BeNil())
	})
	It("should remove the status condition from the nodeClaim when the nodeClaim initialization condition is false", func() {
		nodeClaim.StatusConditions().SetTrue(v1.ConditionTypeEmpty)
		ExpectApplied(ctx, env.Client, nodePool, nodeClaim, node)
		ExpectMakeNodeClaimsInitialized(ctx, env.Client, nodeClaim)
		nodeClaim.StatusConditions().SetFalse(v1.ConditionTypeInitialized, "NotInitialized", "NotInitialized")
		ExpectApplied(ctx, env.Client, nodeClaim)

		ExpectObjectReconciled(ctx, env.Client, nodeClaimDisruptionController, nodeClaim)

		nodeClaim = ExpectExists(ctx, env.Client, nodeClaim)
		Expect(nodeClaim.StatusConditions().Get(v1.ConditionTypeEmpty)).To(BeNil())
	})
	It("should remove the status condition from the nodeClaim when the node doesn't exist", func() {
		nodeClaim.StatusConditions().SetTrue(v1.ConditionTypeEmpty)
		ExpectApplied(ctx, env.Client, nodePool, nodeClaim)
		ExpectMakeNodeClaimsInitialized(ctx, env.Client, nodeClaim)

		ExpectObjectReconciled(ctx, env.Client, nodeClaimDisruptionController, nodeClaim)

		nodeClaim = ExpectExists(ctx, env.Client, nodeClaim)
		Expect(nodeClaim.StatusConditions().Get(v1.ConditionTypeEmpty)).To(BeNil())
	})
	It("should remove the status condition from non-empty NodeClaims", func() {
		nodeClaim.StatusConditions().SetTrue(v1.ConditionTypeEmpty)
		ExpectApplied(ctx, env.Client, nodePool, nodeClaim, node)
		ExpectMakeNodeClaimsInitialized(ctx, env.Client, nodeClaim)

		ExpectApplied(ctx, env.Client, test.Pod(test.PodOptions{
			NodeName:   node.Name,
			Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
		}))

		ExpectObjectReconciled(ctx, env.Client, nodeClaimDisruptionController, nodeClaim)

		nodeClaim = ExpectExists(ctx, env.Client, nodeClaim)
		Expect(nodeClaim.StatusConditions().Get(v1.ConditionTypeEmpty)).To(BeNil())
	})
	It("should remove the status condition from NodeClaims that have a StatefulSet pod in terminating state", func() {
		ss := test.StatefulSet()
		ExpectApplied(ctx, env.Client, ss)

		nodeClaim.StatusConditions().SetTrue(v1.ConditionTypeEmpty)
		ExpectApplied(ctx, env.Client, nodePool, nodeClaim, node)
		ExpectMakeNodeClaimsInitialized(ctx, env.Client, nodeClaim)

		// Pod owned by a StatefulSet
		pod := test.Pod(test.PodOptions{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "apps/v1",
						Kind:               "StatefulSet",
						Name:               ss.Name,
						UID:                ss.UID,
						Controller:         lo.ToPtr(true),
						BlockOwnerDeletion: lo.ToPtr(true),
					},
				},
			},
			NodeName:   node.Name,
			Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
		})
		ExpectApplied(ctx, env.Client, pod)

		// Trigger an eviction to set the deletion timestamp but not delete the pod
		ExpectEvicted(ctx, env.Client, pod)
		ExpectExists(ctx, env.Client, pod)

		// The node isn't empty even though it only has terminating pods
		ExpectObjectReconciled(ctx, env.Client, nodeClaimDisruptionController, nodeClaim)
		nodeClaim = ExpectExists(ctx, env.Client, nodeClaim)
		Expect(nodeClaim.StatusConditions().Get(v1.ConditionTypeEmpty)).To(BeNil())
	})
	It("should remove the status condition when the cluster state node is nominated", func() {
		nodeClaim.StatusConditions().SetTrue(v1.ConditionTypeEmpty)
		ExpectApplied(ctx, env.Client, nodePool, nodeClaim, node)
		ExpectMakeNodeClaimsInitialized(ctx, env.Client, nodeClaim)

		// Add the node to the cluster state and nominate it in the internal cluster state
		Expect(cluster.UpdateNode(ctx, node)).To(Succeed())
		cluster.NominateNodeForPod(ctx, node.Spec.ProviderID)

		result := ExpectObjectReconciled(ctx, env.Client, nodeClaimDisruptionController, nodeClaim)
		Expect(result.RequeueAfter).To(Equal(time.Second * 30))

		nodeClaim = ExpectExists(ctx, env.Client, nodeClaim)
		Expect(nodeClaim.StatusConditions().Get(v1.ConditionTypeEmpty)).To(BeNil())
	})
})
