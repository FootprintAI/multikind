package template

import (
	"bytes"
	"testing"

	"github.com/footprintai/multikf/pkg/machine"
	"github.com/stretchr/testify/assert"
)

var (
	_ KindConfiger = staticConfig{}
)

type staticConfig struct{}

func (s staticConfig) GetName() string {
	return "staticconfig"
}

func (s staticConfig) GetKubeAPIPort() int {
	return 8443
}

func (s staticConfig) GetKubeAPIIP() string {
	return "1.2.3.4"
}

func (s staticConfig) GetGPUs() int {
	return 1
}

func (s staticConfig) AuditEnabled() bool {
	return false
}

func (s staticConfig) AuditFileAbsolutePath() string {
	return ""
}

func (s staticConfig) LocalPath() string {
	return "/mnt/test"
}

func (s staticConfig) GetWorkers() []Worker {
	return []Worker{
		Worker{
			Id:        "1",
			UseGPU:    true,
			LocalPath: s.LocalPath(),
		},
		Worker{
			Id:        "2",
			UseGPU:    true,
			LocalPath: s.LocalPath(),
		},
		Worker{
			Id:        "3",
			UseGPU:    true,
			LocalPath: s.LocalPath(),
		},
	}
}

func (s staticConfig) GetNodeLabels() []machine.NodeLabel {
	return []machine.NodeLabel{
		machine.NodeLabel{
			Key:   "a",
			Value: "b",
		},
		machine.NodeLabel{
			Key:   "c",
			Value: "d",
		},
	}
}

func (s staticConfig) GetExportPorts() []machine.ExportPortPair {
	return []machine.ExportPortPair{
		machine.ExportPortPair{
			HostPort:      80,
			ContainerPort: 8081,
		},
		machine.ExportPortPair{
			HostPort:      443,
			ContainerPort: 8083,
		},
	}
}

func TestKindTemplate(t *testing.T) {
	kt := NewKindTemplate()
	assert.NoError(t, kt.Populate(staticConfig{}))
	buf := &bytes.Buffer{}
	assert.NoError(t, kt.Execute(buf))
	assert.EqualValues(t, gold, buf.String())
}

var gold = `
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: staticconfig
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
        node-labels: "a=b"
        node-labels: "c=d"
  image: kindest/node:v1.25.11@sha256:227fa11ce74ea76a0474eeefb84cb75d8dad1b08638371ecf0e86259b35be0c8
  gpus: true
  extraPortMappings:
  - containerPort: 8081
    hostPort: 80
    protocol: TCP
  - containerPort: 8083
    hostPort: 443
    protocol: TCP
  extraMounts:
  - hostPath: /mnt/test
    containerPath: /var/local-path-provisioner
- role: worker
  image: kindest/node:v1.25.11@sha256:227fa11ce74ea76a0474eeefb84cb75d8dad1b08638371ecf0e86259b35be0c8
  gpus: true
  extraMounts:
  - hostPath: /mnt/test
    containerPath: /var/local-path-provisioner
- role: worker
  image: kindest/node:v1.25.11@sha256:227fa11ce74ea76a0474eeefb84cb75d8dad1b08638371ecf0e86259b35be0c8
  gpus: true
  extraMounts:
  - hostPath: /mnt/test
    containerPath: /var/local-path-provisioner
- role: worker
  image: kindest/node:v1.25.11@sha256:227fa11ce74ea76a0474eeefb84cb75d8dad1b08638371ecf0e86259b35be0c8
  gpus: true
  extraMounts:
  - hostPath: /mnt/test
    containerPath: /var/local-path-provisioner
networking:
  apiServerAddress: 1.2.3.4
  apiServerPort: 8443
`
var (
	_ KindConfiger = auditConfig{}
)

type auditConfig struct{}

func (s auditConfig) GetName() string {
	return "auditConfig"
}

func (s auditConfig) GetKubeAPIPort() int {
	return 8443
}

func (s auditConfig) GetKubeAPIIP() string {
	return "1.2.3.4"
}

func (s auditConfig) GetGPUs() int {
	return 0
}

func (s auditConfig) GetExportPorts() []machine.ExportPortPair {
	return []machine.ExportPortPair{
		machine.ExportPortPair{
			HostPort:      80,
			ContainerPort: 8081,
		},
	}
}

func (s auditConfig) AuditEnabled() bool {
	return true
}

func (s auditConfig) AuditFileAbsolutePath() string {
	return "foo.bar.yaml"
}

func (s auditConfig) GetWorkers() []Worker {
	return []Worker{}
}

func (s auditConfig) GetNodeLabels() []machine.NodeLabel {
	return []machine.NodeLabel{}
}

func (s auditConfig) LocalPath() string {
	return ""
}

func TestKindTemplateWithAudit(t *testing.T) {
	kt := NewKindTemplate()
	assert.NoError(t, kt.Populate(auditConfig{}))
	buf := &bytes.Buffer{}
	assert.NoError(t, kt.Execute(buf))
	assert.EqualValues(t, goldWithAudit, buf.String())
}

var goldWithAudit = `
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: auditConfig
nodes:
- role: control-plane
  kubeadmConfigPatches:
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
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  image: kindest/node:v1.23.17@sha256:e5fd1d9cd7a9a50939f9c005684df5a6d145e8d695e78463637b79464292e66c
  gpus: false
  extraPortMappings:
  - containerPort: 8081
    hostPort: 80
    protocol: TCP
  extraMounts:
  - hostPath: foo.bar.yaml
    containerPath: /etc/kubernetes/policies/audit-policy.yaml
    readOnly: true
networking:
  apiServerAddress: 1.2.3.4
  apiServerPort: 8443
`
