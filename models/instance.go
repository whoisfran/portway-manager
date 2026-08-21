package models

// Instance representa una instancia administrada por SSM, enriquecida
// con datos de EC2 (nombre por tag, IP privada) cuando es posible.
type Instance struct {
	InstanceID string `json:"instanceId"`
	Name       string `json:"name"`
	PlatformOS string `json:"platformOs"`
	PrivateIP  string `json:"privateIp"`
	PingStatus string `json:"pingStatus"`
}
