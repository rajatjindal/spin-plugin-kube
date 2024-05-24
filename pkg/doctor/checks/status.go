package checks

import (
	"context"
	"fmt"

	"github.com/spinkube/spin-plugin-kube/pkg/doctor/provider"
	"gopkg.in/yaml.v2"
)

func Status(ctx context.Context, p provider.Provider) ([]provider.Status, error) {
	statusList := []provider.Status{}

	checks := []provider.Check{}
	err := yaml.Unmarshal(rawChecks, &checks)
	if err != nil {
		return nil, err
	}

	for _, check := range checks {
		checkfn, ok := defaultChecksMap[check.Type]
		if !ok {
			return nil, fmt.Errorf("check type %q not supported", check.Type)
		}

		overrideCheckFn := p.GetCheckOverride(ctx, check)
		if overrideCheckFn != nil {
			checkfn = overrideCheckFn
		}

		status, err := checkfn(ctx, p, check)
		if err != nil {
			return nil, err
		}

		statusList = append(statusList, status)
	}

	return statusList, nil
}
