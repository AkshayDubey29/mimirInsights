package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	github.com/akshaydubey29/mimirInsights/pkg/config"
	github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8s.io/client-go/rest"
	k8s.io/client-go/tools/clientcmd"
)

// Client wraps the Kubernetes client with additional functionality
type Client struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

// NewClient creates a new Kubernetes client
func NewClient() (*Client, error) {
	cfg := config.Get()
	
	var k8sConfig *rest.Config
	var err error

	if cfg.K8s.InCluster {
		// Use in-cluster configuration
		k8sConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
		}
	} else {
		// Use kubeconfig file
		kubeconfig := cfg.K8s.ConfigPath
		if kubeconfig == "" {
			kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
		}

		k8sConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
	}

	// Override cluster URL if specified
	if cfg.K8s.ClusterURL != "" {
		k8sConfig.Host = cfg.K8s.ClusterURL
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	logrus.Info("Kubernetes client initialized successfully")
	
	return &Client{
		clientset: clientset,
		config:    cfg,
	}, nil
}

// GetDeployments retrieves deployments from a namespace
func (c *Client) GetDeployments(ctx context.Context, namespace string, opts metav1.ListOptions) (*appsv1.DeploymentList, error) {
	return c.clientset.AppsV1().Deployments(namespace).List(ctx, opts)
}

// GetStatefulSets retrieves statefulsets from a namespace
func (c *Client) GetStatefulSets(ctx context.Context, namespace string, opts metav1.ListOptions) (*appsv1.StatefulSetList, error) {
	return c.clientset.AppsV1().StatefulSets(namespace).List(ctx, opts)
}

// GetNamespaces retrieves all namespaces
func (c *Client) GetNamespaces(ctx context.Context, opts metav1.ListOptions) (*corev1.NamespaceList, error) {
	return c.clientset.CoreV1().Namespaces().List(ctx, opts)
}

// GetConfigMaps retrieves configmaps from a namespace
func (c *Client) GetConfigMaps(ctx context.Context, namespace string, opts metav1.ListOptions) (*corev1.ConfigMapList, error) {
	return c.clientset.CoreV1().ConfigMaps(namespace).List(ctx, opts)
}

// GetPods retrieves pods from a namespace
func (c *Client) GetPods(ctx context.Context, namespace string, opts metav1.ListOptions) (*corev1.PodList, error) {
	return c.clientset.CoreV1().Pods(namespace).List(ctx, opts)
}

// GetServices retrieves services from a namespace
func (c *Client) GetServices(ctx context.Context, namespace string, opts metav1.ListOptions) (*corev1.ServiceList, error) {
	return c.clientset.CoreV1().Services(namespace).List(ctx, opts)
}

// GetPersistentVolumeClaims retrieves PVCs from a namespace
func (c *Client) GetPersistentVolumeClaims(ctx context.Context, namespace string, opts metav1.ListOptions) (*corev1.PersistentVolumeClaimList, error) {
	return c.clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, opts)
}

// GetDeployment retrieves a specific deployment
func (c *Client) GetDeployment(ctx context.Context, namespace, name string, opts metav1.GetOptions) (*appsv1.Deployment, error) {
	return c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, opts)
}

// GetStatefulSet retrieves a specific statefulset
func (c *Client) GetStatefulSet(ctx context.Context, namespace, name string, opts metav1.GetOptions) (*appsv1.StatefulSet, error) {
	return c.clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, opts)
}

// GetConfigMap retrieves a specific configmap
func (c *Client) GetConfigMap(ctx context.Context, namespace, name string, opts metav1.GetOptions) (*corev1.ConfigMap, error) {
	return c.clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, opts)
}

// UpdateDeployment updates a deployment
func (c *Client) UpdateDeployment(ctx context.Context, namespace string, deployment *appsv1.Deployment, opts metav1.UpdateOptions) (*appsv1.Deployment, error) {
	return c.clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, opts)
}

