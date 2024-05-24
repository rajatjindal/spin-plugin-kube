package checks

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/spinkube/spin-plugin-kube/pkg/doctor/provider"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/kubectl/pkg/cmd/debug"
)

const (
	CheckCRD                      = "crd"
	CheckContainerdVersionOnNodes = "containerd-version-on-nodes"
	CheckRuntimeClass             = "runtimeclass"
	CheckDeploymentRunning        = "deployment-running"
	CheckBinaryInstalledOnNodes   = "binary-installed-on-nodes"
)

var defaultChecksMap = map[string]provider.CheckFn{
	CheckCRD:                      isCrdInstalled,
	CheckContainerdVersionOnNodes: containerdVersionCheck,
	CheckRuntimeClass:             runtimeClassCheck,
	CheckDeploymentRunning:        deploymentRunningCheck,
	CheckBinaryInstalledOnNodes:   binaryVersionCheck,
}

//go:embed data/checks.yaml
var rawChecks []byte

var isCrdInstalled = func(ctx context.Context, k provider.Provider, check provider.Check) (provider.Status, error) {
	_, err := k.DynamicClient().Resource(schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}).Get(ctx, check.ResourceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return provider.Status{
				Name:     check.Name,
				Ok:       false,
				HowToFix: check.HowToFix,
			}, nil
		}

		return provider.Status{
			Name:     check.Name,
			Ok:       false,
			HowToFix: check.HowToFix,
		}, err
	}

	return provider.Status{
		Name: check.Name,
		Ok:   true,
	}, nil
}

var runtimeClassCheck = func(ctx context.Context, k provider.Provider, check provider.Check) (provider.Status, error) {
	_, err := k.Client().NodeV1().RuntimeClasses().Get(ctx, check.ResourceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return provider.Status{
				Name: check.Name,
				Ok:   false,
			}, nil
		}

		return provider.Status{
			Name: check.Name,
			Ok:   false,
		}, err
	}

	return provider.Status{
		Name: check.Name,
		Ok:   true,
	}, nil
}

var deploymentRunningCheck = func(ctx context.Context, k provider.Provider, check provider.Check) (provider.Status, error) {
	resp, err := k.Client().AppsV1().Deployments(v1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return provider.Status{
				Name:     check.Name,
				Ok:       false,
				HowToFix: check.HowToFix,
			}, nil
		}

		return provider.Status{
			Name:     check.Name,
			Ok:       false,
			HowToFix: check.HowToFix,
		}, err
	}

	//TODO: handle pagination
	for _, item := range resp.Items {
		if item.Name == check.ResourceName {
			if len(check.SemVer) > 0 {
				imageTag := getImageTag(item, check)
				ok, err := compareVersions(imageTag, check.SemVer)
				if err != nil {
					return provider.Status{
						Name:     check.Name,
						Ok:       false,
						Msg:      fmt.Sprintf("deployment running, but failed to do version check: %v", err),
						HowToFix: check.HowToFix,
					}, nil
				}

				if !ok {
					return provider.Status{
						Name:     check.Name,
						Ok:       false,
						Msg:      fmt.Sprintf("deployment running, but version check failed: %v", err),
						HowToFix: check.HowToFix,
					}, nil
				}
			}

			return provider.Status{
				Name: check.Name,
				Ok:   true,
			}, nil
		}
	}

	return provider.Status{
		Name:     check.Name,
		Ok:       false,
		HowToFix: check.HowToFix,
	}, nil
}

var containerdVersionCheck = func(ctx context.Context, k provider.Provider, check provider.Check) (provider.Status, error) {
	resp, err := k.Client().CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return provider.Status{}, err
	}

	vok := true
	msgs := []string{}

	for _, node := range resp.Items {
		if !strings.Contains(node.Status.NodeInfo.ContainerRuntimeVersion, "containerd") {
			vok = false
			msgs = append(msgs, fmt.Sprintf("found container runtime %q instead of containerd", node.Status.NodeInfo.ContainerRuntimeVersion))
			continue
		}

		version := strings.ReplaceAll(node.Status.NodeInfo.ContainerRuntimeVersion, "containerd://", "")
		ok, err := compareVersions(version, check.SemVer)
		if err != nil {
			vok = false
			msgs = append(msgs, err.Error())
			continue
		}

		if !ok {
			vok = false
			msgs = append(msgs, fmt.Sprintf("  - node: %s with containerd version %s does not support SpinApps", node.Name, node.Status.NodeInfo.ContainerRuntimeVersion))
			continue
		}
	}

	return provider.Status{
		Name:     check.Name,
		Ok:       vok,
		Msg:      strings.Join(msgs, "\n"),
		HowToFix: check.HowToFix,
	}, nil
}

