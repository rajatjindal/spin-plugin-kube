package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spinkube/spin-plugin-kube/pkg/doctor/provider"
	"github.com/spinkube/spin-plugin-kube/pkg/doctor/provider/k3d"
	"github.com/spinkube/spin-plugin-kube/pkg/doctor/provider/k8s"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "diagnose problems in the cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println()
		fmt.Println("#-------------------------------------")
		fmt.Println("# Running checks for SpinKube setup")
		fmt.Println("#-------------------------------------")
		fmt.Println()

		hint := "k8s"
		if strings.Contains(*configFlags.ClusterName, "k3d") {
			hint = "k3d"
		}

		// if len(os.Args) > 1 {
		// 	hint = os.Args[1]
		// }

		p, err := GetProvider(hint)
		if err != nil {
			return err
		}

		statusList, err := p.Status(context.Background())
		if err != nil {
			return err
		}

		var exitError = false
		for _, status := range statusList {
			if !status.Ok {
				exitError = true
			}

			provider.PrintStatus(status)
		}

		fmt.Println()

		if exitError {
			return fmt.Errorf("please fix above issues")
		}

		fmt.Println("\nAll looks good !!")

		return nil
	},
}

func init() {
	configFlags.AddFlags(doctorCmd.Flags())
	rootCmd.AddCommand(doctorCmd)
}

func GetProvider(hint string) (provider.Provider, error) {
	dc, err := getDynamicClient()
	if err != nil {
		return nil, err
	}

	sc, err := getKubernetesClientset()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	switch hint {
	case "k3d":
		return k3d.New(dc, sc), nil
	default:
		return k8s.New(dc, sc), nil
	}
}
