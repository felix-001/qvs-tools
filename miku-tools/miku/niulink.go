package miku

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/qbox/mikud-live/common/auth/qiniumac.v1"
	"github.com/rs/zerolog/log"
)

type NiuLinkData struct {
	CostLevels []CostLevel `json:"costLevels"`
}

type IdcDedail struct {
	IdcId              string  `json:"idcId"`
	Bandwidth          float64 `json:"bandwidth"`
	NodeTotalBandwidth float64 `json:"nodeTotalBandwidth"`
}

type CostLevel struct {
	Bandwidth        float64   `json:"bandwidth"`
	BillingType      string    `json:"billingType"`
	City             string    `json:"city"`
	CostLevel        float64   `json:"costLevel"`
	CostLevelV2      float64   `json:"costLevelV2"`
	CustomerIds      []uint32  `json:"customerIds,omitempty"`
	GuaranteedRate   float64   `json:"guaranteedRate"`
	IdcDetail        IdcDedail `json:"idcDetail"`
	IsBanTransProv   bool      `json:"isBanTransProv"`
	Isp              string    `json:"isp"` // 真实运营商（原运营商）
	NodeID           string    `json:"nodeId"`
	OriginNodeID     string    `json:"originNodeId"`
	Province         string    `json:"province"`
	NatType          string    `json:"natType"`
	ResourceType     string    `json:"resourceType"`
	SchedulePriority int       `json:"schedulePriority"`
	VendorID         int       `json:"vendorId"`

	//Schedules    []commonModel.Schedule    `json:"schedules"`
	//ScheduleIsps []commonModel.ScheduleIsp `json:"scheduleIsps"` // 可调度的运营商信息
}

func httpReq(method, addr, body string, client *http.Client, headers map[string]string) (*NiuLinkData, error) {
	req, _ := http.NewRequest(method, addr, bytes.NewBuffer([]byte(body)))
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	var (
		resp *http.Response
		err  error
	)
	retry := 3
	for i := 0; i < retry; i++ {
		resp, err = client.Do(req)
		if err != nil {
			log.Info().Msgf("httpReq get niulink info fail, retry: %d", i+1)
		} else {
			log.Info().Msgf("httpReq get niulink info success, retry: %d", i+1)
			break
		}
	}
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	resp_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New("http status code err")
	}
	niuLinkData := &NiuLinkData{}
	err = json.Unmarshal(resp_body, niuLinkData)
	return niuLinkData, err
}

func (m *Miku) niuLinkHttpReq(page, size int) (*NiuLinkData, error) {
	niuLinkCli := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
			DisableKeepAlives: true,
		},
		Timeout: time.Second * time.Duration(600),
	}
	addr := fmt.Sprintf("http://%s%s?page=%d&size=%d", m.conf.Domain, "/billing/v2/vendor/nodes/costlevel", page, size)
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	host := u.Host
	u.Host = ""
	u.Scheme = ""
	headers := map[string]string{}
	headers["Content-Type"] = "application/json"
	token := qiniumac.SignTokenWithParam(m.conf.Ak, m.conf.Sk, http.MethodGet, u.String(), host, "", headers)
	headers["Authorization"] = token
	return httpReq(http.MethodGet, addr, "", niuLinkCli, headers)
}

func (m *Miku) Niulink(config *Config) {
	tmp := make(map[string]CostLevel)
	for i := 1; i <= 6; i++ {
		nilLinkData, err := m.niuLinkHttpReq(i, 1000)
		if err != nil {
			logger.Error().Msgf("get niulink info, err: %s", err.Error())
			continue
		}
		for _, value := range nilLinkData.CostLevels {
			tmp[value.OriginNodeID] = value
		}
	}
	bytes, err := json.MarshalIndent(tmp, "", "  ")
	if err != nil {
		logger.Error().Err(err).Msg("niulink json marshal failed")
		return
	}
	if err := os.WriteFile("/tmp/niulink.json", bytes, 0644); err != nil {
		logger.Error().Err(err).Msg("write niulink json file failed")
		return
	}
	logger.Info().Msg("niulink data saved to /tmp/niulink.json")
}
