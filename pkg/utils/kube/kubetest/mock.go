package kubetest

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/util/openapi"

	"github.com/argoproj/gitops-engine/pkg/utils/kube"
)

type KubectlOutput struct {
	Output string
	Err    error
}

type MockKubectlCmd struct {
	APIResources  []kube.APIResourceInfo
	Commands      map[string]KubectlOutput
	Events        chan watch.Event
	Version       string
	DynamicClient dynamic.Interface

	convertToVersionFunc *func(obj *unstructured.Unstructured, group, version string) (*unstructured.Unstructured, error)
	getResourceFunc      *func(ctx context.Context, config *rest.Config, gvk schema.GroupVersionKind, name string, namespace string) (*unstructured.Unstructured, error)
}

// WithConvertToVersionFunc overrides the default ConvertToVersion behavior.
func (k *MockKubectlCmd) WithConvertToVersionFunc(convertToVersionFunc func(*unstructured.Unstructured, string, string) (*unstructured.Unstructured, error)) *MockKubectlCmd {
	k.convertToVersionFunc = &convertToVersionFunc
	return k
}

// WithGetResourceFunc overrides the default ConvertToVersion behavior.
func (k *MockKubectlCmd) WithGetResourceFunc(getResourcefunc func(context.Context, *rest.Config, schema.GroupVersionKind, string, string) (*unstructured.Unstructured, error)) *MockKubectlCmd {
	k.getResourceFunc = &getResourcefunc
	return k
}

func (k *MockKubectlCmd) NewDynamicClient(config *rest.Config) (dynamic.Interface, error) {
	return k.DynamicClient, nil
}

func (k *MockKubectlCmd) GetAPIResources(config *rest.Config, preferred bool, resourceFilter kube.ResourceFilter) ([]kube.APIResourceInfo, error) {
	return k.APIResources, nil
}

func (k *MockKubectlCmd) GetResource(ctx context.Context, config *rest.Config, gvk schema.GroupVersionKind, name string, namespace string) (*unstructured.Unstructured, error) {
	if k.getResourceFunc != nil {
		return (*k.getResourceFunc)(ctx, config, gvk, name, namespace)
	}

	return nil, nil
}

func (k *MockKubectlCmd) PatchResource(ctx context.Context, config *rest.Config, gvk schema.GroupVersionKind, name string, namespace string, patchType types.PatchType, patchBytes []byte, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, nil
}

func (k *MockKubectlCmd) DeleteResource(ctx context.Context, config *rest.Config, gvk schema.GroupVersionKind, name string, namespace string, deleteOptions metav1.DeleteOptions) error {
	command, ok := k.Commands[name]
	if !ok {
		return nil
	}
	return command.Err
}

func (k *MockKubectlCmd) CreateResource(ctx context.Context, config *rest.Config, gvk schema.GroupVersionKind, name string, namespace string, obj *unstructured.Unstructured, subresources ...string) (*unstructured.Unstructured, error) {
	span := k.Tracer.StartSpan("CreateResource")
	span.SetBaggageItem("kind", gvk.Kind)
	span.SetBaggageItem("name", name)
	defer span.Finish()
	dynamicIf, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	disco, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}
	apiResource, err := ServerResourceForGroupVersionKind(disco, gvk, "create")
	if err != nil {
		return nil, err
	}
	resource := gvk.GroupVersion().WithResource(apiResource.Name)
	resourceIf := ToResourceInterface(dynamicIf, apiResource, resource, namespace)
	return resourceIf.Create(ctx, obj, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}}, subresources...)
}

// ConvertToVersion converts an unstructured object into the specified group/version
func (k *MockKubectlCmd) ConvertToVersion(obj *unstructured.Unstructured, group, version string) (*unstructured.Unstructured, error) {
	if k.convertToVersionFunc != nil {
		return (*k.convertToVersionFunc)(obj, group, version)
	}

	return obj, nil
}

func (k *MockKubectlCmd) GetServerVersion(config *rest.Config) (string, error) {
	return k.Version, nil
}

func (k *MockKubectlCmd) LoadOpenAPISchema(config *rest.Config) (openapi.Resources, *managedfields.GvkParser, error) {
	return nil, nil, nil
}

func (k *MockKubectlCmd) SetOnKubectlRun(onKubectlRun kube.OnKubectlRunFunc) {
}

func (k *MockKubectlCmd) ManageResources(config *rest.Config, openAPISchema openapi.Resources) (kube.ResourceOperations, func(), error) {
	return &MockResourceOps{}, func() {

	}, nil
}
