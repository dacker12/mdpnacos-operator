package nacos

type MdpNacosOpClient interface {
}

type OpClient struct {
	BuildClient  *BuildClient
	CheckClient  *CheckClient
	HealthClient *HealthClient
	StatusClient *StatusClient
}
