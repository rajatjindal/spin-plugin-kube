package k8s

import (
	"context"

	"github.com/spinkube/spin-plugin-kube/pkg/doctor/checks"
	"github.com/spinkube/spin-plugin-kube/pkg/doctor/provider"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type k8s struct {
	k8sclient kubernetes.Interface
	dc        dynamic.Interface
}

func New(dc dynamic.Interface, sc kubernetes.Interface) provider.Provider {
	return &k8s{
		k8sclient: sc,
		dc:        dc,
	}
}

func (k *k8s) Name() string {
	return "k8s"
}

func (k *k8s) Client() kubernetes.Interface {
	return k.k8sclient
}

func (k *k8s) DynamicClient() dynamic.Interface {
	return k.dc
}

func (k *k8s) Status(ctx context.Context) ([]provider.Status, error) {
	return checks.Status(ctx, k)
}

func (k *k8s) GetCheckOverride(ctx context.Context, check provider.Check) provider.CheckFn {
	switch check.Type {
	case checks.CheckBinaryInstalledOnNodes:
		return binaryVersionCheck
	}

	return nil
}

var binaryVersionCheck = func(ctx context.Context, k provider.Provider, check provider.Check) (provider.Status, error) {
	return checks.ExecOnEachNodeFn(ctx, k, check, []string{"/host/opt/kwasm/bin/containerd-shim-spin-v2"}, []string{"-v"})
}