var binaryVersionCheck = func(ctx context.Context, k provider.Provider, check provider.Check) (provider.Status, error) {
	knownBinPaths := []string{
		"/bin",
		"/usr/local/bin",
		"/usr/bin",
	}

	for _, binPath := range knownBinPaths {
		hostAbsBinPath := filepath.Join("/host", binPath, check.ResourceName)
		status, err := ExecOnEachNodeFn(ctx, k, check, []string{hostAbsBinPath}, []string{"-v"})
		if err != nil {
			continue
		}

		return status, nil
	}

	return provider.Status{}, fmt.Errorf("not sure")
}

var ExecOnEachNodeFn = func(ctx context.Context, k provider.Provider, check provider.Check, cmd []string, args []string) (provider.Status, error) {
	resp, err := k.Client().CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return provider.Status{}, err
	}

	vok := true
	msgs := []string{}

	for _, node := range resp.Items {
		output, err := execOnOneNodeFn(ctx, k, &node, check, cmd, args)
		if err != nil {
			vok = false
			continue
		}

		for _, v := range check.SemVer {
			// TODO: add semver check instead of contains check
			if strings.Contains(output, v) {
				vok = true
				break
			}
		}
	}

	return provider.Status{
		Name:     check.Name,
		Ok:       vok,
		Msg:      strings.Join(msgs, "\n"),
		HowToFix: check.HowToFix,
	}, nil
}

var execOnOneNodeFn = func(ctx context.Context, k provider.Provider, node *v1.Node, check provider.Check, cmd []string, args []string) (string, error) {
	pspec, err := generateNodeDebugPod(node, cmd, args)
	if err != nil {
		return "", fmt.Errorf("failed to generateNodeDebugPod %v", err)
	}

	_, err = k.Client().CoreV1().Pods("default").Create(ctx, pspec, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create debug pod %v", err)
	}
	defer func() {
		k.Client().CoreV1().Pods("default").Delete(ctx, pspec.Name, metav1.DeleteOptions{})
	}()

	<-time.NewTimer(5 * time.Second).C

	req := k.Client().CoreV1().Pods("default").GetLogs(pspec.Name, &v1.PodLogOptions{})
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get logs from pod %v", err)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf("failed to copy logs from buffer %v", err)
	}

	return buf.String(), nil
}

// generateNodeDebugPod generates a debugging pod that schedules on the specified node.
// The generated pod will run in the host PID, Network & IPC namespaces, and it will have the node's filesystem mounted at /host.
func generateNodeDebugPod(node *v1.Node, cmd []string, args []string) (*v1.Pod, error) {
	cn := "shim-version-checker"

	// The name of the debugging pod is based on the target node, and it's not configurable to
	// limit the number of command line flags. There may be a collision on the name, but this
	// should be rare enough that it's not worth the API round trip to check.
	pn := fmt.Sprintf("spin-kube-doctor-%s-%s", node.Name, utilrand.String(5))

	p := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: pn,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:                     cn,
					Image:                    "ubuntu:latest",
					Stdin:                    true,
					TerminationMessagePolicy: v1.TerminationMessageReadFile,
					TTY:                      true,
					Command:                  cmd,
					Args:                     args,
				},
			},
			NodeName:      node.Name,
			RestartPolicy: v1.RestartPolicyNever,
			Tolerations: []v1.Toleration{
				{
					Operator: v1.TolerationOpExists,
				},
			},
		},
	}

	profiler, err := debug.NewProfileApplier("legacy")
	if err != nil {
		return nil, err
	}

	if err := profiler.Apply(p, cn, node); err != nil {
		return nil, err
	}

	return p, nil
}
