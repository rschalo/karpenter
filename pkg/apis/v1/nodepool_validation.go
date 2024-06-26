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
	"context"
	"fmt"

	"github.com/robfig/cron/v3"
	"github.com/samber/lo"
	"go.uber.org/multierr"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func (in *NodePool) SupportedVerbs() []admissionregistrationv1.OperationType {
	return []admissionregistrationv1.OperationType{
		admissionregistrationv1.Create,
		admissionregistrationv1.Update,
	}
}

func (in *NodePool) Validate(_ context.Context) error {
	var errs error
	errs = multierr.Append(errs, ValidateObjectMetadata(in))
	errs = multierr.Append(errs, in.Spec.validate())
	return errs
}

// RuntimeValidate will be used to validate any part of the CRD that can not be validated at CRD creation
func (in *NodePool) RuntimeValidate() error {
	var errs error
	errs = multierr.Append(errs, in.Spec.Template.validateLabels())
	errs = multierr.Append(errs, in.Spec.Template.Spec.validateTaints())
	errs = multierr.Append(errs, in.Spec.Template.Spec.validateRequirements())
	errs = multierr.Append(errs, in.Spec.Template.validateRequirementsNodePoolKeyDoesNotExist())
	return errs
}

func (in *NodePoolSpec) validate() error {
	var errs error
	errs = multierr.Append(errs, in.Template.validate())
	errs = multierr.Append(errs, in.Disruption.validate())
	return errs
}

func (in *NodeClaimTemplate) validate() error {
	var errs error
	if len(in.Spec.Resources.Requests) > 0 {
		errs = multierr.Append(errs, fmt.Errorf("resources.requests is a restricted field"))
	}
	errs = multierr.Append(errs, in.validateLabels())
	errs = multierr.Append(errs, in.validateRequirementsNodePoolKeyDoesNotExist())
	errs = multierr.Append(errs, in.Spec.validate())
	return errs
}

func (in *NodeClaimTemplate) validateLabels() error {
	var errs error
	for key, value := range in.Labels {
		if key == NodePoolLabelKey {
			errs = multierr.Append(errs, fmt.Errorf("key: %v is restricted", key))
		}
		for _, err := range validation.IsQualifiedName(key) {
			errs = multierr.Append(errs, fmt.Errorf("invalid label key: %s, err: %s", key, err))
		}
		for _, err := range validation.IsValidLabelValue(value) {
			errs = multierr.Append(errs, fmt.Errorf("invalid label value: %s, for key: %s, err: %s", value, key, err))
		}
		if err := IsRestrictedLabel(key); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("key: %v is restricted, err: %s", key, err.Error()))
		}
	}
	return errs
}

func (in *NodeClaimTemplate) validateRequirementsNodePoolKeyDoesNotExist() error {
	var errs error
	for _, requirement := range in.Spec.Requirements {
		if requirement.Key == NodePoolLabelKey {
			errs = multierr.Append(errs, fmt.Errorf("requirement key: %s is restricted", requirement.Key))
		}
	}
	return errs
}

//nolint:gocyclo
func (in *Disruption) validate() error {
	var errs error
	if in.ConsolidateAfter != nil && in.ConsolidateAfter.Duration != nil && in.ConsolidationPolicy == ConsolidationPolicyWhenUnderutilized {
		return multierr.Append(errs, fmt.Errorf("consolidateAfter cannot be combined with consolidationPolicy=WhenUnderutilized"))
	}
	if in.ConsolidateAfter == nil && in.ConsolidationPolicy == ConsolidationPolicyWhenEmpty {
		return multierr.Append(errs, fmt.Errorf("consolidateAfter must be specified with consolidationPolicy=WhenEmpty"))
	}
	for i := range in.Budgets {
		budget := in.Budgets[i]
		if err := budget.validate(); err != nil {
			errs = multierr.Append(errs, err)
		}
	}
	return errs
}

func (in *Budget) validate() error {
	if (in.Schedule != nil && in.Duration == nil) || (in.Schedule == nil && in.Duration != nil) {
		return fmt.Errorf("schedule and duration must be specified together")
	}
	if in.Schedule != nil {
		if _, err := cron.ParseStandard(lo.FromPtr(in.Schedule)); err != nil {
			return fmt.Errorf("invalid schedule %s, err: %w", lo.FromPtr(in.Schedule), err)
		}
	}
	return nil
}
