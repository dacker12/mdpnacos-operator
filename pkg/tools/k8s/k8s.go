package k8s

// 定义k8s 服务接口
type K8sService interface {
	ConfigMap
	StateFulSet
	Service
}
