package template

import (
	"fmt"
	"html/template"
	"io"

	"github.com/footprintai/multikf/pkg/machine"
)

func NewKindTemplate() *KindFileTemplate {
	return &KindFileTemplate{
		kindFileTemplate: kindDefaultFileTemplate,
	}
}

func (k *KindFileTemplate) Filename() string {
	return "kind-config.yaml"
}

func (k *KindFileTemplate) Execute(w io.Writer) error {
	tmpl, err := template.New("kindconfig").Parse(k.kindFileTemplate)
	if err != nil {
		return err
	}
	if err := tmpl.Execute(w, k); err != nil {
		return err
	}
	return nil
}

type KindConfiger interface {
	NameGetter
	KubeAPIPortGetter
	KubeAPIIPGetter
	GpuGetter
	ExportPortsGetter
	AuditEnabler
	WorkerIDsGetter
}

func (k *KindFileTemplate) Populate(v interface{}) error {
	if _, isKindConfiger := v.(KindConfiger); !isKindConfiger {
		return fmt.Errorf("not implements kindConfig interface")
	}
	c := v.(KindConfiger)
	k.Name = c.GetName()
	k.KubeAPIPort = c.GetKubeAPIPort()
	k.KubeAPIIP = c.GetKubeAPIIP()
	k.UseGPU = c.GetGPUs() > 0
	k.ExportPorts = c.GetExportPorts()
	k.AuditEnabled = c.AuditEnabled()
	k.AuditFileAbsolutePath = c.AuditFileAbsolutePath()
	k.WorkerIDs = c.GetWorkerIDs()

	return nil
}

type KindFileTemplate struct {
	Name                  string
	KubeAPIIP             string
	KubeAPIPort           int
	UseGPU                bool
	kindFileTemplate      string
	ExportPorts           []machine.ExportPortPair
	AuditEnabled          bool
	AuditFileAbsolutePath string
	WorkerIDs             []int
}

var kindDefaultFileTemplate string = `
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: {{.Name}}
nodes:
- role: control-plane
  kubeadmConfigPatches:
  {{- if .AuditEnabled}}
  - |
    kind: ClusterConfiguration
    apiServer:
      # enable auditing flags on the API server
      extraArgs:
        audit-log-path: /var/log/kubernetes/kube-apiserver-audit.log
        audit-policy-file: /etc/kubernetes/policies/audit-policy.yaml
        audit-log-maxage: "30"
        audit-log-maxbackup: "10"
        audit-log-maxsize: "100"
      # mount new files / directories on the control plane
      extraVolumes:
        - name: audit-policies
          hostPath: /etc/kubernetes/policies
          mountPath: /etc/kubernetes/policies
          readOnly: true
          pathType: "DirectoryOrCreate"
        - name: "audit-logs"
          hostPath: "/var/log/kubernetes"
          mountPath: "/var/log/kubernetes"
          readOnly: false
          pathType: DirectoryOrCreate
  {{- end}}
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  image: kindest/node:v1.23.12@sha256:9402cf1330bbd3a0d097d2033fa489b2abe40d479cc5ef47d0b6a6960613148a
  gpus: {{.UseGPU}}
  {{if .ExportPorts}}extraPortMappings:{{end}}
  {{- range $i, $p := .ExportPorts}}
  - containerPort: {{ $p.ContainerPort }}
    hostPort: {{ $p.HostPort }}
    protocol: TCP
  {{- end}}
  {{- if .AuditEnabled}}
  # mount the local file on the control plane
  extraMounts:
  - hostPath: {{.AuditFileAbsolutePath}}
    containerPath: /etc/kubernetes/policies/audit-policy.yaml
    readOnly: true
  {{- end}}
{{- range .WorkerIDs }}
- role: worker
  image: kindest/node:v1.23.12@sha256:9402cf1330bbd3a0d097d2033fa489b2abe40d479cc5ef47d0b6a6960613148a
{{- end}}
networking:
  apiServerAddress: {{.KubeAPIIP}}
  apiServerPort: {{.KubeAPIPort}}
`
