package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spinkube/spin-plugin-kube/pkg/debug"
	"k8s.io/kubectl/pkg/cmd/attach"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var attachOpts *attach.AttachOptions

var debugCmd = &cobra.Command{
	Use:    "debug <name>",
	Short:  "Attach a debug spinapp with same permissions/access as the requested container",
	Hidden: isExperimentalFlagNotSet,
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

		debugImg, err := debug.DoIt(app.Spec.Image)
		if err != nil {
			return err
		}

		fmt.Println("debug img is ", debugImg)
		//edit spinapp to use new img
		app.Spec.Image = debugImg

		k8sclient, err := getRuntimeClient()
		if err != nil {
			return err
		}

		err = k8sclient.Update(context.Background(), &app)
		if err != nil {
			return err
		}

		//wait for new pod to start
		//todo

		//start an attach session
		factory, streams := NewCommandFactory()
		ccmd := attach.NewCmdAttach(factory, streams)

		cmdutil.CheckErr(attachOpts.Complete(factory, ccmd, []string{}))
		cmdutil.CheckErr(attachOpts.Validate())
		cmdutil.CheckErr(attachOpts.Run())

		//cleanup and restore the original img

		return nil
	},
}

func init() {
	configFlags.AddFlags(debugCmd.Flags())
	rootCmd.AddCommand(debugCmd)
}
