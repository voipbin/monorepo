package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"monorepo/bin-sentinel-manager/internal/config"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "sentinel-control",
		Short: "Voipbin Sentinel Management CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if errBind := viper.BindPFlags(cmd.Flags()); errBind != nil {
				return errors.Wrap(errBind, "failed to bind flags")
			}

			config.LoadGlobalConfig()
			return nil
		},
	}

	if err := config.Bootstrap(cmdRoot); err != nil {
		cobra.CheckErr(errors.Wrap(err, "failed to bootstrap config"))
	}

	cmdPod := &cobra.Command{Use: "pod", Short: "Pod monitoring operations"}
	cmdPod.AddCommand(cmdPodList())
	cmdPod.AddCommand(cmdPodGet())

	cmdRoot.AddCommand(cmdPod)
	return cmdRoot
}

func cmdPodList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List monitored pods in a namespace",
		RunE:  runPodList,
	}

	flags := cmd.Flags()
	flags.String("namespace", "voip", "Kubernetes namespace to query")
	flags.String("selector", "", "Label selector (e.g., app=asterisk-call)")

	return cmd
}

func cmdPodGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a specific pod by name",
		RunE:  runPodGet,
	}

	flags := cmd.Flags()
	flags.String("namespace", "voip", "Kubernetes namespace")
	flags.String("name", "", "Pod name (required)")

	return cmd
}

func runPodList(cmd *cobra.Command, args []string) error {
	clientset, err := getKubernetesClient()
	if err != nil {
		return errors.Wrap(err, "failed to create Kubernetes client")
	}

	namespace := viper.GetString("namespace")
	selector := viper.GetString("selector")

	listOptions := metav1.ListOptions{}
	if selector != "" {
		listOptions.LabelSelector = selector
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to list pods")
	}

	// Create a simplified response with essential pod information
	type PodInfo struct {
		Name      string            `json:"name"`
		Namespace string            `json:"namespace"`
		Phase     corev1.PodPhase   `json:"phase"`
		PodIP     string            `json:"pod_ip"`
		HostIP    string            `json:"host_ip"`
		Labels    map[string]string `json:"labels"`
		NodeName  string            `json:"node_name"`
	}

	result := make([]PodInfo, 0, len(pods.Items))
	for _, pod := range pods.Items {
		result = append(result, PodInfo{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Phase:     pod.Status.Phase,
			PodIP:     pod.Status.PodIP,
			HostIP:    pod.Status.HostIP,
			Labels:    pod.Labels,
			NodeName:  pod.Spec.NodeName,
		})
	}

	return printJSON(result)
}

func runPodGet(cmd *cobra.Command, args []string) error {
	clientset, err := getKubernetesClient()
	if err != nil {
		return errors.Wrap(err, "failed to create Kubernetes client")
	}

	namespace := viper.GetString("namespace")
	name := viper.GetString("name")

	if name == "" {
		return fmt.Errorf("pod name is required")
	}

	pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get pod")
	}

	// Create a detailed response with pod information
	type ContainerInfo struct {
		Name    string `json:"name"`
		Image   string `json:"image"`
		Ready   bool   `json:"ready"`
		Started *bool  `json:"started"`
	}

	type PodDetailInfo struct {
		Name       string            `json:"name"`
		Namespace  string            `json:"namespace"`
		Phase      corev1.PodPhase   `json:"phase"`
		PodIP      string            `json:"pod_ip"`
		HostIP     string            `json:"host_ip"`
		Labels     map[string]string `json:"labels"`
		NodeName   string            `json:"node_name"`
		Containers []ContainerInfo   `json:"containers"`
	}

	containers := make([]ContainerInfo, 0, len(pod.Status.ContainerStatuses))
	for _, cs := range pod.Status.ContainerStatuses {
		containers = append(containers, ContainerInfo{
			Name:    cs.Name,
			Image:   cs.Image,
			Ready:   cs.Ready,
			Started: cs.Started,
		})
	}

	result := PodDetailInfo{
		Name:       pod.Name,
		Namespace:  pod.Namespace,
		Phase:      pod.Status.Phase,
		PodIP:      pod.Status.PodIP,
		HostIP:     pod.Status.HostIP,
		Labels:     pod.Labels,
		NodeName:   pod.Spec.NodeName,
		Containers: containers,
	}

	return printJSON(result)
}

func getKubernetesClient() (*kubernetes.Clientset, error) {
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create in-cluster config")
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Kubernetes clientset")
	}

	return clientset, nil
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal JSON")
	}
	fmt.Println(string(data))
	return nil
}
