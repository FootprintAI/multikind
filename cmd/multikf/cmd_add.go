package multikf

import (
	"errors"
	"fmt"
	"strings"

	kfmanifests "github.com/footprintai/multikf/kfmanifests"
	"github.com/footprintai/multikf/pkg/k8s"
	"github.com/footprintai/multikf/pkg/machine"
	"github.com/footprintai/multikf/pkg/machine/plugins"
	"github.com/footprintai/multikf/pkg/machine/vagrant"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/kind/pkg/log"
)

func NewAddCommand(logger log.Logger, ioStreams genericclioptions.IOStreams) *cobra.Command {
	var (
		provisionerStr              string // provider specifies the underly privisoner for virtual machine, either docker (under host) or vagrant
		cpus                        int    // number of cpus allocated to the geust machine
		memoryInG                   int    // number of Gigabytes allocated to the guest machine
		useGPUs                     int    // use GPU resources
		withKubeflow                bool   // install with kubeflow components
		withKubeflowVersion         string // with kubeflow version
		withKubeflowDefaultPassword string // with kubeflow defaultpassword
		withIP                      string // with specific IP
		withAudit                   bool   // with audit enabled
		withWorkers                 int    // with workers
		withLabels                  string // with labels
		exportPorts                 string // export ports on hostmachine
		forceOverwrite              bool   // force overwrite existing config
		useLocalPath                string // with localpath
		withK8sVersion              string
		withK8sSHA256               string
	)

	ensureNoGPUForVagrant := func(vag machine.MachineCURDFactory, useGPUs int) error {
		if _, isVargant := vag.(*vagrant.VagrantMachines); isVargant && useGPUs > 0 {
			return errors.New("vagrant machine haven't support gpu passthrough yet")
		}
		return nil
	}

	handle := func(machineName string) error {
		vag, err := newMachineFactoryWithProvisioner(
			machine.MustParseProvisioner(provisionerStr),
			logger,
		)
		if err != nil {
			return err
		}
		if err := ensureNoGPUForVagrant(vag, useGPUs); err != nil {
			return err
		}

		m, err := vag.NewMachine(machineName, machineConfig{
			logger:         logger,
			Cpus:           cpus,
			MemoryInG:      memoryInG,
			UseGPUs:        useGPUs,
			KubeAPIIP:      withIP,
			ExportPorts:    exportPorts,
			ForceOverwrite: forceOverwrite,
			IsAuditEnabled: withAudit,
			Workers:        withWorkers,
			NodeLabels:     withLabels,
			LocalPath:      useLocalPath,
			NodeVersion: k8s.NewKindK8sVersion(
				withK8sVersion,
				withK8sSHA256,
			),
		})
		if err != nil {
			return err
		}
		if err := m.Up(); err != nil {
			logger.Errorf("cmdadd: add node (%s) failed, err:%+v\n", machineName, err)
			return err
		}
		var installedPlugins []plugins.Plugin
		if withKubeflow {
			installedPlugins = append(installedPlugins,
				kubeflowPlugin{withKubeflowDefaultPassword: withKubeflowDefaultPassword, kubeflowVersion: plugins.NewTypePluginVersion(withKubeflowVersion)},
			)
		}
		return plugins.AddPlugins(m, installedPlugins...)
	}
	cmd := &cobra.Command{
		Use:   "add <machine-name>",
		Short: "add a guest machine",
		RunE: func(cmd *cobra.Command, args []string) error {
			return handle(args[0])
		},
	}
	kfVersions := kfmanifests.ListVersions()

	cmd.Flags().StringVar(&provisionerStr, "provisioner", "docker", "provisioner, possible value: docker and vagrant")
	cmd.Flags().IntVar(&cpus, "cpus", 1, "number of cpus allocated to the guest machine")
	cmd.Flags().IntVar(&memoryInG, "memoryg", 1, "number of memory in gigabytes allocated to the guest machine")
	cmd.Flags().BoolVar(&forceOverwrite, "f", false, "force to overwrite existing config. (default: false)")
	cmd.Flags().BoolVar(&withKubeflow, "with_kubeflow", true, "install kubeflow modules (default: true)")
	cmd.Flags().StringVar(&withKubeflowVersion, "kubeflow_version", kfVersions[0], fmt.Sprintf("support kubeflow version: %s", strings.Join(kfVersions, ",")))
	cmd.Flags().BoolVar(&withAudit, "with_audit", true, "enable k8s auditing (default: true)")
	cmd.Flags().StringVar(&withKubeflowDefaultPassword, "with_password", "12341234", "with a specific password for default user (default: 12341234)")
	cmd.Flags().IntVar(&useGPUs, "use_gpus", 0, "use gpu resources (default: 0), possible value (0 or 1)")
	cmd.Flags().StringVar(&withIP, "with_ip", "0.0.0.0", "with a specific ip address for kubeapi (default: 0.0.0.0)")
	cmd.Flags().StringVar(&exportPorts, "export_ports", "", "export ports to host, delimited by comma(example: 8443:443 stands for mapping host port 8443 to container port 443)")
	cmd.Flags().IntVar(&withWorkers, "with_workers", 0, "use workers (default: 0)")
	cmd.Flags().StringVar(&withLabels, "with_labels", "", "attach labels, format: key1=value1,key2=value2(default: )")
	cmd.Flags().StringVar(&useLocalPath, "use_localpath", "", "mount local path to kind cluster")
	cmd.Flags().StringVar(&withK8sVersion, "with_k8s_version", k8s.DefaultVersion().Version(), fmt.Sprintf("support verisions:%s", strings.Join(k8s.ListVersionString(), ",")))
	cmd.Flags().StringVar(&withK8sSHA256, "with_k8s_sha256", k8s.DefaultVersion().Sha256(), fmt.Sprintf("k8s version and its sha256 mapping list:%s", strings.Join(k8s.ListVersionSha256String(), ",")))

	return cmd
}
