package qvs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadGBDeviceConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gbdev.yaml")
	data := []byte(`
server:
  id: "34020000002000000001"
  realm: "3402000000"
  address: "127.0.0.1:5060"
device:
  id: "34020000001320000001"
  password: "12345678"
local:
  address: "0.0.0.0:5061"
  advertise_address: "127.0.0.1:5061"
channels:
  - device_id: "34020000001320000002"
`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	conf, err := loadGBDeviceConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if conf.Transport != "udp" {
		t.Fatalf("transport = %q, want udp", conf.Transport)
	}
	if conf.RegisterExpires != 3600 {
		t.Fatalf("register_expires = %d, want 3600", conf.RegisterExpires)
	}
	if conf.KeepaliveInterval != time.Minute {
		t.Fatalf("keepalive_interval = %s, want 1m", conf.KeepaliveInterval)
	}
	if conf.Channels[0].Status != "ON" {
		t.Fatalf("channel status = %q, want ON", conf.Channels[0].Status)
	}
	if conf.Channels[0].ParentID != conf.Device.ID {
		t.Fatalf("channel parent_id = %q, want %q", conf.Channels[0].ParentID, conf.Device.ID)
	}
}

func TestMarshalCatalogResponse(t *testing.T) {
	body, err := marshalMANSCDP(gbCatalogResponse{
		CmdType:  "Catalog",
		SN:       7,
		DeviceID: "34020000001320000001",
		SumNum:   1,
		DeviceList: gbCatalogDeviceList{
			Num: 1,
			Items: []GBCatalogDevice{{
				DeviceID: "34020000001320000002",
				Name:     "Camera 1",
				Status:   "ON",
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	xmlBody := string(body)
	for _, want := range []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<CmdType>Catalog</CmdType>`,
		`<SN>7</SN>`,
		`<DeviceList Num="1">`,
		`<DeviceID>34020000001320000002</DeviceID>`,
		`<Status>ON</Status>`,
	} {
		if !strings.Contains(xmlBody, want) {
			t.Errorf("Catalog XML does not contain %q: %s", want, xmlBody)
		}
	}
}
