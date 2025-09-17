/*
Copyright 2025.

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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// nolint:unused
// log is for logging in this package.
var nodelog = logf.Log.WithName("node-resource")

// SetupNodeWebhookWithManager registers the webhook for Node in the manager.
func SetupNodeWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&corev1.Node{}).
		WithValidator(&NodeCustomValidator{}).
		WithDefaulter(&NodeCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate--v1-node,mutating=true,failurePolicy=fail,sideEffects=None,groups="",resources=nodes/status,verbs=create;update,versions=v1,name=mnode-v1.kb.io,admissionReviewVersions=v1

// NodeCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Node when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type NodeCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &NodeCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Node.
func (d *NodeCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	node, ok := obj.(*corev1.Node)

	if !ok {
		return fmt.Errorf("expected an Node object but got %T", obj)
	}
	nodelog.Info("Defaulting for Node", "name", node.GetName())

	// TODO(user): fill in your defaulting logic.
	lastTransition := time.Now().Add(-time.Minute)
	node.Status.Conditions = []corev1.NodeCondition{
		{
			Type:               corev1.NodeReady,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: lastTransition},
		},
		{
			Type:               corev1.NodeNetworkUnavailable,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: lastTransition},
		},
		{
			Type:               corev1.NodeDiskPressure,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: lastTransition},
		},
		{
			Type:               corev1.NodeMemoryPressure,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: lastTransition},
		},
	}

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate--v1-node,mutating=false,failurePolicy=fail,sideEffects=None,groups="",resources=nodes,verbs=create;update,versions=v1,name=vnode-v1.kb.io,admissionReviewVersions=v1

// NodeCustomValidator struct is responsible for validating the Node resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type NodeCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &NodeCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Node.
func (v *NodeCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	node, ok := obj.(*corev1.Node)
	if !ok {
		return nil, fmt.Errorf("expected a Node object but got %T", obj)
	}
	nodelog.Info("Validation for Node upon creation", "name", node.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Node.
func (v *NodeCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	node, ok := newObj.(*corev1.Node)
	if !ok {
		return nil, fmt.Errorf("expected a Node object for the newObj but got %T", newObj)
	}
	nodelog.Info("Validation for Node upon update", "name", node.GetName())

	// TODO(user): fill in your validation logic upon object update.
	if node.Status.Conditions == nil {
		return nil, fmt.Errorf("trying to mark node as not ready")
	}
	found := false
	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			if cond.Status != corev1.ConditionTrue {
				return nil, fmt.Errorf("trying to mark node as not ready")
			}
			found = true
		}
	}

	if !found {
		return nil, fmt.Errorf("trying to mark node as not ready")
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Node.
func (v *NodeCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	node, ok := obj.(*corev1.Node)
	if !ok {
		return nil, fmt.Errorf("expected a Node object but got %T", obj)
	}
	nodelog.Info("Validation for Node upon deletion", "name", node.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
