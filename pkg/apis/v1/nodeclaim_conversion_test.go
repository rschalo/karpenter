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

package v1_test

import (
	"encoding/json"
	"time"

	"github.com/awslabs/operatorpkg/object"
	"github.com/awslabs/operatorpkg/status"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/karpenter/pkg/test/v1alpha1"

	. "sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/apis/v1beta1"
	"sigs.k8s.io/karpenter/pkg/operator/injection"
	"sigs.k8s.io/karpenter/pkg/test"
)

var _ = Describe("Convert v1 to v1beta1 NodeClaim API", func() {
	var (
		v1nodepool       *NodePool
		v1beta1nodepool  *v1beta1.NodePool
		v1nodeclaim      *NodeClaim
		v1beta1nodeclaim *v1beta1.NodeClaim
	)

	BeforeEach(func() {
		v1nodepool = test.NodePool()
		v1beta1nodepool = &v1beta1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-nodepool",
			},
			Spec: v1beta1.NodePoolSpec{
				Template: v1beta1.NodeClaimTemplate{
					Spec: v1beta1.NodeClaimSpec{
						NodeClassRef: &v1beta1.NodeClassReference{
							Name:       "test",
							Kind:       "test-kind",
							APIVersion: "test-group",
						},
						Requirements: []v1beta1.NodeSelectorRequirementWithMinValues{},
					},
				},
			},
		}
		v1nodeclaim = &NodeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					NodePoolLabelKey: v1nodepool.Name,
				},
			},
			Spec: NodeClaimSpec{
				NodeClassRef: &NodeClassReference{
					Name:  "test",
					Kind:  "test-kind",
					Group: "test-group",
				},
			},
		}
		v1beta1nodeclaim = &v1beta1.NodeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					NodePoolLabelKey: v1beta1nodepool.Name,
				},
			},
			Spec: v1beta1.NodeClaimSpec{
				NodeClassRef: &v1beta1.NodeClassReference{
					Name:       "test",
					Kind:       "test-kind",
					APIVersion: "test-group/test-version",
				},
			},
		}
		Expect(env.Client.Create(ctx, v1nodepool)).To(Succeed())
		gvk := lo.Map(cloudProvider.GetSupportedNodeClasses(), func(nc status.Object, _ int) schema.GroupVersionKind {
			return object.GVK(nc)
		})
		cloudProvider.NodeClassGroupVersionKind = gvk
		ctx = injection.WithNodeClasses(ctx, gvk)
	})

	It("should convert v1 nodeclaim metadata", func() {
		v1nodeclaim.ObjectMeta = test.ObjectMeta()
		Expect(v1nodeclaim.ConvertFrom(ctx, v1beta1nodeclaim)).To(Succeed())
		v1beta1nodeclaim.Annotations = lo.Assign(v1beta1nodeclaim.Annotations, map[string]string{KubeletCompatabilityAnnotationKey: "null"})
		Expect(v1beta1nodeclaim.ObjectMeta).To(BeEquivalentTo(v1nodeclaim.ObjectMeta))
	})
	Context("NodeClaim Spec", func() {
		It("should convert v1 nodeclaim taints", func() {
			v1nodeclaim.Spec.Taints = []v1.Taint{
				{
					Key:    "test-key-1",
					Value:  "test-value-1",
					Effect: v1.TaintEffectNoExecute,
				},
				{
					Key:    "test-key-2",
					Value:  "test-value-2",
					Effect: v1.TaintEffectNoSchedule,
				},
			}
			Expect(v1nodeclaim.ConvertTo(ctx, v1beta1nodeclaim)).To(Succeed())
			for i := range v1nodeclaim.Spec.Taints {
				Expect(v1beta1nodeclaim.Spec.Taints[i].Key).To(Equal(v1nodeclaim.Spec.Taints[i].Key))
				Expect(v1beta1nodeclaim.Spec.Taints[i].Value).To(Equal(v1nodeclaim.Spec.Taints[i].Value))
				Expect(v1beta1nodeclaim.Spec.Taints[i].Effect).To(Equal(v1nodeclaim.Spec.Taints[i].Effect))
			}
		})
		It("should convert v1 nodeclaim startup taints", func() {
			v1nodeclaim.Spec.StartupTaints = []v1.Taint{
				{
					Key:    "test-key-startup-1",
					Value:  "test-value-startup-1",
					Effect: v1.TaintEffectNoExecute,
				},
				{
					Key:    "test-key-startup-2",
					Value:  "test-value-startup-2",
					Effect: v1.TaintEffectNoSchedule,
				},
			}
			Expect(v1nodeclaim.ConvertTo(ctx, v1beta1nodeclaim)).To(Succeed())
			for i := range v1nodeclaim.Spec.StartupTaints {
				Expect(v1beta1nodeclaim.Spec.StartupTaints[i].Key).To(Equal(v1nodeclaim.Spec.StartupTaints[i].Key))
				Expect(v1beta1nodeclaim.Spec.StartupTaints[i].Value).To(Equal(v1nodeclaim.Spec.StartupTaints[i].Value))
				Expect(v1beta1nodeclaim.Spec.StartupTaints[i].Effect).To(Equal(v1nodeclaim.Spec.StartupTaints[i].Effect))
			}
		})
		It("should convert v1 nodeclaim requirements", func() {
			v1nodeclaim.Spec.Requirements = []NodeSelectorRequirementWithMinValues{
				{
					NodeSelectorRequirement: v1.NodeSelectorRequirement{
						Key:      v1.LabelArchStable,
						Operator: v1.NodeSelectorOpExists,
					},
					MinValues: lo.ToPtr(451613),
				},
				{
					NodeSelectorRequirement: v1.NodeSelectorRequirement{
						Key:      CapacityTypeLabelKey,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{CapacityTypeSpot},
					},
					MinValues: lo.ToPtr(9787513),
				},
			}
			Expect(v1nodeclaim.ConvertTo(ctx, v1beta1nodeclaim)).To(Succeed())
			for i := range v1nodeclaim.Spec.Requirements {
				Expect(v1beta1nodeclaim.Spec.Requirements[i].Key).To(Equal(v1nodeclaim.Spec.Requirements[i].Key))
				Expect(v1beta1nodeclaim.Spec.Requirements[i].Operator).To(Equal(v1nodeclaim.Spec.Requirements[i].Operator))
				Expect(v1beta1nodeclaim.Spec.Requirements[i].Values).To(Equal(v1nodeclaim.Spec.Requirements[i].Values))
				Expect(v1beta1nodeclaim.Spec.Requirements[i].MinValues).To(Equal(v1nodeclaim.Spec.Requirements[i].MinValues))
			}
		})
		It("should convert v1 nodeclaim resources", func() {
			v1nodeclaim.Spec.Resources = ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("1"),
					v1.ResourceMemory: resource.MustParse("134G"),
				},
			}
			Expect(v1nodeclaim.ConvertTo(ctx, v1beta1nodeclaim)).To(Succeed())
			for key := range v1nodeclaim.Spec.Resources.Requests {
				Expect(v1nodeclaim.Spec.Resources.Requests[key]).To(Equal(v1beta1nodeclaim.Spec.Resources.Requests[key]))
			}
		})
		Context("NodeClassRef", func() {
			It("should convert v1 nodeclaim template nodeClassRef", func() {
				v1nodeclaim.Spec.NodeClassRef = &NodeClassReference{
					Kind:  object.GVK(&v1alpha1.TestNodeClass{}).Kind,
					Name:  "nodeclass-test",
					Group: object.GVK(&v1alpha1.TestNodeClass{}).Group,
				}
				Expect(v1nodeclaim.ConvertTo(ctx, v1beta1nodeclaim)).To(Succeed())
				Expect(v1beta1nodeclaim.Spec.NodeClassRef.Kind).To(Equal(v1nodeclaim.Spec.NodeClassRef.Kind))
				Expect(v1beta1nodeclaim.Spec.NodeClassRef.Name).To(Equal(v1nodeclaim.Spec.NodeClassRef.Name))
				Expect(v1beta1nodeclaim.Spec.NodeClassRef.APIVersion).To(Equal(cloudProvider.NodeClassGroupVersionKind[0].GroupVersion().String()))
			})
			It("should not include APIVersion for v1beta1 if Group and Kind is not in the supported nodeclass", func() {
				v1nodeclaim.Spec.NodeClassRef = &NodeClassReference{
					Kind:  "test-kind",
					Name:  "nodeclass-test",
					Group: "testgroup.sh",
				}
				Expect(v1nodeclaim.ConvertTo(ctx, v1beta1nodeclaim)).To(Succeed())
				Expect(v1beta1nodeclaim.Spec.NodeClassRef.Kind).To(Equal(v1nodeclaim.Spec.NodeClassRef.Kind))
				Expect(v1beta1nodeclaim.Spec.NodeClassRef.Name).To(Equal(v1nodeclaim.Spec.NodeClassRef.Name))
				Expect(v1beta1nodeclaim.Spec.NodeClassRef.APIVersion).To(Equal(""))
			})
		})
	})
	Context("NodeClaim Status", func() {
		It("should convert v1 nodeclaim nodename", func() {
			v1nodeclaim.Status.NodeName = "test-node-name"
			Expect(v1nodeclaim.ConvertTo(ctx, v1beta1nodeclaim)).To(Succeed())
			Expect(v1nodeclaim.Status.NodeName).To(Equal(v1beta1nodeclaim.Status.NodeName))
		})
		It("should convert v1 nodeclaim provider id", func() {
			v1nodeclaim.Status.ProviderID = "test-provider-id"
			Expect(v1nodeclaim.ConvertTo(ctx, v1beta1nodeclaim)).To(Succeed())
			Expect(v1nodeclaim.Status.ProviderID).To(Equal(v1beta1nodeclaim.Status.ProviderID))
		})
		It("should convert v1 nodeclaim image id", func() {
			v1nodeclaim.Status.ImageID = "test-image-id"
			Expect(v1nodeclaim.ConvertTo(ctx, v1beta1nodeclaim)).To(Succeed())
			Expect(v1nodeclaim.Status.ImageID).To(Equal(v1beta1nodeclaim.Status.ImageID))
		})
		It("should convert v1 nodeclaim capacity", func() {
			v1nodeclaim.Status.Capacity = v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("13432"),
				v1.ResourceMemory: resource.MustParse("1332G"),
			}
			Expect(v1nodeclaim.ConvertTo(ctx, v1beta1nodeclaim)).To(Succeed())
			Expect(v1nodeclaim.Status.Capacity).To(Equal(v1beta1nodeclaim.Status.Capacity))
		})
		It("should convert v1 nodeclaim allocatable", func() {
			v1nodeclaim.Status.Allocatable = v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("13432"),
				v1.ResourceMemory: resource.MustParse("1332G"),
			}
			Expect(v1nodeclaim.ConvertTo(ctx, v1beta1nodeclaim)).To(Succeed())
			Expect(v1nodeclaim.Status.Allocatable).To(Equal(v1beta1nodeclaim.Status.Allocatable))
		})
		It("should convert v1 nodeclaim conditions", func() {
			v1nodeclaim.Status.Conditions = []status.Condition{
				{
					Status: status.ConditionReady,
					Reason: "test-reason",
				},
				{
					Status: ConditionTypeDrifted,
					Reason: "test-reason",
				},
			}
			Expect(v1nodeclaim.ConvertTo(ctx, v1beta1nodeclaim)).To(Succeed())
			Expect(v1nodeclaim.Status.Conditions).To(Equal(v1beta1nodeclaim.Status.Conditions))
		})
	})
})

