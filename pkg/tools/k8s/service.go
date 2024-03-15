package k8s

import (
	"context"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// 实现controller与k8s交互的service接口
type Service interface {
	GetService(namespace string, name string) (*corev1.Service, error)
	CreateService(namespace string, service *corev1.Service) error
	UpdateService(namespace string, service *corev1.Service) error
	CreateUpdateService(namespace string, service *corev1.Service) error
	DeleteService(namespace string, service *corev1.Service) error
	CreateIfNotExistsService(namespace string, service *corev1.Service) error
	ListService(namespace string) (*corev1.ServiceList, error)
}

type SvcService struct {
	KubeClient kubernetes.Interface
	Logger     zap.Logger
}

func (s *SvcService) GetService(namespace string, name string) (*corev1.Service, error) {
	svc, err := s.KubeClient.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return svc, err
}

func (s *SvcService) CreateService(namespace string, service *corev1.Service) error {
	_, err := s.KubeClient.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	s.Logger.Info("staring create service", zap.Any("namespace", namespace), zap.Any("name", zap.Any("name", service.Name)))
	return nil
}

func (s *SvcService) UpdateService(namespace string, service *corev1.Service) error {
	_, err := s.KubeClient.CoreV1().Services(namespace).Update(context.TODO(), service, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	s.Logger.Info("staring update service", zap.Any("namespace", namespace), zap.Any("name", service.Name))
	return nil
}

func (s *SvcService) CreateUpdateService(namespace string, service *corev1.Service) error {
	cacheSvc, err := s.GetService(namespace, service.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			s.Logger.Info("GetService is not found,The svc will be created", zap.Any("serviceName", service.Name), zap.Any("namespace", namespace))
			return s.CreateService(namespace, service)
		}
		return err
	}
	service.ResourceVersion = cacheSvc.ResourceVersion
	service.Spec.ClusterIP = cacheSvc.Spec.ClusterIP

	return s.UpdateService(namespace, service)
}

func (s *SvcService) DeleteService(namespace string, service *corev1.Service) error {
	//Foreground 策略:当你删除一个资源时，该资源会被标记为删除中，但不会被立即从 Kubernetes 系统中移除。
	//Kubernetes 会等待该资源所有的依赖资源也被删除。例如，如果你删除一个 Deployment，那么所有属于这个 Deployment 的 Pod 会首先被删除。
	//一旦所有的依赖资源都被成功删除，该资源本身才会被从系统中移除
	//Background 策略：资源会立即被标记为删除，并且 Kubernetes 会尝试在后台删除所有的依赖资源。但即使依赖资源的删除失败，原始资源仍然会被从系统中移除。
	//Orphan 策略：只删除资源本身，不删除任何依赖资源。这些依赖资源会变成“孤儿”资源，需要手动管理。
	s.Logger.Info("The svc will be deleted", zap.Any("namespace", namespace), zap.Any("name", service.Name))
	propagation := metav1.DeletePropagationForeground
	return s.KubeClient.CoreV1().Services(namespace).Delete(context.TODO(), service.Name,
		metav1.DeleteOptions{PropagationPolicy: &propagation})
}

func (s *SvcService) ListService(namespace string) (*corev1.ServiceList, error) {
	return s.KubeClient.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
}

func (s *SvcService) CreateIfNotExistsService(namespace string, service *corev1.Service) error {
	_, err := s.GetService(namespace, service.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			s.Logger.Info("GetService is not found, Tne service will be created", zap.Any("namespacee", namespace),
				zap.Any("name", service.Name))
			return s.CreateService(namespace, service)
		}
		return err
	}
	return nil
}

// TODO
func NewSvcService(kubeclient kubernetes.Interface, logger zap.Logger) *SvcService {
	return &SvcService{
		Logger:     logger,
		KubeClient: kubeclient,
	}
}
