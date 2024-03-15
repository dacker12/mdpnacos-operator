package k8s

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"strings"
)

type StateFulSet interface {
	GetStateFulSet(namespace, name string) (*appsv1.StatefulSet, error)
	CreateStateFulSet(namespace string, statefulset *appsv1.StatefulSet) error
	DeleteStateFulSet(namespace, name string) error
	GetStateFulSetPodS(namespace, name string) (*corev1.PodList, error)
	UpdateStateFulSet(namespace string, statefulset *appsv1.StatefulSet) error
	ListStateFulSet(namespace string) (*appsv1.StatefulSetList, error)
	CreateAndUpdateStateFulSet(namespace string, statefulset *appsv1.StatefulSet) error
	GetStateFulSetReadyPods(namespace, name string) ([]corev1.Pod, error)
}

type StateFulSetAbstraction struct {
	KubeClient kubernetes.Interface
	Logger     zap.Logger
}

// 找出statefulset pod中运行状态ready为true的pod列表
func (s *StateFulSetAbstraction) GetStateFulSetReadyPods(namespace, name string) ([]corev1.Pod, error) {
	var readypod []corev1.Pod
	podList, err := s.GetStateFulSetPodS(namespace, name)
	if err != nil {
		s.Logger.Error("GetStateFulSetPods failed", zap.Error(err))
		return readypod, err
	}

	num := 0
	//遍历pod列表
	for _, pod := range podList.Items {
		//TODO
		//这里为什么是4呢？？？？
		if len(pod.Status.Conditions) != 4 {
			continue
		}
		if pod.Status.Conditions[1].Type == "Ready" &&
			pod.Status.Conditions[1].Status == "True" {
			num = num + 1
			readypod = append(readypod, pod)
		}
	}
	return readypod, nil
}

func (s *StateFulSetAbstraction) GetStateFulSet(namespace, name string) (*appsv1.StatefulSet, error) {
	stateful, err := s.KubeClient.AppsV1().StatefulSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		s.Logger.Error("GetStateFulSet fails", zap.Error(err))
		return nil, err
	}
	return stateful, nil
}

func (s *StateFulSetAbstraction) CreateStateFulSet(namespace string, statefulset *appsv1.StatefulSet) error {
	_, err := s.KubeClient.AppsV1().StatefulSets(namespace).Create(context.TODO(), statefulset, metav1.CreateOptions{})
	if err != nil {
		s.Logger.Error("StateFulSet creation failed", zap.Error(err))
		return err
	}
	return nil
}

func (s *StateFulSetAbstraction) DeleteStateFulSet(namespace, name string) error {
	propagation := metav1.DeletePropagationForeground
	err := s.KubeClient.AppsV1().StatefulSets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{PropagationPolicy: &propagation})
	if err != nil {
		s.Logger.Error("StateFulSet deleting failed", zap.Error(err))
		return err
	}
	return nil
}

func (s *StateFulSetAbstraction) GetStateFulSetPodS(namespace, name string) (*corev1.PodList, error) {
	statefulset, err := s.GetStateFulSet(namespace, name)
	if err != nil {
		s.Logger.Error("GetStateFulSet request failed", zap.Error(err))
		return nil, err
	}

	labels := []string{}
	for k, v := range statefulset.Spec.Selector.MatchLabels {
		labels = append(labels, fmt.Sprintf("%s=%s", k, v))
	}
	selector := strings.Join(labels, ",")
	return s.KubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector})
}

func (s *StateFulSetAbstraction) UpdateStateFulSet(namespace string, statefulset *appsv1.StatefulSet) error {
	_, err := s.KubeClient.AppsV1().StatefulSets(namespace).Update(context.TODO(), statefulset, metav1.UpdateOptions{})
	if err != nil {
		s.Logger.Error("UpdateStateFulSet request failed", zap.Error(err))
		return err
	}
	s.Logger.Info("UpdateStateFulSet request successful", zap.Any("namespace", namespace), zap.Any("name", statefulset.Name))
	return err
}

func (s *StateFulSetAbstraction) ListStateFulSet(namespace string) (*appsv1.StatefulSetList, error) {
	return s.KubeClient.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{})
}

func (s *StateFulSetAbstraction) CreateAndUpdateStateFulSet(namespace string, statefulset *appsv1.StatefulSet) error {
	cachests, err := s.GetStateFulSet(namespace, statefulset.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			s.Logger.Info("cachests is not found, About to create cachests", zap.Any("namespace", namespace),
				zap.Any("name", statefulset.Name))
			err := s.CreateStateFulSet(namespace, statefulset)
			if err != nil {
				s.Logger.Error("stateFulset create failed", zap.Error(err))
			}
		}
		s.Logger.Error("GetStateFulSet request failed", zap.Error(err))
		return err
	}
	//根据新、旧版本模版判断做何操作（删除、更新或不作处理）
	switch CompareNewAdnOldSTS(cachests, statefulset) {

	case Update:
		statefulset.ResourceVersion = cachests.ResourceVersion
		return s.UpdateStateFulSet(namespace, statefulset)

	case Delete:
		return s.DeleteStateFulSet(namespace, statefulset.Name)
	}
	return nil
}

// 定义操作选项
type OperatorOptions int

const (
	None   OperatorOptions = 0
	Update OperatorOptions = 1
	Delete OperatorOptions = 2
)

// CompareNewAndOldSTS 检查是更新/删除StateFulset
func CompareNewAdnOldSTS(old *appsv1.StatefulSet, new *appsv1.StatefulSet) OperatorOptions {
	oldrs, _ := json.Marshal(old.Spec.Template.Spec.Containers[0].Resources)
	newrs, _ := json.Marshal(new.Spec.Template.Spec.Containers[0].Resources)

	oldEnv, _ := json.Marshal(old.Spec.Template.Spec.Containers[0].Env)
	newEnv, _ := json.Marshal(new.Spec.Template.Spec.Containers[0].Env)

	if CompareNewAndOldVolume(old, new) {
		return Delete
	}

	if !bytes.Equal(oldrs, newrs) || *old.Spec.Replicas != *new.Spec.Replicas || bytes.Equal(oldEnv, newEnv) {
		return Update
	}
	return None
}

// CompareNewAndOldVolume 检查持久化存储
// 如果新、旧模版中的持久化字段都为申明，则直接返回false，无需删除
// 如果新、旧模版中的持久化字段不一致，则删除旧模版

func CompareNewAndOldVolume(old *appsv1.StatefulSet, new *appsv1.StatefulSet) bool {
	oldv := old.Spec.VolumeClaimTemplates
	newv := new.Spec.VolumeClaimTemplates

	fmt.Printf("oldv=%s newv=%s", oldv, newv)
	if len(oldv) == 0 && len(newv) == 0 {
		return false
	}

	if len(oldv) != len(newv) {
		return true
	}

	//TODO
	if len(oldv) > 0 && len(newv) > 0 {
		oldvolume, _ := json.Marshal(old.Spec.VolumeClaimTemplates[0].Spec.Resources)
		newvolume, _ := json.Marshal(new.Spec.VolumeClaimTemplates[0].Spec.Resources)

		oldScName := old.Spec.VolumeClaimTemplates[0].Spec.StorageClassName
		newScName := new.Spec.VolumeClaimTemplates[0].Spec.StorageClassName
		return !bytes.Equal(oldvolume, newvolume) || !strings.EqualFold(*oldScName, *newScName)
	}
	return false
}
