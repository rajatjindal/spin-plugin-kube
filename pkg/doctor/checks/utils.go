package checks

import (
	"fmt"
	"strings"

	semver "github.com/Masterminds/semver/v3"
	dockerparser "github.com/novln/docker-parser"
	"github.com/spinkube/spin-plugin-kube/pkg/doctor/provider"
	v1 "k8s.io/api/apps/v1"
)

func compareVersions(version string, expectedSemVer []string) (bool, error) {
	actualVersion, err := semver.NewVersion(version)
	if err != nil {
		return false, err
	}

	for _, ver := range expectedSemVer {
		vcheck, err := semver.NewConstraint(ver)
		if err != nil {
			return false, err
		}

		ok, errlist := vcheck.Validate(actualVersion)
		if len(errlist) > 0 {
			continue
		}

		if ok {
			return ok, nil
		}
	}

	return false, fmt.Errorf("actual version: %q not one of the expected versions: %v", version, expectedSemVer)
}

// func getFileFromNode(_ context.Context, node v1.Node) error {
// 	cmd := exec.Command("kubectl", "debug", fmt.Sprintf("node/%s", node.Name), "-it", "--image", "ubuntu", "--", "cat", "/host/var/lib/rancher/k3s/agent/etc/containerd/config.toml")
// 	output, err := cmd.CombinedOutput()
// 	if err != nil {
// 		return err
// 	}

// 	fmt.Println("output from config.toml", string(output))
// 	// file -> /host/var/lib/rancher/k3s/agent/etc/containerd/config.toml

// 	return nil
// }

func getImageTag(deployment v1.Deployment, check provider.Check) string {
	for _, container := range deployment.Spec.Template.Spec.Containers {
		nameFromImgRef, tag, err := getNameFromImageReference(container.Image)
		if err != nil {
			continue
		}

		if nameFromImgRef != check.ImageName {
			continue
		}

		return tag
	}

	return ""
}

func getNameFromImageReference(imageRef string) (string, string, error) {
	ref, err := dockerparser.Parse(imageRef)
	if err != nil {
		return "", "", err
	}

	if strings.Contains(ref.ShortName(), "/") {
		parts := strings.Split(ref.ShortName(), "/")
		return parts[len(parts)-1], ref.Tag(), nil
	}

	return ref.ShortName(), ref.Tag(), nil
}
