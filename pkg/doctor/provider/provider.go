package provider

import (
	"context"
	"fmt"

	"github.com/spinkube/spin-plugin-kube/pkg/doctor/icons"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type Status struct {
	Name     string
	Ok       bool
	Msg      string
	HowToFix string
}

type Check struct {
	Name         string   `yaml:"name"`
	Type         string   `yaml:"checkType"`
	ResourceName string   `yaml:"resourceName"`
	SemVer       []string `yaml:"semver"`
	ImageName    string   `yaml:"imageName"`
	HowToFix     string   `yaml:"howToFix"`
}

type CheckFn func(ctx context.Context, k Provider, check Check) (Status, error)

type Provider interface {
	Name() string
	Client() kubernetes.Interface
	DynamicClient() dynamic.Interface
	Status(ctx context.Context) ([]Status, error)
	GetCheckOverride(ctx context.Context, check Check) CheckFn
}

func PrintStatus(s Status) {
	if s.Ok {
		fmt.Printf("%s %s", icons.IconWhiteCheckmark, s.Name)
	} else {
		fmt.Printf("%s %s", icons.IconRedCross, s.Name)

		if s.Msg != "" {
			fmt.Printf("\n-> %s\n", s.Msg)
		}
	}
	fmt.Println()
}