var _ = Describe("Convert V1beta1 to V1 NodeClaim API", func() {
	var (
		v1nodepool       *NodePool
		v1beta1nodepool  *v1beta1.NodePool
		v1nodeclaim      *NodeClaim
		v1beta1nodeclaim *v1beta1.NodeClaim
	)

	BeforeEach(func() {
		v1beta1nodepool = &v1beta1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-nodepool",
			},
			Spec: v1beta1.NodePoolSpec{
				Template: v1beta1.NodeClaimTemplate{
					Spec: v1beta1.NodeClaimSpec{
						NodeClassRef: &v1beta1.NodeClassReference{
							Name:       "test",
							Kind:       "test-kind",
							APIVersion: "test-group",
						},
						Requirements: []v1beta1.NodeSelectorRequirementWithMinValues{},
					},
				},
			},
		}
		v1nodepool = test.NodePool()
		v1nodeclaim = &NodeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					NodePoolLabelKey: v1nodepool.Name,
				},
			},
			Spec: NodeClaimSpec{
				NodeClassRef: &NodeClassReference{
					Name:  "test",
					Kind:  "test-kind",
					Group: "test-group",
				},
			},
		}
		v1beta1nodeclaim = &v1beta1.NodeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					NodePoolLabelKey: v1beta1nodepool.Name,
				},
			},
			Spec: v1beta1.NodeClaimSpec{
				NodeClassRef: &v1beta1.NodeClassReference{
					Name:       "test",
					Kind:       "test-kind",
					APIVersion: "test-group/test-version",
				},
			},
		}
		Expect(env.Client.Create(ctx, v1beta1nodepool)).To(Succeed())
		gvk := lo.Map(cloudProvider.GetSupportedNodeClasses(), func(nc status.Object, _ int) schema.GroupVersionKind {
			return object.GVK(nc)
		})
		cloudProvider.NodeClassGroupVersionKind = gvk
		ctx = injection.WithNodeClasses(ctx, gvk)
	})

	It("should convert v1beta1 nodeclaim metadata", func() {
		v1beta1nodeclaim.ObjectMeta = test.ObjectMeta()
		Expect(v1nodeclaim.ConvertFrom(ctx, v1beta1nodeclaim)).To(Succeed())
		v1beta1nodeclaim.Annotations = lo.Assign(v1beta1nodeclaim.Annotations, map[string]string{KubeletCompatabilityAnnotationKey: "null"})
		Expect(v1nodeclaim.ObjectMeta).To(BeEquivalentTo(v1beta1nodeclaim.ObjectMeta))
	})
	Context("NodeClaim Spec", func() {
		It("should convert v1beta1 nodeclaim taints", func() {
			v1beta1nodeclaim.Spec.Taints = []v1.Taint{
				{
					Key:    "test-key-1",
					Value:  "test-value-1",
					Effect: v1.TaintEffectNoExecute,
				},
				{
					Key:    "test-key-2",
					Value:  "test-value-2",
					Effect: v1.TaintEffectNoSchedule,
				},
			}
			Expect(v1nodeclaim.ConvertFrom(ctx, v1beta1nodeclaim)).To(Succeed())
			for i := range v1beta1nodeclaim.Spec.Taints {
				Expect(v1nodeclaim.Spec.Taints[i].Key).To(Equal(v1beta1nodeclaim.Spec.Taints[i].Key))
				Expect(v1nodeclaim.Spec.Taints[i].Value).To(Equal(v1beta1nodeclaim.Spec.Taints[i].Value))
				Expect(v1nodeclaim.Spec.Taints[i].Effect).To(Equal(v1beta1nodeclaim.Spec.Taints[i].Effect))
			}
		})
		It("should convert v1beta1 nodeclaim startup taints", func() {
			v1beta1nodeclaim.Spec.StartupTaints = []v1.Taint{
				{
					Key:    "test-key-startup-1",
					Value:  "test-value-startup-1",
					Effect: v1.TaintEffectNoExecute,
				},
				{
					Key:    "test-key-startup-2",
					Value:  "test-value-startup-2",
					Effect: v1.TaintEffectNoSchedule,
				},
			}
			Expect(v1nodeclaim.ConvertFrom(ctx, v1beta1nodeclaim)).To(Succeed())
			for i := range v1beta1nodeclaim.Spec.StartupTaints {
				Expect(v1nodeclaim.Spec.StartupTaints[i].Key).To(Equal(v1beta1nodeclaim.Spec.StartupTaints[i].Key))
				Expect(v1nodeclaim.Spec.StartupTaints[i].Value).To(Equal(v1beta1nodeclaim.Spec.StartupTaints[i].Value))
				Expect(v1nodeclaim.Spec.StartupTaints[i].Effect).To(Equal(v1beta1nodeclaim.Spec.StartupTaints[i].Effect))
			}
		})
		It("should convert v1beta1 nodeclaim requirements", func() {
			v1beta1nodeclaim.Spec.Requirements = []v1beta1.NodeSelectorRequirementWithMinValues{
				{
					NodeSelectorRequirement: v1.NodeSelectorRequirement{
						Key:      v1.LabelArchStable,
						Operator: v1.NodeSelectorOpExists,
					},
					MinValues: lo.ToPtr(4189133),
				},
				{
					NodeSelectorRequirement: v1.NodeSelectorRequirement{
						Key:      CapacityTypeLabelKey,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{CapacityTypeSpot},
					},
					MinValues: lo.ToPtr(7716191),
				},
			}
			Expect(v1nodeclaim.ConvertFrom(ctx, v1beta1nodeclaim)).To(Succeed())
			for i := range v1beta1nodeclaim.Spec.Requirements {
				Expect(v1nodeclaim.Spec.Requirements[i].Key).To(Equal(v1beta1nodeclaim.Spec.Requirements[i].Key))
				Expect(v1nodeclaim.Spec.Requirements[i].Operator).To(Equal(v1beta1nodeclaim.Spec.Requirements[i].Operator))
				Expect(v1nodeclaim.Spec.Requirements[i].Values).To(Equal(v1beta1nodeclaim.Spec.Requirements[i].Values))
				Expect(v1nodeclaim.Spec.Requirements[i].MinValues).To(Equal(v1beta1nodeclaim.Spec.Requirements[i].MinValues))
			}
		})
		It("should convert v1beta1 nodeclaim resources", func() {
			v1beta1nodeclaim.Spec.Resources = v1beta1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("1"),
					v1.ResourceMemory: resource.MustParse("134G"),
				},
			}
			Expect(v1nodeclaim.ConvertFrom(ctx, v1beta1nodeclaim)).To(Succeed())
			for key := range v1beta1nodeclaim.Spec.Resources.Requests {
				Expect(v1beta1nodeclaim.Spec.Resources.Requests[key]).To(Equal(v1nodeclaim.Spec.Resources.Requests[key]))
			}
		})
		It("should convert v1 nodeclaim template kubelet", func() {
			v1beta1nodeclaim.Spec.Kubelet = &v1beta1.KubeletConfiguration{
				ClusterDNS:                  []string{"test-cluster-dns"},
				MaxPods:                     lo.ToPtr(int32(9383)),
				PodsPerCore:                 lo.ToPtr(int32(9334283)),
				SystemReserved:              map[string]string{"system-key": "reserved"},
				KubeReserved:                map[string]string{"kube-key": "reserved"},
				EvictionHard:                map[string]string{"eviction-key": "eviction"},
				EvictionSoft:                map[string]string{"eviction-key": "eviction"},
				EvictionSoftGracePeriod:     map[string]metav1.Duration{"test-soft-grace": {Duration: time.Hour}},
				EvictionMaxPodGracePeriod:   lo.ToPtr(int32(382902)),
				ImageGCHighThresholdPercent: lo.ToPtr(int32(382902)),
				CPUCFSQuota:                 lo.ToPtr(false),
			}
			Expect(v1nodeclaim.Annotations).To(BeNil())
			Expect(v1nodeclaim.ConvertFrom(ctx, v1beta1nodeclaim)).To(Succeed())
			kubelet := &v1beta1.KubeletConfiguration{}
			kubeletString, found := v1nodeclaim.Annotations[KubeletCompatabilityAnnotationKey]
			Expect(found).To(BeTrue())
			err := json.Unmarshal([]byte(kubeletString), kubelet)
			Expect(err).To(BeNil())
			Expect(kubelet.ClusterDNS).To(Equal(v1beta1nodeclaim.Spec.Kubelet.ClusterDNS))
			Expect(lo.FromPtr(kubelet.MaxPods)).To(Equal(lo.FromPtr(v1beta1nodeclaim.Spec.Kubelet.MaxPods)))
			Expect(lo.FromPtr(kubelet.PodsPerCore)).To(Equal(lo.FromPtr(v1beta1nodeclaim.Spec.Kubelet.PodsPerCore)))
			Expect(lo.FromPtr(kubelet.EvictionMaxPodGracePeriod)).To(Equal(lo.FromPtr(v1beta1nodeclaim.Spec.Kubelet.EvictionMaxPodGracePeriod)))
			Expect(lo.FromPtr(kubelet.ImageGCHighThresholdPercent)).To(Equal(lo.FromPtr(v1beta1nodeclaim.Spec.Kubelet.ImageGCHighThresholdPercent)))
			Expect(lo.FromPtr(kubelet.ImageGCHighThresholdPercent)).To(Equal(lo.FromPtr(v1beta1nodeclaim.Spec.Kubelet.ImageGCHighThresholdPercent)))
			Expect(lo.FromPtr(kubelet.ImageGCHighThresholdPercent)).To(Equal(lo.FromPtr(v1beta1nodeclaim.Spec.Kubelet.ImageGCHighThresholdPercent)))
			Expect(lo.FromPtr(kubelet.ImageGCHighThresholdPercent)).To(Equal(lo.FromPtr(v1beta1nodeclaim.Spec.Kubelet.ImageGCHighThresholdPercent)))
			Expect(kubelet.SystemReserved).To(Equal(v1beta1nodeclaim.Spec.Kubelet.SystemReserved))
			Expect(kubelet.KubeReserved).To(Equal(v1beta1nodeclaim.Spec.Kubelet.KubeReserved))
			Expect(kubelet.EvictionHard).To(Equal(v1beta1nodeclaim.Spec.Kubelet.EvictionHard))
			Expect(kubelet.EvictionSoft).To(Equal(v1beta1nodeclaim.Spec.Kubelet.EvictionSoft))
			Expect(kubelet.EvictionSoftGracePeriod).To(Equal(v1beta1nodeclaim.Spec.Kubelet.EvictionSoftGracePeriod))
			Expect(lo.FromPtr(kubelet.CPUCFSQuota)).To(Equal(lo.FromPtr(v1beta1nodeclaim.Spec.Kubelet.CPUCFSQuota)))
		})
		Context("NodeClassRef", func() {
			It("should convert v1beta1 nodeclaim template nodeClassRef", func() {
				v1beta1nodeclaim.Spec.NodeClassRef = &v1beta1.NodeClassReference{
					Kind:       "test-kind",
					Name:       "nodeclass-test",
					APIVersion: "testgroup.sh/testversion",
				}
				Expect(v1nodeclaim.ConvertFrom(ctx, v1beta1nodeclaim)).To(Succeed())
				Expect(v1nodeclaim.Spec.NodeClassRef.Kind).To(Equal(v1beta1nodeclaim.Spec.NodeClassRef.Kind))
				Expect(v1nodeclaim.Spec.NodeClassRef.Name).To(Equal(v1beta1nodeclaim.Spec.NodeClassRef.Name))
				Expect(v1nodeclaim.Spec.NodeClassRef.Group).To(Equal("testgroup.sh"))
			})
			It("should set default nodeclass group and kind on v1beta1 nodeclassRef", func() {
				v1beta1nodeclaim.Spec.NodeClassRef = &v1beta1.NodeClassReference{
					Name: "nodeclass-test",
				}
				Expect(v1nodeclaim.ConvertFrom(ctx, v1beta1nodeclaim)).To(Succeed())
				Expect(v1nodeclaim.Spec.NodeClassRef.Kind).To(Equal(cloudProvider.NodeClassGroupVersionKind[0].Kind))
				Expect(v1nodeclaim.Spec.NodeClassRef.Name).To(Equal(v1beta1nodeclaim.Spec.NodeClassRef.Name))
				Expect(v1nodeclaim.Spec.NodeClassRef.Group).To(Equal(cloudProvider.NodeClassGroupVersionKind[0].Group))
			})
		})
	})
	Context("NodeClaim Status", func() {
		It("should convert v1beta1 nodeclaim nodename", func() {
			v1beta1nodeclaim.Status.NodeName = "test-node-name"
			Expect(v1nodeclaim.ConvertFrom(ctx, v1beta1nodeclaim)).To(Succeed())
			Expect(v1beta1nodeclaim.Status.NodeName).To(Equal(v1nodeclaim.Status.NodeName))
		})
		It("should convert v1beta1 nodeclaim provider id", func() {
			v1beta1nodeclaim.Status.ProviderID = "test-provider-id"
			Expect(v1nodeclaim.ConvertFrom(ctx, v1beta1nodeclaim)).To(Succeed())
			Expect(v1beta1nodeclaim.Status.ProviderID).To(Equal(v1nodeclaim.Status.ProviderID))
		})
		It("should convert v1beta1 nodeclaim image id", func() {
			v1beta1nodeclaim.Status.ImageID = "test-image-id"
			Expect(v1nodeclaim.ConvertFrom(ctx, v1beta1nodeclaim)).To(Succeed())
			Expect(v1beta1nodeclaim.Status.ImageID).To(Equal(v1nodeclaim.Status.ImageID))
		})
		It("should convert v1beta1 nodeclaim capacity", func() {
			v1beta1nodeclaim.Status.Capacity = v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("13432"),
				v1.ResourceMemory: resource.MustParse("1332G"),
			}
			Expect(v1nodeclaim.ConvertFrom(ctx, v1beta1nodeclaim)).To(Succeed())
			Expect(v1beta1nodeclaim.Status.Capacity).To(Equal(v1nodeclaim.Status.Capacity))
		})
		It("should convert v1beta1 nodeclaim allocatable", func() {
			v1beta1nodeclaim.Status.Allocatable = v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("13432"),
				v1.ResourceMemory: resource.MustParse("1332G"),
			}
			Expect(v1nodeclaim.ConvertFrom(ctx, v1beta1nodeclaim)).To(Succeed())
			Expect(v1beta1nodeclaim.Status.Allocatable).To(Equal(v1nodeclaim.Status.Allocatable))
		})
		It("should convert v1beta1 nodeclaim conditions", func() {
			v1beta1nodeclaim.Status.Conditions = []status.Condition{
				{
					Status: status.ConditionReady,
					Reason: "test-reason",
				},
				{
					Status: ConditionTypeDrifted,
					Reason: "test-reason",
				},
			}
			Expect(v1nodeclaim.ConvertFrom(ctx, v1beta1nodeclaim)).To(Succeed())
			Expect(v1beta1nodeclaim.Status.Conditions).To(Equal(v1nodeclaim.Status.Conditions))
		})
	})
})
