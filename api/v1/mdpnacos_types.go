/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	STS_APP_NAME_TAG = "mdp-nacos-operator"
	NACOS_PORT       = 8848
	STS_CPU_REQUEST  = "700m"
	STS_MEM_REQUEST  = "1G"
	STS_CPU_LIMIT    = "700m"
	STS_MEM_LIMIT    = "1G"
	STS_NODESELECTOR = "nacos-deploy"
	NACOS_SVC_NAME   = "xhx-nacos-svc"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MdpnacosSpec defines the desired state of Mdpnacos
type MdpnacosSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Mdpnacos. Edit mdpnacos_types.go to remove/update
	Image           string `json:"image"`
	STSSelector     string `json:"stsSelector"`
	TargetPort      *int32 `json:"targetPort"`
	STSnamespace    string `json:"stsnamespace"`
	STSname         string `json:"stsname"`
	HeadlessSvcName string `json:"headlessSvcName"`
	MinMumPod       *int32 `json:"minMumPod"`
	MaxMumPod       *int32 `json:"maxMumPod"`
	//ScaleCoefficient *ScaleCoefficient                `json:"elasticCoefficient"`
	ScalingPolicy *ScalingPolicy                   `json:"scalingPolicy"`
	PVSpec        corev1.PersistentVolumeSpec      `json:"pvSpec"`
	PVCSpec       corev1.PersistentVolumeClaimSpec `json:"pvcSpec"`
}

// ScaleCoefficient 定义伸缩系数值
type ScaleCoefficient struct {
	Sensitive    string `json:"sensitive,omitempty"`    //灵敏的，阔所容至当前所需的pod数量
	Unresponsive string `json:"unresponsive,omitempty"` //反应迟钝的，每次增加一个pod
}

// ScalingPolicy 定义弹性伸缩策略
type ScalingPolicy struct {
	ScaleCpuPolicy *ScalingCpuPolicy `json:"scalingPolicyCpu,omitempty"`
	ScaleMemPolicy *ScalingMemPolicy `json:"scalingMemPolicy,omitempty"`
	ScaleQpsPolicy *ScaleQpsPolicy   `json:"scaleQpsPolicy,omitempty"`
}

// ScaleQpsPolicy 定义Qps弹性伸缩
type ScaleQpsPolicy struct {
	SingleAppQps *int32 `json:"singleAppQps"`
	AllAppQpss   *int32 `json:"allAppQps"`
}

// ScalingCpuPolicy 定义CPU、MEM弹性伸缩
type ScalingCpuPolicy struct {
	MaxCpuUsage *int32 `json:"maxCpuUsage"`
	MinCpuUsage *int32 `json:"minCpuUsage"`
}

type ScalingMemPolicy struct {
	MaxMemUsage *int32 `json:"maxMemUsage"`
	MinMemUsage *int32 `json:"minMemUsage"`
}

// MdpnacosStatus defines the observed state of Mdpnacos
type MdpnacosStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	AppCurrentQps *int32 `json:"appCurrentQps"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Mdpnacos is the Schema for the mdpnacos API
type Mdpnacos struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MdpnacosSpec   `json:"spec,omitempty"`
	Status MdpnacosStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MdpnacosList contains a list of Mdpnacos
type MdpnacosList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Mdpnacos `json:"items"`
}

// CalculateReplicasNum 计算pod实例数
//func (mdpN *Mdpnacos) CalculateReplicasNum() int32 {
//	//初始化日志框架zap
//	logger, err := zap.NewDevelopment()
//	if err != nil {
//		panic(err)
//	}
//
//	defer func(logger *zap.Logger) {
//		err := logger.Sync()
//		if err != nil {
//			fmt.Println("日志同步失败，请检查")
//		}
//	}(logger)
//
//	//校验MdpNacos yaml中给定的值是否符合规则
//	//nacos集群模式最小副本数不能低于3
//	if *mdpN.Spec.MaxMumPod <= *mdpN.Spec.MinMumPod {
//		logger.Error("MaxMumPod should not be less than or equal to MinMumPod,The minimum number of instances is 3")
//		panic(err)
//	} else if *mdpN.Spec.MinMumPod < 3 {
//		logger.Error("The minimum number of instances is 3")
//		panic(err)
//	}
//
//	//根据最大实例数量与最小实例数量的控制范围确定最终的实例个数
//	replicanum := controller.QpsConversionPodNum(mdpN)
//	if *mdpN.Spec.MinMumPod <= replicanum && replicanum <= *mdpN.Spec.MaxMumPod {
//		logger.Info("The replicas number is")
//		return replicanum
//	} else if replicanum < *mdpN.Spec.MinMumPod {
//		logger.Info("Cannot be lower than the minimum value")
//		return *mdpN.Spec.MinMumPod
//	} else if replicanum > *mdpN.Spec.MaxMumPod {
//		logger.Info("Not higher than the maximum")
//		return *mdpN.Spec.MaxMumPod
//	}
//	return replicanum
//}

func init() {
	SchemeBuilder.Register(&Mdpnacos{}, &MdpnacosList{})
}
