package template

import (
	"bytes"
	"testing"

	"github.com/footprintai/multikf/pkg/machine"
	"github.com/stretchr/testify/assert"
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
  image: kindest/node:v1.21.2
  gpus: true
  extraPortMappings:
  - containerPort: 8081
    hostPort: 80
    protocol: TCP
  - containerPort: 8083
    hostPort: 443
    protocol: TCP
networking:
  apiServerAddress: 1.2.3.4
  apiServerPort: 8443
`