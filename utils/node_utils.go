package utils

import (
	"clustersnapshot_kwok_parser/kwok"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"

	"github.com/samber/lo"
	"k8s.io/client-go/kubernetes"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func SyncNodes(ctx context.Context, clientSet *kubernetes.Clientset, snapshotID string, nodeInfos []kwok.NodeInfo) error {
	// deletedNodeNames := sets.New[string]()
	// nodeInfosByName := lo.Associate(nodeInfos, func(item kwok.NodeInfo) (string, struct{}) {
	// 	return item.Name, struct{}{}
	// })
	virtualNodes, err := ListAllNodes(ctx, clientSet)
	if err != nil {
		return fmt.Errorf("cannot list the nodes in virtual cluster: %w", err)
	}
	virtualNodesMap := lo.KeyBy(virtualNodes, func(item corev1.Node) string {
		return item.Name
	})

	// for _, vn := range virtualNodes {
	// 	_, ok := nodeInfosByName[vn.Name]
	// 	if ok {
	// 		continue
	// 	}
	// 	err := clientSet.CoreV1().Nodes().Delete(ctx, vn.Name, metav1.DeleteOptions{})
	// 	if err != nil && !apierrors.IsNotFound(err) {
	// 		return fmt.Errorf("%s | cannot delete the virtual node %q: %w", snapshotID, vn.Name, err)
	// 	}
	// 	deletedNodeNames.Insert(vn.Name)
	// 	//delete(virtualNodesMap, vn.Name)
	// 	slog.Info("%s | syncNodes deleted the virtual node %q", snapshotID, vn.Name)
	// }
	// podsList, err := ListAllPods(ctx, clientSet)
	// if err != nil {
	// 	return fmt.Errorf("cannot list the pods in virtual cluster: %w", err)
	// }
	// for _, pod := range podsList {
	// 	if deletedNodeNames.Has(pod.Spec.NodeName) {
	// 		pod.Spec.NodeName = ""
	// 		_, err := clientSet.CoreV1().Pods(pod.Namespace).Update(ctx, &pod, metav1.UpdateOptions{})
	// 		if err != nil {
	// 			return fmt.Errorf("cannot update the pod %q: %w", pod.Name, err)
	// 		}
	// 		slog.Info("Cleared NodeName from pod", "pod.Name", pod.Name)
	// 	}
	// }

	for _, nodeInfo := range nodeInfos {
		oldVNode, exists := virtualNodesMap[nodeInfo.Name]
		var sameLabels, sameTaints bool
		if exists {
			sameLabels = maps.Equal(oldVNode.Labels, nodeInfo.Labels)
			sameTaints = slices.EqualFunc(oldVNode.Spec.Taints, nodeInfo.Taints, IsEqualTaint)
		}
		if exists && sameLabels && sameTaints {
			continue
		}
		node := corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nodeInfo.Name,
				Namespace: nodeInfo.Namespace,
				Labels:    nodeInfo.Labels,
			},
			Spec: corev1.NodeSpec{
				Taints:     nodeInfo.Taints,
				ProviderID: nodeInfo.ProviderID,
				PodCIDRs:   oldVNode.Spec.PodCIDRs,
			},
			Status: corev1.NodeStatus{
				Capacity:    nodeInfo.Capacity,
				Allocatable: nodeInfo.Allocatable,
				Conditions:  BuildReadyConditions(),
				Phase:       corev1.NodeRunning,
			},
		}
		node.Spec.Taints = lo.Filter(node.Spec.Taints, func(item corev1.Taint, index int) bool {
			return item.Key != "node.kubernetes.io/not-ready"
		})
		// node.Spec.Taints = append(node.Spec.Taints, corev1.Taint{
		// 	Effect: corev1.TaintEffectNoSchedule,
		// 	Key:    "kwok-provider",
		// 	Value:  "true",
		// })
		// node.ResourceVersion = ""
		// nodeStatus := node.Status
		if !exists {
			_, err = clientSet.CoreV1().Nodes().Create(context.Background(), &node, metav1.CreateOptions{})
			if apierrors.IsAlreadyExists(err) {
				slog.Warn("syncNodes: node already exists. updating node %q", "snapshotID", snapshotID, "nodeName", node.Name)
				_, err = clientSet.CoreV1().Nodes().Update(context.Background(), &node, metav1.UpdateOptions{})
			}
			if err == nil {
				slog.Info("syncNodes CREATED node.", "snapshotID", snapshotID, "nodeName", node.Name)
			}
		}
		// else {
		// 	_, err = clientSet.CoreV1().Nodes().Update(context.Background(), &node, metav1.UpdateOptions{})
		// 	slog.Info(" syncNodes UPDATED node.", "snapshotID", snapshotID, "nodeName", node.Name)
		// }
		// if err != nil {
		// 	return fmt.Errorf("syncNodes cannot create/update node with name %q: %w", node.Name, err)
		// }

		// fmt.Printf("Node before adjust:\n%#v\n", node)
		// node.Status = nodeStatus
		// node.Status.Conditions = BuildReadyConditions()
		// err = adjustNode(clientSet, node.Name, node.Status)
		// if err != nil {
		// return fmt.Errorf("syncNodes cannot adjust the node with name %q: %w", node.Name, err)
		// }
	}
	return nil
}

