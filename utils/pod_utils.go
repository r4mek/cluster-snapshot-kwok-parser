package utils

import (
	"clustersnapshot_kwok_parser/kwok"
	"context"
	"fmt"
	"log/slog"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func DeleteExistingPods(ctx context.Context, client *kubernetes.Clientset) error {
	deletePodList, err := ListAllPods(ctx, client)
	if err != nil {
		return err
	}
	for _, pod := range deletePodList {
		err := client.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, v1.DeleteOptions{})
		if err != nil {
			return err
		}
		slog.Info("delete pod finished.", "pod.Name", pod.Name, "pod.Namespace", pod.Namespace, "pod.NodeName", pod.Spec.NodeName)
	}
	slog.Info("deleted all existing pods")
	return nil
}

func DoDeployPod(ctx context.Context, clientSet *kubernetes.Clientset, pod corev1.Pod) error {
	podNew, err := clientSet.CoreV1().Pods(pod.Namespace).Create(ctx, &pod, metav1.CreateOptions{})
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("doDeployPod cannot create the pod  %s: %w", pod.Name, err)
		} else {
			return nil
		}
	}
	if podNew.Spec.NodeName != "" {
		podNew.Status.Phase = corev1.PodRunning
		podNew.ResourceVersion = ""
		_, err = clientSet.CoreV1().Pods(pod.Namespace).UpdateStatus(ctx, podNew, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("doDeployPod cannot change the pod Phase to Running for %s: %w", pod.Name, err)
		}
	}
	slog.Info("doDeployPod finished.", "pod.Name", pod.Name, "pod.Namespace", pod.Namespace, "pod.NodeName", pod.Spec.NodeName)
	return nil
}

func GetCorePodFromPodInfo(podInfo kwok.PodInfo) corev1.Pod {
	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    podInfo.Labels,
			Name:      podInfo.Name,
			Namespace: podInfo.Namespace,
		},
		Spec: podInfo.Spec,
	}
	pod.Spec.NodeName = podInfo.NodeName
	if pod.Spec.NodeName == "" {
		// FIXME HACK
		pod.Spec.Tolerations = append(pod.Spec.Tolerations, corev1.Toleration{
			Key:      "kwok-provider",
			Value:    "true",
			Effect:   corev1.TaintEffectNoSchedule,
			Operator: corev1.TolerationOpEqual,
		})
	}
	pod.Spec.ServiceAccountName = "default"
	pod.Status.NominatedNodeName = podInfo.NominatedNodeName
	return pod
}

func ListAllPods(ctx context.Context, clientSet *kubernetes.Clientset) ([]corev1.Pod, error) {
	return ListAllPodsWithPageSize(ctx, clientSet, 0)
}

func ListAllPodsWithPageSize(ctx context.Context, clientSet *kubernetes.Clientset, pageSize int) ([]corev1.Pod, error) {
	// Initialize the list options with a page size
	var listOptions metav1.ListOptions
	if pageSize > 0 {
		listOptions = metav1.ListOptions{
			Limit: int64(pageSize), // Set a limit for pagination
		}
	}
	var allPods []corev1.Pod
	for {
		// List nodes with the current list options
		if ctx.Err() != nil {
			return nil, fmt.Errorf("cannot list Pods since context.Err is non-nil: %w", ctx.Err())
		}
		nodes, err := clientSet.CoreV1().Pods("").List(ctx, listOptions)
		if err != nil {
			return nil, err
		}
		// Append the current page of nodes to the allPods slice
		allPods = append(allPods, nodes.Items...)
		// Check if there is another page
		if nodes.Continue == "" {
			break
		}
		// Set the continue token for the next request
		listOptions.Continue = nodes.Continue
	}
	return allPods, nil
}
