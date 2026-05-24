package runtime

import "sync"

type xmlModeInfo struct {
	Mode        string
	Namespace   string
	EGMLocation string
}

var xmlModeRegistry sync.Map

func rememberXMLModeInfo(instanceID string, info G2SXMLConfig) {
	if instanceID == "" {
		return
	}
	xmlModeRegistry.Store(instanceID, xmlModeInfo{
		Mode:        info.Mode,
		Namespace:   info.Namespace,
		EGMLocation: info.EGMLocation,
	})
}

func xmlModeInfoForInstance(instanceID string) xmlModeInfo {
	current, _ := xmlModeRegistry.Load(instanceID)
	info, _ := current.(xmlModeInfo)
	return info
}