func ListAllNodes(ctx context.Context, clientSet *kubernetes.Clientset) ([]corev1.Node, error) {
	return ListAllNodesWithPageSize(ctx, clientSet, 0)
}

func ListAllNodesWithPageSize(ctx context.Context, clientSet *kubernetes.Clientset, pageSize int) ([]corev1.Node, error) {
	// Initialize the list options with a page size
	var listOptions metav1.ListOptions
	if pageSize > 0 {
		listOptions = metav1.ListOptions{
			Limit: int64(pageSize), // Set a limit for pagination
		}
	}
	var allNodes []corev1.Node
	for {
		// List nodes with the current list options
		if ctx.Err() != nil {
			return nil, fmt.Errorf("cannot list nodes since context.Err is non-nil: %w", ctx.Err())
		}
		nodes, err := clientSet.CoreV1().Nodes().List(ctx, listOptions)
		if err != nil {
			return nil, err
		}
		// Append the current page of nodes to the allNodes slice
		allNodes = append(allNodes, nodes.Items...)
		// Check if there is another page
		if nodes.Continue == "" {
			break
		}
		// Set the continue token for the next request
		listOptions.Continue = nodes.Continue
	}
	return allNodes, nil
}

func adjustNode(clientSet *kubernetes.Clientset, nodeName string, nodeStatus corev1.NodeStatus) error {
	nd, err := clientSet.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("adjustNode cannot get node with name %q: %w", nd.Name, err)
	}
	nd.Spec.Taints = lo.Filter(nd.Spec.Taints, func(item corev1.Taint, index int) bool {
		return item.Key != "node.kubernetes.io/not-ready"
	})
	nd.ResourceVersion = ""
	fmt.Printf("Node after spec before:\n%#v\n", nd)
	nd, err = clientSet.CoreV1().Nodes().Update(context.Background(), nd, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("adjustNode cannot update node with name %q: %w", nd.Name, err)
	}
	// nd, err = clientSet.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	// if err != nil {
	// 	return fmt.Errorf("adjustNode cannot get node with name %q: %w", nd.Name, err)
	// }
	// fmt.Printf("Node before update status:\n%#v\n", nd)
	nd.ResourceVersion = ""
	nd.Status = nodeStatus
	nd.Status.Conditions = BuildReadyConditions()
	nd.Status.Phase = corev1.NodeRunning
	nd, err = clientSet.CoreV1().Nodes().UpdateStatus(context.Background(), nd, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("adjustNode cannot update the status of node with name %q: %w", nd.Name, err)
	}
	// nd, err = clientSet.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	// if err != nil {
	// 	return fmt.Errorf("adjustNode cannot get node with name %q: %w", nd.Name, err)
	// }
	// fmt.Printf("Node after update status:\n%#v\n", nd)
	return nil
}
