package qvs

import (
	"encoding/json"
	"fmt"
	"log"
	"middle-source-analysis/util"
	"strings"

	"github.com/qbox/pili/common/ipdb.v1"
)

type GbDevice struct {
	Device string `json:"name,omitempty" bson:"name,omitempty"`
	Desc   string `json:"desc,omitempty" bson:"desc,omitempty"`

	// device type 1:camera, 2:platform
	Type       int     `json:"type" bson:"type"`
	DeviceType float32 `json:"deviceType,omitempty"`

	NamespaceId string `json:"nsId" bson:"nsId"`
	Namespace   string `json:"nsName" bson:"nsName,omitempty"`

	GBID string `json:"gbId,omitempty" bson:"gbId,omitempty"`

	// device state: offline: 离线, online: 在线, notReg: 未注册,
	State          string `json:"state" bson:"state"`
	UID            uint32 `json:"-" bson:"uid"`
	Username       string `json:"username" bson:"username"`
	Password       string `json:"password" bson:"password"`
	PullIfRegister bool   `json:"pullIfRegister" bson:"pullIfRegister"` // 注册成功后启动拉流

	CreatedAt int64 `json:"createdAt" bson:"createdAt"`
	UpdatedAt int64 `json:"updatedAt" bson:"updatedAt"`

	Channels         int    `json:"channels" bson:"channels,omitempty"`
	Vendor           string `json:"vendor,omitempty" bson:"vendor,omitempty"`
	NodeId           string `json:"-" bson:"nodeId,omitempty"`
	RemoteIp         string `json:"remoteIp" bson:"remoteIp,omitempty"`
	RtpAccessIp      string `json:"-" bson:"rtpAccessIp,omitempty"`
	AudioRtpAccessIp string `json:"-" bson:"audioRtpAccessIp,omitempty"`
	LastRegisterAt   int64  `json:"lastRegisterAt" bson:"lastRegisterAt,omitempty"`
	LastKeepaliveAt  int64  `json:"lastKeepaliveAt" bson:"-"`
	UserAgent        string `json:"-" bson:"userAgent,omitempty"`
}

func getDevices() {
	if Conf.AdminAk == "" {
		log.Println("err, admin ak empty")
		return
	}
	if Conf.AdminSk == "" {
		log.Println("err, admin sk empty")
		return
	}

	ipParser, err := ipdb.NewCity(Conf.IPDB)
	if err != nil {
		log.Fatalf("[IPDB NewCity] err: %+v\n", err)
		return
	}
	left := 0
	offset := 0
	for {
		Conf.Addr = fmt.Sprintf("https://qvs-admin.qiniuapi.com/v1/devices?line=1000&offset=%d&uid=%s&state=online", offset, Conf.Uid)
		resp, err := util.MikuHttpReq("GET", Conf.Addr, "", Conf.AdminAk, Conf.AdminSk)
		if err != nil {
			log.Println("mikuHttpReq err", err)
			return
		}
		//fmt.Println(resp)
		devices := struct {
			Items []GbDevice `json:"items"`
			Total int        `json:"onlineDeviceCount"`
		}{}
		if err := json.Unmarshal([]byte(resp), &devices); err != nil {
			log.Println(err)
			return
		}
		for _, device := range devices.Items {
			locate, err := ipParser.Find(device.RemoteIp)
			if device.RemoteIp == "" {
				continue
			}
			if err != nil {
				log.Println("IpParser.Find err", err, device.RemoteIp)
				continue
			}
			if strings.Contains(locate.Region, "湖南") && locate.Isp == "移动" {
				fmt.Println(device.GBID, device.RemoteIp, locate.Region, locate.City, locate.Isp)
			}
		}
		if left == 0 {
			left = devices.Total
		} else {
			left -= 1000
			offset += 1000
			log.Println("offset:", offset)
		}
		if left <= 0 {
			break
		}
	}
}
