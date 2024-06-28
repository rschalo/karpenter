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

package v1

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/samber/lo"
	"go.uber.org/multierr"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"knative.dev/pkg/ptr"
)

var (
	SupportedNodeSelectorOps = sets.NewString(
		string(v1.NodeSelectorOpIn),
		string(v1.NodeSelectorOpNotIn),
		string(v1.NodeSelectorOpGt),
		string(v1.NodeSelectorOpLt),
		string(v1.NodeSelectorOpExists),
		string(v1.NodeSelectorOpDoesNotExist),
	)

	SupportedReservedResources = sets.NewString(
		v1.ResourceCPU.String(),
		v1.ResourceMemory.String(),
		v1.ResourceEphemeralStorage.String(),
		"pid",
	)

	SupportedEvictionSignals = sets.NewString(
		"memory.available",
		"nodefs.available",
		"nodefs.inodesFree",
		"imagefs.available",
		"imagefs.inodesFree",
		"pid.available",
	)
)

func (in *NodeClaim) SupportedVerbs() []admissionregistrationv1.OperationType {
	return []admissionregistrationv1.OperationType{
		admissionregistrationv1.Create,
		admissionregistrationv1.Update,
	}
}

func (in *NodeClaimSpec) validate() error {
	var errs error
	errs = multierr.Append(errs, in.validateTaints())
	errs = multierr.Append(errs, in.validateRequirements())
	errs = multierr.Append(errs, in.Kubelet.validate())
	return errs
}

type taintKeyEffect struct {
	OwnerKey string
	Effect   v1.TaintEffect
}

func (in *NodeClaimSpec) validateTaints() error {
	var errs error
	existing := map[taintKeyEffect]struct{}{}
	errs = multierr.Append(errs, in.validateTaintsField(in.Taints, existing, "taint"))
	errs = multierr.Append(errs, in.validateTaintsField(in.StartupTaints, existing, "startupTaint"))
	return errs
}

func (in *NodeClaimSpec) validateTaintsField(taints []v1.Taint, existing map[taintKeyEffect]struct{}, fieldName string) error {
	var errs error
	for i, taint := range taints {
		// Validate OwnerKey
		if len(taint.Key) == 0 {
			errs = multierr.Append(errs, fmt.Errorf("missing taint key for %s, taint: %v", fieldName, i))
		}
		for _, err := range validation.IsQualifiedName(taint.Key) {
			errs = multierr.Append(errs, fmt.Errorf("invalid %s key: %s, err: %s", fieldName, taint.Key, err))
		}
		// Validate Value
		if len(taint.Value) != 0 {
			for _, err := range validation.IsQualifiedName(taint.Value) {
				errs = multierr.Append(errs, fmt.Errorf("invalid %s value: %s, for key: %s, err: %s", fieldName, taint.Value, taint.Key, err))
			}
		}
		// Validate effect
		switch taint.Effect {
		case v1.TaintEffectNoSchedule, v1.TaintEffectPreferNoSchedule, v1.TaintEffectNoExecute, "":
		default:
			errs = multierr.Append(errs, fmt.Errorf("invalid %s effect: %s, for value: %s, for key: %s", fieldName, taint.Effect, taint.Value, taint.Key))
		}

		// Check for duplicate OwnerKey/Effect pairs
		key := taintKeyEffect{OwnerKey: taint.Key, Effect: taint.Effect}
		if _, ok := existing[key]; ok {
			errs = multierr.Append(errs, fmt.Errorf("duplicate taint Key/Effect pair %s=%s", taint.Key, taint.Effect))
		}
		existing[key] = struct{}{}
	}
	return errs
}

// This function is used by the NodeClaim validation to verify the nodepool requirements.
// When this function is called, the nodepool's requirements do not include the requirements from labels.
// NodeClaim requirements only support well known labels.
func (in *NodeClaimSpec) validateRequirements() error {
	var errs error
	for _, requirement := range in.Requirements {
		if err := ValidateRequirement(requirement); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("invalid requirement: %v", requirement))
		}
	}
	return errs
}

func ValidateRequirement(requirement NodeSelectorRequirementWithMinValues) error { //nolint:gocyclo
	var errs error
	if normalized, ok := NormalizedLabels[requirement.Key]; ok {
		requirement.Key = normalized
	}
	if !SupportedNodeSelectorOps.Has(string(requirement.Operator)) {
		errs = multierr.Append(errs, fmt.Errorf("key %s has an unsupported operator %s not in %s", requirement.Key, requirement.Operator, SupportedNodeSelectorOps.UnsortedList()))
	}
	if e := IsRestrictedLabel(requirement.Key); e != nil {
		errs = multierr.Append(errs, e)
	}
	for _, err := range validation.IsQualifiedName(requirement.Key) {
		errs = multierr.Append(errs, fmt.Errorf("key %s is not a qualified name, %s", requirement.Key, err))
	}
	for _, value := range requirement.Values {
		for _, err := range validation.IsValidLabelValue(value) {
			errs = multierr.Append(errs, fmt.Errorf("invalid value %s for key %s, %s", value, requirement.Key, err))
		}
	}
	if requirement.Operator == v1.NodeSelectorOpIn && len(requirement.Values) == 0 {
		errs = multierr.Append(errs, fmt.Errorf("key %s with operator %s must have a value defined", requirement.Key, requirement.Operator))
	}

	if requirement.Operator == v1.NodeSelectorOpIn && requirement.MinValues != nil && len(requirement.Values) < lo.FromPtr(requirement.MinValues) {
		errs = multierr.Append(errs, fmt.Errorf("key %s with operator %s must have at least minimum number of values defined in 'values' field", requirement.Key, requirement.Operator))
	}

	if requirement.Operator == v1.NodeSelectorOpGt || requirement.Operator == v1.NodeSelectorOpLt {
		if len(requirement.Values) != 1 {
			errs = multierr.Append(errs, fmt.Errorf("key %s with operator %s must have a single positive integer value", requirement.Key, requirement.Operator))
		} else {
			value, err := strconv.Atoi(requirement.Values[0])
			if err != nil || value < 0 {
				errs = multierr.Append(errs, fmt.Errorf("key %s with operator %s must have a single positive integer value", requirement.Key, requirement.Operator))
			}
		}
	}
	return errs
}

