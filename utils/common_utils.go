package utils

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"k8s.io/client-go/kubernetes"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildReadyConditions sets up mock NodeConditions
func BuildReadyConditions() []corev1.NodeCondition {
	lastTransition := time.Now().Add(-time.Minute)
	return []corev1.NodeCondition{
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
}

func CreateNamespaces(ctx context.Context, clientSet *kubernetes.Clientset, nss ...string) error {
	for _, ns := range nss {
		if ns == "default" {
			continue
		}
		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		}
		_, err := clientSet.CoreV1().Namespaces().Create(ctx, &namespace, metav1.CreateOptions{})
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("cannot create the namespace %q in virtual cluster: %w", ns, err)
		}
		slog.Info("created namespace", "namespace", ns)
	}
	return nil
}

func IsEqualTaint(a, b corev1.Taint) bool {
	return a.Key == b.Key && a.Value == b.Value && a.Effect == b.Effect
}
