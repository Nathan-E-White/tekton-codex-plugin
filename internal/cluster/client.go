package cluster

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	tektonv1 "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	Context    string
	Namespace  string
	Config     *rest.Config
	Kubernetes dynamic.Interface
	Tekton     tektonv1.TektonV1Interface
}

type DeploymentInfo struct {
	Name       string            `json:"name"`
	Generation int64             `json:"generation"`
	Images     []string          `json:"images"`
	Labels     map[string]string `json:"labels,omitempty"`
}

var (
	namespacesGVR               = schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}
	deploymentsGVR              = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	selfSubjectAccessReviewsGVR = schema.GroupVersionResource{Group: "authorization.k8s.io", Version: "v1", Resource: "selfsubjectaccessreviews"}
	tektonGVRs                  = []schema.GroupVersionResource{
		{Group: "tekton.dev", Version: "v1", Resource: "tasks"},
		{Group: "tekton.dev", Version: "v1", Resource: "pipelines"},
		{Group: "tekton.dev", Version: "v1", Resource: "taskruns"},
		{Group: "tekton.dev", Version: "v1", Resource: "pipelineruns"},
		{Group: "triggers.tekton.dev", Version: "v1beta1", Resource: "eventlisteners"},
		{Group: "triggers.tekton.dev", Version: "v1beta1", Resource: "triggerbindings"},
		{Group: "triggers.tekton.dev", Version: "v1beta1", Resource: "triggertemplates"},
	}
)

func New(contextName, namespace string) (*Client, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if value := os.Getenv("KUBECONFIG"); value != "" {
		rules.ExplicitPath = value
	}
	raw, err := rules.Load()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}
	if contextName == "" {
		return nil, fmt.Errorf("an explicit kubeconfig context is required")
	}
	if namespace == "" {
		return nil, fmt.Errorf("an explicit namespace is required")
	}
	if _, ok := raw.Contexts[contextName]; !ok {
		return nil, fmt.Errorf("kubeconfig context %q does not exist", contextName)
	}
	overrides := &clientcmd.ConfigOverrides{CurrentContext: contextName}
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("resolve kubeconfig context: %w", err)
	}
	kube, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	tekton, err := tektonv1.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &Client{Context: contextName, Namespace: namespace, Config: config, Kubernetes: kube, Tekton: tekton}, nil
}

func (c *Client) AccessReview(ctx context.Context, verb, group, resource, namespace string) (bool, string, error) {
	review := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "authorization.k8s.io/v1",
		"kind":       "SelfSubjectAccessReview",
		"spec": map[string]any{"resourceAttributes": map[string]any{
			"namespace": namespace, "verb": verb, "group": group, "resource": resource,
		}},
	}}
	result, err := c.Kubernetes.Resource(selfSubjectAccessReviewsGVR).Create(ctx, review, metav1.CreateOptions{})
	if err != nil {
		return false, "", err
	}
	allowed, _, _ := unstructured.NestedBool(result.Object, "status", "allowed")
	reason, _, _ := unstructured.NestedString(result.Object, "status", "reason")
	return allowed, reason, nil
}

func (c *Client) Identity(ctx context.Context) (string, error) {
	ns, err := c.Kubernetes.Resource(namespacesGVR).Get(ctx, "kube-system", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("resolve cluster identity: %w", err)
	}
	digest := sha256.Sum256([]byte(c.Config.Host + "\x00" + string(ns.GetUID())))
	return hex.EncodeToString(digest[:]), nil
}

func (c *Client) StateHash(ctx context.Context) (string, error) {
	state := map[string]any{}
	if namespace, err := c.Kubernetes.Resource(namespacesGVR).Get(ctx, c.Namespace, metav1.GetOptions{}); err == nil {
		state["scope"] = fmt.Sprintf("%s:%s", namespace.GetUID(), namespace.GetResourceVersion())
	} else {
		state["scope"] = "unavailable"
	}
	for _, namespace := range []string{"tekton-pipelines", "tekton-chains", "pipelines-as-code"} {
		deployments, err := c.Deployments(ctx, namespace)
		if err != nil {
			state[namespace] = "unavailable"
			continue
		}
		items := make([]string, 0, len(deployments))
		for _, deployment := range deployments {
			sort.Strings(deployment.Images)
			items = append(items, fmt.Sprintf("%s:%s:%d", deployment.Name, strings.Join(deployment.Images, ","), deployment.Generation))
		}
		sort.Strings(items)
		state[namespace] = items
	}
	for _, gvr := range tektonGVRs {
		list, err := c.Kubernetes.Resource(gvr).Namespace(c.Namespace).List(ctx, metav1.ListOptions{})
		key := gvr.Group + "/" + gvr.Resource
		if err != nil {
			state[key] = "unavailable"
			continue
		}
		items := make([]string, 0, len(list.Items))
		for _, item := range list.Items {
			items = append(items, fmt.Sprintf("%s:%s:%s:%d", item.GetName(), item.GetUID(), item.GetResourceVersion(), item.GetGeneration()))
		}
		sort.Strings(items)
		state[key] = items
	}
	b, _ := json.Marshal(state)
	digest := sha256.Sum256(b)
	return hex.EncodeToString(digest[:]), nil
}

func (c *Client) Deployments(ctx context.Context, namespace string) ([]DeploymentInfo, error) {
	list, err := c.Kubernetes.Resource(deploymentsGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	result := make([]DeploymentInfo, 0, len(list.Items))
	for _, item := range list.Items {
		containers, _, _ := unstructured.NestedSlice(item.Object, "spec", "template", "spec", "containers")
		images := []string{}
		for _, value := range containers {
			container, ok := value.(map[string]any)
			if !ok {
				continue
			}
			if image, ok := container["image"].(string); ok {
				images = append(images, image)
			}
		}
		result = append(result, DeploymentInfo{Name: item.GetName(), Generation: item.GetGeneration(), Images: images, Labels: item.GetLabels()})
	}
	return result, nil
}

func (c *Client) DataLossInventory(ctx context.Context) (map[string]int64, error) {
	result := map[string]int64{}
	for _, gvr := range tektonGVRs {
		list, err := c.Kubernetes.Resource(gvr).Namespace(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("enumerate %s: %w", gvr.Resource, err)
		}
		result[gvr.Group+"/"+gvr.Resource] = int64(len(list.Items))
	}
	return result, nil
}

func (c *Client) ListRuns(ctx context.Context, namespace, kind string, limit int64) (any, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	options := metav1.ListOptions{Limit: limit}
	switch strings.ToLower(kind) {
	case "taskrun", "taskruns":
		return c.Tekton.TaskRuns(namespace).List(ctx, options)
	case "pipelinerun", "pipelineruns", "":
		return c.Tekton.PipelineRuns(namespace).List(ctx, options)
	default:
		return nil, fmt.Errorf("unsupported run kind %q", kind)
	}
}

func (c *Client) GetRun(ctx context.Context, namespace, kind, name string) (any, error) {
	options := metav1.GetOptions{}
	switch strings.ToLower(kind) {
	case "taskrun", "taskruns":
		return c.Tekton.TaskRuns(namespace).Get(ctx, name, options)
	case "pipelinerun", "pipelineruns", "":
		return c.Tekton.PipelineRuns(namespace).Get(ctx, name, options)
	default:
		return nil, fmt.Errorf("unsupported run kind %q", kind)
	}
}
