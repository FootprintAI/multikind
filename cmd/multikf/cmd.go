package multikf

import (
	"errors"
	goflag "flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/log"

	"github.com/footprintai/multikf/pkg/machine"
	_ "github.com/footprintai/multikf/pkg/machine/host"
	_ "github.com/footprintai/multikf/pkg/machine/vagrant"
	"github.com/footprintai/multikf/pkg/version"
)

var (
	cpus           int    // number of cpus allocated to the geust machine
	memoryInG      int    // number of Gigabytes allocated to the guest machine
	provisionerStr string // provider specifies the underly privisoner for virtual machine, either docker (under host) or vagrant
	guestRootDir   string // root dir which containing multiple guest machines, each folder(i.e. $machinename) represents a single virtual machine configuration (default: ./.multilind)
	forceDelete    bool   // force to deleted the instance (default: false)
	forceCreate    bool   // force to create the instance regardless the instance's status (default: false)
	forceOverwrite bool   // force to overwrite the existing kubeconf file
	verbose        bool   // verbose (default: true)
	kubeconfigPath string // kubeconfig path of a guest machine (default: ./.mulitkind/$machine/kubeconfig)
	namespace      string // namespace
	withKubeflow   bool   // install with kubeflow components

	rootCmd = &cobra.Command{
		Use:   "multikf",
		Short: "a multikf cli tool",
		Long:  `multikf is a command-line tool which use vagrant and docker to provision Kubernetes and kubeflow single-node cluster.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// For cobra + glog flags. Available to all subcommands.
			goflag.Parse()
		},
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "version of multikf ",
		RunE: func(cmd *cobra.Command, args []string) error {
			version.Print()
			return nil
		},
	}

	exportCmd = &cobra.Command{
		Use:   "export <machine-name>",
		Short: "export kubeconfig from a guest machine",
		RunE: func(cmd *cobra.Command, args []string) error {
			run := mustNewRunCmd()
			return run.Export(args[0], kubeconfigPath)
		},
	}

	addCmd = &cobra.Command{
		Use:   "add <machine-name>",
		Short: "add a guest machine",
		RunE: func(cmd *cobra.Command, args []string) error {
			run := mustNewRunCmd()
			return run.Add(args[0], withKubeflow, cpus, memoryInG)
		},
	}
	deleteCmd = &cobra.Command{
		Use:   "delete <machine-name>",
		Short: "delete a guest machine",
		RunE: func(cmd *cobra.Command, args []string) error {
			run := mustNewRunCmd()
			return run.Delete(args[0])
		},
	}
	listCmd = &cobra.Command{
		Use:   "list",
		Short: "list guest machines",
		RunE: func(cmd *cobra.Command, args []string) error {
			run := mustNewRunCmd()
			return run.List()
		},
	}
	connectkubeflowCmd = &cobra.Command{
		Use:   "kubeflow command",
		Short: "kubeflow command",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("failed to recognize cluster name")
			}
			run := mustNewRunCmd()
			return run.ConnectKubeflow(args[0])
		},
	}
	connectCmd = &cobra.Command{
		Use:   "connect",
		Short: "connect",
	}
	getPodsCmd = &cobra.Command{
		Use:   "pods",
		Short: "pods",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("failed to recognize cluster name")
			}
			run := mustNewRunCmd()
			return run.GetPods(args[0])
		},
	}
	getCmd = &cobra.Command{
		Use:   "get",
		Short: "get",
	}
)

func mustNewRunCmd() *runCmd {
	cmd, err := newRunCmd()
	if err != nil {
		panic(err)
	}
	return cmd
}

func newRunCmd() (*runCmd, error) {
	p, err := machine.ParseProvisioner(provisionerStr)
	if err != nil {
		return nil, err
	}
	logger := cmd.NewLogger()

	vag, err := machine.NewMachineFactory(p, logger, guestRootDir, verbose)
	if err != nil {
		return nil, err
	}
	return &runCmd{vag: vag, logger: logger}, nil
}

type runCmd struct {
	vag    machine.MachinesCURD
	logger log.Logger
}

type machineConfig struct {
	cpus      int
	memoryInG int
}

func (m machineConfig) GetCPUs() int {
	return m.cpus
}

func (m machineConfig) GetMemory() int {
	return m.memoryInG
}

func (r *runCmd) Add(name string, withKubeflow bool, cpus, memoryInG int) error {
	m, err := r.vag.NewMachine(name, machineConfig{cpus: cpus, memoryInG: memoryInG})
	if err != nil {
		return err
	}
	if err := m.Up(forceCreate, withKubeflow); err != nil {
		r.logger.Errorf("runcmd: add node (%s) failed, err:%+v\n", name, err)
		return err
	}
	return nil
}

func (r *runCmd) Export(name string, path string) error {
	if path == "" {
		path = filepath.Join(guestRootDir, name, "kubeconfig")
	}
	m, err := r.vag.NewMachine(name, nil)
	if err != nil {
		return err
	}
	if err := m.ExportKubeConfig(path, forceOverwrite); err != nil {
		r.logger.Errorf("runcmd: export node (%s) failed, err:%+v\n", name, err)
		return err
	}
	return nil
}

func (r *runCmd) Delete(name string) error {
	m, err := r.vag.NewMachine(name, nil)
	if err != nil {
		return err
	}
	if err := m.Destroy(forceDelete); err != nil {
		r.logger.Errorf("runcmd: delete node (%s) failed, err:%+v\n", name, err)
		return err
	}
	return nil
}

// OutputMachineInfo defines the output format returned for each Machine
type OutputMachineInfo struct {
	Name       string `json:"name"`
	MachineDir string `json:"dir"`
	Status     string `json:"status"`
	Cpus       string `json:"cpus"`
	Gpus       string `json:"gpus"`
	KubeApi    string `json:"kubeAPI"`
	Memory     string `json:"memory"`
}

func (o *OutputMachineInfo) Headers() []string {
	return []string{
		"name",
		"dir",
		"status",
		"gpus",
		"kubeAPI",
		"cpus",
		"memory",
	}
}

func (o *OutputMachineInfo) Values() []string {
	return []string{
		o.Name,
		o.MachineDir,
		o.Status,
		o.Gpus,
		o.KubeApi,
		o.Cpus,
		o.Memory,
	}
}

var dummyRow = &OutputMachineInfo{}

func (r *runCmd) List() error {
	w := NewFormatWriter(os.Stdout, Table)
	machineList, err := r.vag.ListMachines()
	if err != nil {
		return err
	}
	machineNamesMap := map[string]*OutputMachineInfo{}
	for _, m := range machineList {
		info, err := m.Info()
		if err != nil {
			return err
		}
		machineNamesMap[m.Name()] = &OutputMachineInfo{
			Name:       m.Name(),
			MachineDir: m.HostDir(),
			Status:     info.Status,
			Gpus:       fmt.Sprintf("%s", info.GpuInfo.Info()),
			KubeApi:    info.KubeApi,
			Cpus:       fmt.Sprintf("%d", info.CpuInfo.NumCPUs()),
			Memory:     fmt.Sprintf("%s/%s", info.MemInfo.Free(), info.MemInfo.Total()),
		}
	}

	var csvValues [][]string
	for _, v := range machineNamesMap {
		csvValues = append(csvValues, v.Values())
	}
	return w.WriteAndClose(dummyRow.Headers(), csvValues)
}

func (r *runCmd) ConnectKubeflow(name string) error {
	m, err := r.vag.NewMachine(name, nil)
	if err != nil {
		return err
	}
	_, err = m.Portforward("svc/istio-ingressgateway", "istio-system", 80)
	if err != nil {
		r.logger.Errorf("runcmd: unable to connect %s failed, err:%+v\n", name, err)
		return err
	}
	return nil
}

func (r *runCmd) GetPods(name string) error {
	m, err := r.vag.NewMachine(name, nil)
	if err != nil {
		return err
	}
	err = m.GetPods(namespace)
	if err != nil {
		r.logger.Errorf("runcmd: failed to get pods, err:%+v\n", err)
		return err
	}
	return nil

}

func Main() {
	rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(connectCmd)
	connectCmd.AddCommand(connectkubeflowCmd)
	rootCmd.AddCommand(getCmd)
	getCmd.AddCommand(getPodsCmd)

	rootCmd.PersistentFlags().StringVar(&guestRootDir, "dir", ".multikfdir", "multikf root dir")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", true, "verbose (default: true)")
	rootCmd.PersistentFlags().StringVar(&provisionerStr, "provisioner", "docker", "provisioner, possible value: docker and vagrant")
	addCmd.Flags().IntVar(&cpus, "cpus", 1, "number of cpus allocated to the guest machine")
	addCmd.Flags().IntVar(&memoryInG, "memoryg", 1, "number of memory in gigabytes allocated to the guest machine")
	addCmd.Flags().BoolVar(&forceCreate, "f", false, "force to create instance regardless the machine status")
	addCmd.Flags().BoolVar(&withKubeflow, "with_kubeflow", true, "install kubeflow modules (default: true)")
	deleteCmd.Flags().BoolVar(&forceDelete, "f", false, "force remove the guest instance")
	exportCmd.Flags().StringVar(&kubeconfigPath, "kubeconfig_path", "", "force remove the guest instance")
	exportCmd.Flags().BoolVar(&forceOverwrite, "f", false, "force to overwrite the exiting file")
	getPodsCmd.Flags().StringVar(&namespace, "namespace", "", "namespace used (default: all-namespaces)")

	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}