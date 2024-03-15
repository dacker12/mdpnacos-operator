package nacos

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
)

// 新版本nacos2.2
type NewNacosNodeInfo struct {
	Code    int         `json:"code"`
	Message interface{} `json:"message"`
	Data    []struct {
		Ip         string `json:"ip"`
		Port       int    `json:"port"`
		State      string `json:"state"`
		ExtendInfo struct {
			LastRefreshTime int64 `json:"lastRefreshTime"`
			RaftMetaData    struct {
				MetaDataMap struct {
					NamingInstanceMetadata struct {
						RaftGroupMember []string `json:"raftGroupMember"`
					} `json:"naming_instance_metadata"`
					NamingPersistentServiceV2 struct {
						RaftGroupMember []string `json:"raftGroupMember"`
					} `json:"naming_persistent_service_v2"`
					NamingServiceMetadata struct {
						RaftGroupMember []string `json:"raftGroupMember"`
					} `json:"naming_service_metadata"`
				} `json:"metaDataMap"`
			} `json:"raftMetaData"`
			RaftPort       string `json:"raftPort"`
			ReadyToUpgrade bool   `json:"readyToUpgrade"`
			Version        string `json:"version"`
		} `json:"extendInfo"`
		Address       string `json:"address"`
		FailAccessCnt int    `json:"failAccessCnt"`
		Abilities     struct {
			RemoteAbility struct {
				SupportRemoteConnection bool `json:"supportRemoteConnection"`
				GrpcReportEnabled       bool `json:"grpcReportEnabled"`
			} `json:"remoteAbility"`
			ConfigAbility struct {
				SupportRemoteMetrics bool `json:"supportRemoteMetrics"`
			} `json:"configAbility"`
			NamingAbility struct {
				SupportJraft bool `json:"supportJraft"`
			} `json:"namingAbility"`
		} `json:"abilities"`
	} `json:"data"`
}

// mdp定制化数据结构
type MdpNacosNodeInfo struct {
	Code    int         `json:"code"`
	Message interface{} `json:"message"`
	Data    []struct {
		Ip            string `json:"ip"`
		Dc            string `json:"dc"`
		Port          int    `json:"port"`
		ModifyPort    int    `json:"modifyPort"`
		SubscribePort int    `json:"subscribePort"`
		State         string `json:"state"`
		ExtendInfo    struct {
			Adweight        string `json:"adWeight"`
			Dc              string `json:"dc"`
			LastRefreshTime int64  `json:"lastRefreshTime"`
			ModifyPort      string `json:"modifyPort"`
			MointorPort     string `json:"mointorPort"`
			Naming          struct {
				Ip             string `json:"ip"`
				HeartbeatDueMs int    `json:"heartbeatDueMs"`
				Term           int    `json:"term"`
				LeaderDueMs    int    `json:"leaderDueMs"`
				State          string `json:"state"`
				VoteFor        string `json:"voteFor,omitempty"`
			} `json:"naming"`
			RaftMetaData struct {
				MetaDataMap struct {
					NacosConfig struct {
						Leader          string   `json:"leader"`
						RaftGroupMember []string `json:"raftGroupMember"`
						Term            int      `json:"term"`
					} `json:"nacos_config"`
				} `json:"metaDataMap"`
			} `json:"raftMetaData"`
			RaftPort      string `json:"raftPort"`
			Site          string `json:"site"`
			SubscribePort string `json:"subscribePort"`
			Version       string `json:"version"`
			Weight        string `json:"weight"`
		} `json:"extendInfo"`
		Address          string `json:"address"`
		FailAccessCnt    int    `json:"failAccessCnt"`
		SuccessAccessCnt int    `json:"successAccessCnt"`
	} `json:"data"`
}

type MdpNacosClient struct {
	HttpClient http.Client
	Log        zap.Logger
}

// GetNacosServerNode TODO
// 内网测试一下cluster.go 代码，完成该部分
func (n *MdpNacosClient) GetNacosServerNode(ip string) (NewNacosNodeInfo, error) {
	nacosServer := NewNacosNodeInfo{}

	//返回[]byte类型的response数据结构
	resp, err := n.HttpClient.Get(fmt.Sprintf("http://%s:8848/nacos/v1/core/cluster/nodes?withInstances=false&pageNo=1&pageSize=10&keyword=&namespaceId=", ip))
	if err != nil {
		n.Log.Error("Failed to request the nacos interface", zap.Any("nacosServerip", ip))
		return nacosServer, err
	}
	//读取响应体中body，返回字节切片类型数据[]byte
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		n.Log.Error("Failed to read the response body resp body", zap.Error(err))
		return nacosServer, err
	}
	defer resp.Body.Close()

	//将[]byte类型的resp.body反序列化为NewNacosNodeInfo数据结构
	err = json.Unmarshal(body, &nacosServer)
	if err != nil {
		n.Log.Error("body unmarshal failed", zap.Error(err), zap.Any("nacosServerIP", ip))
		return nacosServer, err
	}
	return nacosServer, nil
}
