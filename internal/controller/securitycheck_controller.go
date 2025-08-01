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

package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	securityv1 "github.com/pyar6329/k8s-operator-sample/api/v1"
)

// SecurityCheckReconciler reconciles a SecurityCheck object
type SecurityCheckReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=security.k8s-operator.pyar.bz,resources=securitychecks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=security.k8s-operator.pyar.bz,resources=securitychecks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=security.k8s-operator.pyar.bz,resources=securitychecks/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SecurityCheck object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *SecurityCheckReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var logger = logf.FromContext(ctx)

	// Fetch the SecurityCheck instance
	var securityCheck securityv1.SecurityCheck
	if err := r.Get(ctx, req.NamespacedName, &securityCheck); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("SecurityCheck resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get SecurityCheck")
		return ctrl.Result{}, err
	}

	// Check pods in the target namespace
	var totalPods, violations, err = r.checkPodsInNamespace(ctx, &securityCheck)
	if err != nil {
		logger.Error(err, "Failed to check pods in namespace")
		return ctrl.Result{RequeueAfter: time.Minute * 5}, err
	}

	// Update the status of the SecurityCheck
	securityCheck.Status.TotalPods = int32(totalPods)
	securityCheck.Status.ViolationsCount = int32(violations)
	now := metav1.Now()
	securityCheck.Status.LastCheckTime = &now

	// Update conditions
	r.updateConditions(&securityCheck, violations)

	if err := r.Status().Update(ctx, &securityCheck); err != nil {
		logger.Error(err, "Failed to update SecurityCheck status")
		return ctrl.Result{}, err
	}

	logger.Info("SecurityCheck reconciled successfully",
		"namespace", securityCheck.Spec.TargetNamespace,
		"totalPods", totalPods,
		"violations", violations)

	// Requeue after 2 minutes
	return ctrl.Result{RequeueAfter: time.Minute * 2}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecurityCheckReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&securityv1.SecurityCheck{}).
		Named("securitycheck").
		Complete(r)
}

func (r *SecurityCheckReconciler) checkPodsInNamespace(ctx context.Context, sc *securityv1.SecurityCheck) (int, int, error) {
	logger := logf.FromContext(ctx)

	var podList corev1.PodList
	if err := r.List(ctx, &podList, client.InNamespace(sc.Spec.TargetNamespace)); err != nil {
		return 0, 0, fmt.Errorf("failed to list pods in namespace %s: %w", sc.Spec.TargetNamespace, err)
	}

	totalPods := len(podList.Items)
	violations := 0

	logger.Info("Checking pods for security violations",
		"namespace", sc.Spec.TargetNamespace,
		"podCount", totalPods)

	for _, pod := range podList.Items {
		podViolations := r.checkPodSecurity(ctx, &pod, sc)
		violations += podViolations

		if podViolations > 0 {
			logger.Info("Security violations found",
				"pod", pod.Name,
				"namespace", pod.Namespace,
				"violations", podViolations)
		}
	}

	return totalPods, violations, nil
}

func (r *SecurityCheckReconciler) checkPodSecurity(ctx context.Context, pod *corev1.Pod, sc *securityv1.SecurityCheck) int {
	violations := 0

	for _, rule := range sc.Spec.Rules {
		// Skip disabled rules
		if rule.Enabled != nil && !*rule.Enabled {
			continue
		}

		switch rule.Name {
		case "no-root-user":
			if r.checkRootUser(pod) {
				violations++
				r.recordViolation(ctx, pod, sc, rule.Name, "Pod is running as root user (UID 0)")
			}
		case "required-security-context":
			if r.checkSecurityContext(pod) {
				violations++
				r.recordViolation(ctx, pod, sc, rule.Name, "Pod container is missing security context")
			}
		case "no-privileged":
			if r.checkPrivileged(pod) {
				violations++
				r.recordViolation(ctx, pod, sc, rule.Name, "Pod is running in privileged mode")
			}
			// case "no-host-network":
			// 	if r.checkHostNetwork(pod) {
			// 		violations++
			// 		r.recordViolation(ctx, pod, sc, rule.Name, "Pod is using host network")
			// 	}
		}
	}

	return violations
}

func (r *SecurityCheckReconciler) checkRootUser(pod *corev1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		if container.SecurityContext != nil &&
			container.SecurityContext.RunAsUser != nil &&
			*container.SecurityContext.RunAsUser == 0 {
			return true
		}
	}
	return false
}

func (r *SecurityCheckReconciler) checkSecurityContext(pod *corev1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		if container.SecurityContext == nil {
			return true
		}
	}
	return false
}

func (r *SecurityCheckReconciler) checkPrivileged(pod *corev1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		if container.SecurityContext != nil &&
			container.SecurityContext.Privileged != nil &&
			*container.SecurityContext.Privileged {
			return true
		}
	}
	return false
}

func (r *SecurityCheckReconciler) recordViolation(ctx context.Context, pod *corev1.Pod, sc *securityv1.SecurityCheck, ruleName, message string) {
	logger := logf.FromContext(ctx)

	eventMessage := fmt.Sprintf("Pod %s/%s violated rule '%s': %s",
		pod.Namespace, pod.Name, ruleName, message)

	r.Recorder.Event(sc, corev1.EventTypeWarning, "SecurityViolation", eventMessage)

	logger.Info("Security violation recorded",
		"pod", pod.Name,
		"namespace", pod.Namespace,
		"rule", ruleName,
		"message", message)
}

func (r *SecurityCheckReconciler) updateConditions(sc *securityv1.SecurityCheck, violations int) {
	now := metav1.Now()

	var readyCondition metav1.Condition
	if violations == 0 {
		readyCondition = metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: now,
			Reason:             "NoViolations",
			Message:            "All pods comply with security policies",
		}
	} else {
		readyCondition = metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: now,
			Reason:             "SecurityViolations",
			Message:            fmt.Sprintf("Found %d security violations", violations),
		}
	}

	// Update or add the Ready condition
	updated := false
	for i, condition := range sc.Status.Conditions {
		if condition.Type == "Ready" {
			sc.Status.Conditions[i] = readyCondition
			updated = true
			break
		}
	}

	if !updated {
		sc.Status.Conditions = append(sc.Status.Conditions, readyCondition)
	}
}
