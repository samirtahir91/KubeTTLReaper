/*
Copyright 2024.

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

package controller

import (
	"context"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	TtlLabel = "kubettlreaper.samir.io/ttl"
)

var (
	MetadataLabel     = fmt.Sprintf("metadata.labels.%s", TtlLabel)
	OperatorNamespace = os.Getenv("OPERATOR_NAMESPACE")
)

// TtlReaperReconciler reconciles a TtlReaper object
type TtlReaperReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	ConfigurationName string
}

// +kubebuilder:rbac:groups=core,resources=*,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=*/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=*/finalizers,verbs=update

// Reconcile
func (r *TtlReaperReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	l.Info("Reconciling", "ConfigurationName", r.ConfigurationName)

	// Fetch the ConfigMap that stores the GVKs
	configMap := &corev1.ConfigMap{}
	err := r.Get(ctx, client.ObjectKey{
		Namespace: OperatorNamespace,
		Name:      r.ConfigurationName,
	}, configMap)
	if err != nil {
		l.Error(err, "Failed to fetch ConfigMap")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	// Fetch the requeue interval from the ConfigMap
	requeueAfterTime, err := r.getRequeueTimeFromConfigMap(configMap)
	if err != nil {
		l.Error(err, "Failed to get requeue interval from ConfigMap")
		return ctrl.Result{}, err
	}
	l.Info("Requeue interval fetched from ConfigMap", "requeueAfter", requeueAfterTime)

	// Parse the GVKs from the ConfigMap data
	var gvkList []schema.GroupVersionKind
	err = yaml.Unmarshal([]byte(configMap.Data["gvk-list"]), &gvkList)
	if err != nil {
		l.Error(err, "Failed to parse GVK list")
		return ctrl.Result{RequeueAfter: requeueAfterTime}, err
	}

	// Log and skip processing if GVK list is empty
	if len(gvkList) == 0 {
		l.Info("GVK list is empty, skipping reconciliation")
		return ctrl.Result{RequeueAfter: requeueAfterTime}, nil
	} else {
		l.Info("GVK list is not empty", "gvkList", gvkList)
	}

	// Loop through each GVK and list the resources
	for _, gvk := range gvkList {
		resources := &unstructured.UnstructuredList{}
		resources.SetGroupVersionKind(gvk)

		opts := []client.ListOption{
			client.HasLabels{TtlLabel},
		}

		if err := r.Client.List(context.Background(), resources, opts...); err != nil {
			l.Error(err, "Failed to list resources", "gvk", gvk.String())
			return ctrl.Result{}, err
		}

		// Log and skip if no resources found for the GVK
		if len(resources.Items) == 0 {
			l.Info("No resources found for GVK, skipping", "gvk", gvk.String())
			continue
		} else {
			l.Info("Resources found", "count", len(resources.Items), "gvk", gvk.String())
		}

		// Loop through each resource and check TTL
		for _, resource := range resources.Items {
			ttlValue, exists := resource.GetLabels()[TtlLabel]
			if !exists {
				continue
			}

			ttlDuration, err := time.ParseDuration(ttlValue)
			if err != nil {
				l.Error(err, "Invalid TTL value", "resource", resource.GetName())
				continue
			}

			creationTime := resource.GetCreationTimestamp().Time
			expirationTime := creationTime.Add(ttlDuration)

			if time.Now().After(expirationTime) {
				l.Info("Deleting expired resource", "resource", resource.GetName(), "gvk", gvk.String())
				if err := r.Client.Delete(ctx, &resource); err != nil {
					l.Error(err, "Failed to delete resource", "resource", resource.GetName())
				}
			}
		}
	}

	return ctrl.Result{RequeueAfter: requeueAfterTime}, nil
}

// Get check interval from config map
func (r *TtlReaperReconciler) getRequeueTimeFromConfigMap(configMap *corev1.ConfigMap) (time.Duration, error) {

	// Get the value for the "check-interval" key
	checkIntervalStr, exists := configMap.Data["check-interval"]
	if !exists {
		return 0, fmt.Errorf("check-interval not found in ConfigMap")
	}

	checkInterval, err := time.ParseDuration(checkIntervalStr)
	if err != nil {
		return 0, fmt.Errorf("invalid check-interval value: %v", err)
	}

	return checkInterval, nil
}

// Only use configmap named as per param
func nameMatchPredicate(name string) predicate.Predicate {
	return predicate.NewPredicateFuncs(func(object client.Object) bool {
		return object.GetName() == name && object.GetNamespace() == OperatorNamespace
	})
}

func (r *TtlReaperReconciler) SetupWithManager(mgr ctrl.Manager, configurationName string) error {
	r.ConfigurationName = configurationName

	// Watch the ConfigMap for changes (GVKs to watch)
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{},
			builder.WithPredicates(
				predicate.ResourceVersionChangedPredicate{},
				nameMatchPredicate(configurationName),
			)).
		Complete(r)
}
