package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/yaml"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"clustersnapshot_kwok_parser/kwok"
	"clustersnapshot_kwok_parser/utils"
)

func getVirtualClusterClient() (*kubeclient.Clientset, error) {
	kubeconfigPath := "/Users/I759930/.kube/config"
	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}
	restConfig.QPS = 1000.0
	restConfig.Burst = 2000.0
	// Create clientset
	clientset, err := kubeclient.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot create virtual cluster clientset: %w", err)
	}
	return clientset, nil
}

func waitForScheduling(ctx context.Context, client *kubernetes.Clientset, pollInterval time.Duration) {
	unScheduledPodCount := 0
	for {
		slog.Info("waiting for a stabilize interval for scheduler to finish its job", "pollInterval", pollInterval)
		time.Sleep(pollInterval)
		// list all pods, check if number of unscheduled pods remains the same
		unschedPodList, _ := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
			FieldSelector: `spec.nodeName=,status.nominatedNodeName=`,
		})
		slog.Info("Unscheduled pods", "num", len(unschedPodList.Items))
		if len(unschedPodList.Items) == 0 {
			break
		}
		if unScheduledPodCount == len(unschedPodList.Items) {
			slog.Info("Unscheduled Pod Names", "names", lo.Map(unschedPodList.Items, func(pod corev1.Pod, _ int) string {
				return pod.Name
			}))
			break
		} else {
			unScheduledPodCount = len(unschedPodList.Items)
		}
	}
}

func syncVirtualCluster(ctx context.Context, client *kubernetes.Clientset, snap kwok.ClusterSnapshot) (err error) {
	nsSet := snap.GetPodNamspaces()
	err = utils.CreateNamespaces(ctx, client, nsSet.UnsortedList()...)
	if err != nil {
		panic(err)
	}
	for _, pClass := range snap.PriorityClasses {
		_, err = client.SchedulingV1().PriorityClasses().Create(ctx, &pClass.PriorityClass, metav1.CreateOptions{})

		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				klog.Infof("priorityclass %s already exists", pClass.Name)
				continue
			}
			if strings.Contains(err.Error(), "Only one default can exist") {
				slog.Info("A default PriorityClass already exists, skipping creation of another default.")
				continue
			}
			return fmt.Errorf("syncVirtualCluster cannot create the priority class %s: %w", pClass.Name, err)
		}
		slog.Info("syncVirtualCluster successfully created the priority class", "pc.Name", pClass.Name)
	}

	// TODO: remove all existing pods in the virtual cluster
	// utils.DeleteExistingPods(ctx, client)
	err = utils.SyncNodes(ctx, client, snap.ID, snap.Nodes)
	if err != nil {
		return err
	}

	for _, pod := range snap.Pods {
		err = utils.DoDeployPod(ctx, client, utils.GetCorePodFromPodInfo(pod))
		if err != nil {
			return err
		}
	}

	pollingInterval := 30 * time.Second
	waitForScheduling(ctx, client, pollingInterval)

	return nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	file, err := os.Open("input/scale-incremental-cs.json")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	clusterSnapshot := kwok.ClusterSnapshot{}
	if err := json.Unmarshal(data, &clusterSnapshot); err != nil {
		panic(err)
	}

	client, err := getVirtualClusterClient()
	if err != nil {
		panic(err)
	}

	// Deploying Kwok Provider Config
	var kwokProviderConfig kwok.KwokProviderConfig
	kwokProviderConfig.APIVersion = "v1alpha"
	kwokProviderConfig.ReadNodesFrom = "configmap"
	kwokProviderConfig.Nodegroups = &kwok.NodegroupsConfig{FromNodeLabelKey: "worker.gardener.cloud/pool"}
	kwokProviderConfig.ConfigMap = &kwok.ConfigMapConfig{Name: "kwok-provider-templates"}

	providerConfigYaml, err := yaml.Marshal(kwokProviderConfig)
	if err != nil {
		panic(err)
	}

	providerConfig := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "kwok-provider-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"config": string(providerConfigYaml),
		},
	}
	client.CoreV1().ConfigMaps("default").Create(ctx, providerConfig, v1.CreateOptions{})

	// Deploying Kwok Provider Templates
	var kwokProviderTemplates kwok.KwokProviderTemplates
	kwokProviderTemplates.APIVersion = "v1"
	kwokProviderTemplates.Kind = "List"
	for _, nodeTemplate := range clusterSnapshot.AutoscalerConfig.NodeTemplates {
		node := corev1.Node{
			TypeMeta: v1.TypeMeta{
				Kind:       "Node",
				APIVersion: "v1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:   nodeTemplate.Name,
				Labels: nodeTemplate.Labels,
				Annotations: map[string]string{
					"kwok.x-k8s.io/node": "fake",
				},
			},
			Spec: corev1.NodeSpec{
				Taints: nodeTemplate.Taints,
			},
			Status: corev1.NodeStatus{
				Capacity:    nodeTemplate.Capacity,
				Allocatable: nodeTemplate.Allocatable,
				Conditions:  utils.BuildReadyConditions(),
			},
		}
		kwokProviderTemplates.Items = append(kwokProviderTemplates.Items, node)
	}

	providerTemplatesYaml, err := yaml.Marshal(kwokProviderTemplates)
	if err != nil {
		panic(err)
	}

	providerTemplate := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "kwok-provider-templates",
			Namespace: "default",
		},
		Data: map[string]string{
			"templates": string(providerTemplatesYaml),
		},
	}
	client.CoreV1().ConfigMaps("default").Create(ctx, providerTemplate, v1.CreateOptions{})

	// start ca-kwok
	err = syncVirtualCluster(ctx, client, clusterSnapshot)
	if err != nil {
		panic(err)
	}
}
