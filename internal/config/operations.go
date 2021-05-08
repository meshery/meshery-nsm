package config

import (
	"github.com/layer5io/meshery-adapter-library/adapter"
	"github.com/layer5io/meshery-adapter-library/common"
	"github.com/layer5io/meshery-adapter-library/meshes"
)

func getOperations(dev adapter.Operations) adapter.Operations {
	versions, _ := getLatestReleaseNames(3)

	dev[NSMMeshOperation] = &adapter.Operation{
		Type:                 int32(meshes.OpCategory_INSTALL),
		Description:          "NSM",
		Versions:             versions,
		Templates:            []adapter.Template{},
		AdditionalProperties: map[string]string{},
	}

	dev[NSMICMPResponderSampleApp] = &adapter.Operation{
		Type:        int32(meshes.OpCategory_SAMPLE_APPLICATION),
		Description: "ICMP Responder",
		AdditionalProperties: map[string]string{
			HelmChart:          "icmp-responder",
			common.ServiceName: "ICMP Responder",
		},
	}

	dev[NSMVPPICMPResponderSampleApp] = &adapter.Operation{
		Type:        int32(meshes.OpCategory_SAMPLE_APPLICATION),
		Description: "VPP ICMP Responder",
		AdditionalProperties: map[string]string{
			HelmChart:          "vpp-icmp-responder",
			common.ServiceName: "VPP ICMP Responder",
		},
	}

	dev[NSMVPMSampleApp] = &adapter.Operation{
		Type:        int32(meshes.OpCategory_SAMPLE_APPLICATION),
		Description: "VPN",
		AdditionalProperties: map[string]string{
			HelmChart:          "vpn",
			common.ServiceName: "VPN",
		},
	}

	return dev
}
