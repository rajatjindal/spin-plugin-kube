package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spinkube/spin-plugin-kube/pkg/debug"
	"k8s.io/kubectl/pkg/cmd/attach"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var attachOpts = &attach.AttachOptions{}
var component string

var debugCmd = &cobra.Command{
	Use:          "debug <name>",
	Short:        "Attach a debug spinapp with same permissions/access as the requested container",
	Hidden:       isExperimentalFlagNotSet,
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, args []string) error {
		var appName string
		if len(args) > 0 {
			appName = args[0]
		}

		if appName == "" && appNameFromCurrentDirContext != "" {
			appName = appNameFromCurrentDirContext
		}

		okey := client.ObjectKey{
			Namespace: namespace,
			Name:      appName,
		}

		app, err := kubeImpl.GetSpinApp(context.Background(), okey)
		if err != nil {
			return err
		}
		originalImg := app.Spec.Image

		debugImg, err := debug.DoIt(originalImg, component)
		if err != nil {
			return err
		}

		fmt.Println("debug img is ", debugImg)
		//edit spinapp to use new img
		app.Spec.Image = debugImg

		err = kubeImpl.UpdateSpinApp(context.Background(), &app)
		if err != nil {
			return err
		}

		//wait for new pod to start
		//todo
		timer := time.NewTimer(60 * time.Second)
		defer timer.Stop()

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

	OUTER:
		for {
			select {
			case <-timer.C:
				return fmt.Errorf("timeout when waiting for pod to be running")
			case <-ticker.C:
				app, err := kubeImpl.GetSpinApp(context.Background(), okey)
				if err != nil {
					return err
				}

				// fmt.Printf("expected: %d, got: %d\n", app.Spec.Replicas, app.Status.ReadyReplicas)
				if app.Status.ReadyReplicas == app.Spec.Replicas {
					break OUTER
				}
			}
		}

		//start an attach session
		factory, streams := NewCommandFactory()
		ccmd := attach.NewCmdAttach(factory, streams)

		attachOpts = attach.NewAttachOptions(streams)
		attachOpts.Stdin = true
		attachOpts.TTY = true

		deploy := fmt.Sprintf("deploy/%s", appName)
		cmdutil.CheckErr(attachOpts.Complete(factory, ccmd, []string{deploy}))
		cmdutil.CheckErr(attachOpts.Validate())
		cmdutil.CheckErr(attachOpts.Run())

		//cleanup and restore the original img
		fmt.Println()
		fmt.Println()
		fmt.Println("restoring the app")
		app.Spec.Image = originalImg

		err = kubeImpl.UpdateSpinApp(context.Background(), &app)
		if err != nil {
			return err
		}

		fmt.Println("restored the original app")
		return nil
	},
}

func init() {
	scaffoldCmd.Flags().StringVarP(&component, "component", "c", "", "Name of the component to debug")
	configFlags.AddFlags(debugCmd.Flags())
	rootCmd.AddCommand(debugCmd)
}