// UpdateStatefulSet updates a statefulset
func (c *Client) UpdateStatefulSet(ctx context.Context, namespace string, statefulSet *appsv1.StatefulSet, opts metav1.UpdateOptions) (*appsv1.StatefulSet, error) {
	return c.clientset.AppsV1().StatefulSets(namespace).Update(ctx, statefulSet, opts)
}

// UpdateConfigMap updates a configmap
func (c *Client) UpdateConfigMap(ctx context.Context, namespace string, configMap *corev1.ConfigMap, opts metav1.UpdateOptions) (*corev1.ConfigMap, error) {
	return c.clientset.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, opts)
}

// PatchDeployment patches a deployment
func (c *Client) PatchDeployment(ctx context.Context, namespace, name string, pt string, data []byte, opts metav1.PatchOptions) (*appsv1.Deployment, error) {
	return c.clientset.AppsV1().Deployments(namespace).Patch(ctx, name, pt, data, opts)
}

// PatchStatefulSet patches a statefulset
func (c *Client) PatchStatefulSet(ctx context.Context, namespace, name string, pt string, data []byte, opts metav1.PatchOptions) (*appsv1.StatefulSet, error) {
	return c.clientset.AppsV1().StatefulSets(namespace).Patch(ctx, name, pt, data, opts)
}

// PatchConfigMap patches a configmap
func (c *Client) PatchConfigMap(ctx context.Context, namespace, name string, pt string, data []byte, opts metav1.PatchOptions) (*corev1.ConfigMap, error) {
	return c.clientset.CoreV1().ConfigMaps(namespace).Patch(ctx, name, pt, data, opts)
}

// GetPodLogs retrieves logs from a pod
func (c *Client) GetPodLogs(ctx context.Context, namespace, podName string, opts *corev1.PodLogOptions) ([]byte, error) {
	req := c.clientset.CoreV1().Pods(namespace).GetLogs(podName, opts)
	return req.Do(ctx).Raw()
}

// GetPodStatus retrieves the status of a pod
func (c *Client) GetPodStatus(ctx context.Context, namespace, podName string) (*corev1.PodStatus, error) {
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return &pod.Status, nil
}

// GetResourceQuota retrieves resource quotas from a namespace
func (c *Client) GetResourceQuota(ctx context.Context, namespace string, opts metav1.ListOptions) (*corev1.ResourceQuotaList, error) {
	return c.clientset.CoreV1().ResourceQuotas(namespace).List(ctx, opts)
}

// GetLimitRange retrieves limit ranges from a namespace
func (c *Client) GetLimitRange(ctx context.Context, namespace string, opts metav1.ListOptions) (*corev1.LimitRangeList, error) {
	return c.clientset.CoreV1().LimitRanges(namespace).List(ctx, opts)
}

// GetEvents retrieves events from a namespace
func (c *Client) GetEvents(ctx context.Context, namespace string, opts metav1.ListOptions) (*corev1.EventList, error) {
	return c.clientset.CoreV1().Events(namespace).List(ctx, opts)
}

// GetNodeList retrieves all nodes
func (c *Client) GetNodeList(ctx context.Context, opts metav1.ListOptions) (*corev1.NodeList, error) {
	return c.clientset.CoreV1().Nodes().List(ctx, opts)
}

// GetNamespace retrieves a specific namespace
func (c *Client) GetNamespace(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Namespace, error) {
	return c.clientset.CoreV1().Namespaces().Get(ctx, name, opts)
}

// TestConnection tests the connection to the Kubernetes cluster
func (c *Client) TestConnection(ctx context.Context) error {
	_, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		return fmt.Errorf("failed to connect to Kubernetes cluster: %w", err)
	}
	return nil
}

// GetClusterInfo retrieves basic cluster information
func (c *Client) GetClusterInfo(ctx context.Context) (map[string]interface{}, error) {
	info := make(map[string]interface{})

	// Get cluster version
	version, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get server version: %w", err)
	}
	info["version"] = version

	// Get node count
	nodes, err := c.GetNodeList(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}
	info["nodeCount"] = len(nodes.Items)

	// Get namespace count
	namespaces, err := c.GetNamespaces(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get namespaces: %w", err)
	}
	info["namespaceCount"] = len(namespaces.Items)

	return info, nil
} 