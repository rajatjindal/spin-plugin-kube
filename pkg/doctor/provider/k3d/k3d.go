package k3d

import (
	"context"

	"github.com/spinkube/spin-plugin-kube/pkg/doctor/checks"
	"github.com/spinkube/spin-plugin-kube/pkg/doctor/provider"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type k3d struct {
	k8sclient kubernetes.Interface
	dc        dynamic.Interface
}

func New(dc dynamic.Interface, sc kubernetes.Interface) provider.Provider {
	return &k3d{
		k8sclient: sc,
		dc:        dc,
	}
}

func (k *k3d) Name() string {
	return "k3d"
}

func (k *k3d) Client() kubernetes.Interface {
	return k.k8sclient
}

func (k *k3d) DynamicClient() dynamic.Interface {
	return k.dc
}

func (k *k3d) Status(ctx context.Context) ([]provider.Status, error) {
	return checks.Status(ctx, k)
}

func (k *k3d) GetCheckOverride(ctx context.Context, check provider.Check) provider.CheckFn {
	switch check.Type {
	case checks.CheckBinaryInstalledOnNodes:
		return binaryVersionCheck
	}

	return nil
}

var binaryVersionCheck = func(ctx context.Context, k provider.Provider, check provider.Check) (provider.Status, error) {
	return checks.ExecOnEachNodeFn(ctx, k, check, []string{"/host/bin/containerd-shim-spin-v2"}, []string{"-v"})
}