func (in *KubeletConfiguration) validate() error {
	var errs error
	if in == nil {
		return nil
	}
	errs = multierr.Append(errs, validateEvictionThresholds(in.EvictionHard, "evictionHard"))
	errs = multierr.Append(errs, validateEvictionThresholds(in.EvictionSoft, "evictionSoft"))
	errs = multierr.Append(errs, validateReservedResources(in.KubeReserved, "kubeReserved"))
	errs = multierr.Append(errs, validateReservedResources(in.SystemReserved, "systemReserved"))
	errs = multierr.Append(errs, in.validateImageGCHighThresholdPercent())
	errs = multierr.Append(errs, in.validateImageGCLowThresholdPercent())
	errs = multierr.Append(errs, in.validateEvictionSoftGracePeriod())
	errs = multierr.Append(errs, in.validateEvictionSoftPairs())
	return errs
}

func (in *KubeletConfiguration) validateEvictionSoftGracePeriod() error {
	var errs error
	for k := range in.EvictionSoftGracePeriod {
		if !SupportedEvictionSignals.Has(k) {
			errs = multierr.Append(errs, fmt.Errorf("invalid key: %s for evictionSoftGracePeriod", k))
		}
	}
	return errs
}

func (in *KubeletConfiguration) validateEvictionSoftPairs() error {
	var errs error
	evictionSoftKeys := sets.New(lo.Keys(in.EvictionSoft)...)
	evictionSoftGracePeriodKeys := sets.New(lo.Keys(in.EvictionSoftGracePeriod)...)

	evictionSoftDiff := evictionSoftKeys.Difference(evictionSoftGracePeriodKeys)
	for k := range evictionSoftDiff {
		errs = multierr.Append(errs, fmt.Errorf("OwnerKey: %s does not have a matching evictionSoftGracePeriod", k))
	}
	evictionSoftGracePeriodDiff := evictionSoftGracePeriodKeys.Difference(evictionSoftKeys)
	for k := range evictionSoftGracePeriodDiff {
		errs = multierr.Append(errs, fmt.Errorf("OwnerKey: %s does not have a matching evictionSoft threshold value", k))
	}
	return errs
}

func validateReservedResources(m map[string]string, fieldName string) error {
	var errs error
	for k, v := range m {
		if !SupportedReservedResources.Has(k) {
			errs = multierr.Append(errs, fmt.Errorf("invalid key: %s, for: %s", k, fieldName))
		}
		quantity, err := resource.ParseQuantity(v)
		if err != nil {
			errs = multierr.Append(errs, fmt.Errorf(`%s["%s"]=%s, value must be a quantity value`, fieldName, k, v))
		}
		if quantity.Value() < 0 {
			errs = multierr.Append(errs, fmt.Errorf(`%s["%s"]=%s, value cannot be a negative resource quantity`, fieldName, k, v))
		}
	}
	return errs
}

func validateEvictionThresholds(m map[string]string, fieldName string) error {
	var errs error
	if m == nil {
		return nil
	}
	for k, v := range m {
		if !SupportedEvictionSignals.Has(k) {
			errs = multierr.Append(errs, fmt.Errorf("invalid key: %s, for: %s", k, fieldName))
		}
		if strings.HasSuffix(v, "%") {
			p, err := strconv.ParseFloat(strings.Trim(v, "%"), 64)
			if err != nil {
				errs = multierr.Append(errs, fmt.Errorf(`%s["%s"]=%s, value could not be parsed as a percentage value, %s`, fieldName, k, v, err.Error()))
			}
			if p < 0 {
				errs = multierr.Append(errs, fmt.Errorf(`%s["%s"]=%s, percentage values cannot be negative`, fieldName, k, v))
			}
			if p > 100 {
				errs = multierr.Append(errs, fmt.Errorf(`%s["%s"]=%s, percentage values cannot be greater than 100`, fieldName, k, v))
			}
		} else {
			_, err := resource.ParseQuantity(v)
			if err != nil {
				errs = multierr.Append(errs, fmt.Errorf(`%s[%s]=%s, invalid value, err: %s`, fieldName, k, v, err.Error()))
			}
		}
	}
	return errs
}

// Validate validateImageGCHighThresholdPercent
func (in *KubeletConfiguration) validateImageGCHighThresholdPercent() error {
	var errs error
	if in.ImageGCHighThresholdPercent != nil && ptr.Int32Value(in.ImageGCHighThresholdPercent) < ptr.Int32Value(in.ImageGCLowThresholdPercent) {
		return multierr.Append(errs, fmt.Errorf("imageGCHighThresholdPercent must be greater than imageGCLowThresholdPercent"))
	}

	return errs
}

// Validate imageGCLowThresholdPercent
func (in *KubeletConfiguration) validateImageGCLowThresholdPercent() error {
	var errs error
	if in.ImageGCHighThresholdPercent != nil && ptr.Int32Value(in.ImageGCLowThresholdPercent) > ptr.Int32Value(in.ImageGCHighThresholdPercent) {
		return multierr.Append(errs, fmt.Errorf("imageGCLowThresholdPercent must be less than imageGCHighThresholdPercent"))
	}

	return errs
}
