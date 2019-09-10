package nsm

type supportedOperation struct {
	// a friendly name
	name string
	// the template file name
	templateName string
	// the app label
	appLabel string
	// // returnLogs specifies if the operation logs should be returned
	// returnLogs bool
}

const (
	customOpCommand    = "custom"
	installNSMCommand  = "nsm_install"
	installVPNCommand = "vpn_install"
)

var supportedOps = map[string]supportedOperation{
	installNSMCommand: {
		name: "Install Network Service Mesh",
	},
	installVPNCommand: {
		name: "Install VPN Application",
	},
	customOpCommand: {
		name: "Custom YAML",
	},
}
