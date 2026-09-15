package qvs

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"gopkg.in/yaml.v3"
)

const manscdpContentType = "Application/MANSCDP+xml"

type GBDeviceConfig struct {
	Server            GBServerConfig    `yaml:"server"`
	Device            GBDeviceInfo      `yaml:"device"`
	Local             GBLocalConfig     `yaml:"local"`
	Transport         string            `yaml:"transport"`
	RegisterExpires   int               `yaml:"register_expires"`
	KeepaliveInterval time.Duration     `yaml:"keepalive_interval"`
	Channels          []GBCatalogDevice `yaml:"channels"`
}

type GBServerConfig struct {
	ID      string `yaml:"id"`
	Realm   string `yaml:"realm"`
	Address string `yaml:"address"`
}

type GBDeviceInfo struct {
	ID           string `yaml:"id"`
	Password     string `yaml:"password"`
	Name         string `yaml:"name"`
	Manufacturer string `yaml:"manufacturer"`
	Model        string `yaml:"model"`
	Firmware     string `yaml:"firmware"`
}

type GBLocalConfig struct {
	Address          string `yaml:"address"`
	AdvertiseAddress string `yaml:"advertise_address"`
}

type GBCatalogDevice struct {
	DeviceID     string `yaml:"device_id" xml:"DeviceID"`
	Name         string `yaml:"name" xml:"Name"`
	Manufacturer string `yaml:"manufacturer" xml:"Manufacturer"`
	Model        string `yaml:"model" xml:"Model"`
	Owner        string `yaml:"owner" xml:"Owner"`
	CivilCode    string `yaml:"civil_code" xml:"CivilCode"`
	Address      string `yaml:"address" xml:"Address"`
	Parental     int    `yaml:"parental" xml:"Parental"`
	ParentID     string `yaml:"parent_id" xml:"ParentID"`
	SafetyWay    int    `yaml:"safety_way" xml:"SafetyWay"`
	RegisterWay  int    `yaml:"register_way" xml:"RegisterWay"`
	Secrecy      int    `yaml:"secrecy" xml:"Secrecy"`
	Status       string `yaml:"status" xml:"Status"`
}

type gbCatalogQuery struct {
	CmdType  string `xml:"CmdType"`
	SN       int    `xml:"SN"`
	DeviceID string `xml:"DeviceID"`
}

type gbNotify struct {
	XMLName  xml.Name `xml:"Notify"`
	CmdType  string   `xml:"CmdType"`
	SN       uint32   `xml:"SN"`
	DeviceID string   `xml:"DeviceID"`
	Status   string   `xml:"Status"`
}

type gbCatalogResponse struct {
	XMLName    xml.Name            `xml:"Response"`
	CmdType    string              `xml:"CmdType"`
	SN         int                 `xml:"SN"`
	DeviceID   string              `xml:"DeviceID"`
	SumNum     int                 `xml:"SumNum"`
	DeviceList gbCatalogDeviceList `xml:"DeviceList"`
}

type gbCatalogDeviceList struct {
	Num   int               `xml:"Num,attr"`
	Items []GBCatalogDevice `xml:"Item"`
}

type gbCatalogEvent struct {
	SN int
}

type gbDevice struct {
	config  GBDeviceConfig
	client  *sipgo.Client
	events  chan gbCatalogEvent
	sn      atomic.Uint32
	address string
}

// GBDev starts a GB28181 device using the command-specific YAML configuration.
func GBDev(configFile string) {
	conf, err := loadGBDeviceConfig(configFile)
	if err != nil {
		log.Printf("gbdev: %v", err)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ua, err := sipgo.NewUA(
		sipgo.WithUserAgent(conf.Device.ID),
		sipgo.WithUserAgentHostname(hostOnly(conf.Local.AdvertiseAddress)),
	)
	if err != nil {
		log.Printf("gbdev: create user agent: %v", err)
		return
	}
	defer ua.Close()

	server, err := sipgo.NewServer(ua)
	if err != nil {
		log.Printf("gbdev: create SIP server: %v", err)
		return
	}

	clientOptions := []sipgo.ClientOption{
		sipgo.WithClientAddr(conf.Local.AdvertiseAddress),
	}
	client, err := sipgo.NewClient(ua, clientOptions...)
	if err != nil {
		log.Printf("gbdev: create SIP client: %v", err)
		return
	}
	defer client.Close()

	device := &gbDevice{
		config:  *conf,
		client:  client,
		events:  make(chan gbCatalogEvent, 32),
		address: conf.Server.Address,
	}
	server.OnRequest(sip.MESSAGE, device.handleMessage)

	listener, err := listenGBDevice(conf.Transport, conf.Local.Address)
	if err != nil {
		log.Printf("gbdev: listen on %s/%s: %v", conf.Local.Address, conf.Transport, err)
		return
	}
	defer listener.Close()

	listenErr := make(chan error, 1)
	go func() {
		var err error
		if conf.Transport == "udp" {
			err = server.ServeUDP(listener.(net.PacketConn))
		} else {
			err = server.ServeTCP(listener.(net.Listener))
		}
		listenErr <- err
	}()

	if err := device.register(ctx); err != nil {
		log.Printf("gbdev: register failed: %v", err)
		return
	}
	log.Printf("gbdev: device %s registered to %s over %s", conf.Device.ID, conf.Server.Address, strings.ToUpper(conf.Transport))

	workerErr := make(chan error, 1)
	go func() {
		workerErr <- device.run(ctx)
	}()

	select {
	case <-ctx.Done():
		log.Printf("gbdev: stopped")
	case err := <-listenErr:
		if err != nil && ctx.Err() == nil {
			log.Printf("gbdev: SIP listener stopped: %v", err)
		}
	case err := <-workerErr:
		if err != nil && ctx.Err() == nil {
			log.Printf("gbdev: worker stopped: %v", err)
		}
	}
}

func loadGBDeviceConfig(path string) (*GBDeviceConfig, error) {
	if path == "" {
		return nil, errors.New("-config is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	conf := &GBDeviceConfig{}
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)
	if err := decoder.Decode(conf); err != nil {
		return nil, fmt.Errorf("parse YAML config %q: %w", path, err)
	}
	if err := conf.setDefaultsAndValidate(); err != nil {
		return nil, fmt.Errorf("invalid config %q: %w", path, err)
	}
	return conf, nil
}

func (c *GBDeviceConfig) setDefaultsAndValidate() error {
	c.Transport = strings.ToLower(c.Transport)
	if c.Transport == "" {
		c.Transport = "udp"
	}
	if c.Transport != "udp" && c.Transport != "tcp" {
		return fmt.Errorf("transport must be udp or tcp, got %q", c.Transport)
	}
	if c.Server.ID == "" || c.Server.Address == "" {
		return errors.New("server.id and server.address are required")
	}
	if c.Server.Realm == "" {
		return errors.New("server.realm is required")
	}
	if c.Device.ID == "" || c.Device.Password == "" {
		return errors.New("device.id and device.password are required")
	}
	if c.Local.Address == "" {
		return errors.New("local.address is required")
	}
	for field, address := range map[string]string{
		"server.address": c.Server.Address,
		"local.address":  c.Local.Address,
	} {
		if _, _, err := net.SplitHostPort(address); err != nil {
			return fmt.Errorf("%s must be host:port: %w", field, err)
		}
	}
	if c.Local.AdvertiseAddress == "" {
		c.Local.AdvertiseAddress = c.Local.Address
	}
	if _, _, err := net.SplitHostPort(c.Local.AdvertiseAddress); err != nil {
		return fmt.Errorf("local.advertise_address must be host:port: %w", err)
	}
	if c.RegisterExpires <= 0 {
		c.RegisterExpires = 3600
	}
	if c.KeepaliveInterval <= 0 {
		c.KeepaliveInterval = 60 * time.Second
	}
	if c.Device.Name == "" {
		c.Device.Name = c.Device.ID
	}
	for i := range c.Channels {
		channel := &c.Channels[i]
		if channel.DeviceID == "" {
			return fmt.Errorf("channels[%d].device_id is required", i)
		}
		if channel.Name == "" {
			channel.Name = channel.DeviceID
		}
		if channel.Manufacturer == "" {
			channel.Manufacturer = c.Device.Manufacturer
		}
		if channel.Model == "" {
			channel.Model = c.Device.Model
		}
		if channel.ParentID == "" {
			channel.ParentID = c.Device.ID
		}
		if channel.Status == "" {
			channel.Status = "ON"
		}
	}
	return nil
}

func listenGBDevice(transport, address string) (interface{ Close() error }, error) {
	if transport == "udp" {
		return net.ListenPacket("udp", address)
	}
	return net.Listen("tcp", address)
}

func (d *gbDevice) register(ctx context.Context) error {
	req := d.newRequest(sip.REGISTER, nil)
	req.RemoveHeader("To")
	req.AppendHeader(&sip.ToHeader{
		Address: d.deviceURI(),
		Params:  sip.NewParams(),
	})
	req.AppendHeader(&sip.ContactHeader{
		Address: d.localURI(),
		Params:  sip.NewParams(),
	})
	expires := sip.ExpiresHeader(d.config.RegisterExpires)
	req.AppendHeader(&expires)

	requestCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	res, err := d.client.Do(requestCtx, req)
	if err != nil {
		return err
	}
	if res.StatusCode == sip.StatusUnauthorized || res.StatusCode == sip.StatusProxyAuthRequired {
		res, err = d.client.DoDigestAuth(requestCtx, req, res, sipgo.DigestAuth{
			Username: d.config.Device.ID,
			Password: d.config.Device.Password,
		})
		if err != nil {
			return fmt.Errorf("digest authentication: %w", err)
		}
	}
	if res.StatusCode != sip.StatusOK {
		return fmt.Errorf("unexpected SIP response: %d %s", res.StatusCode, res.Reason)
	}
	return nil
}

func (d *gbDevice) run(ctx context.Context) error {
	keepalive := time.NewTicker(d.config.KeepaliveInterval)
	defer keepalive.Stop()

	registerInterval := time.Duration(d.config.RegisterExpires) * time.Second / 2
	registerTimer := time.NewTimer(registerInterval)
	defer registerTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-keepalive.C:
			if err := d.sendKeepalive(ctx); err != nil {
				log.Printf("gbdev: keepalive failed: %v", err)
			}
		case event := <-d.events:
			if err := d.sendCatalog(ctx, event.SN); err != nil {
				log.Printf("gbdev: Catalog response failed: %v", err)
			}
		case <-registerTimer.C:
			if err := d.register(ctx); err != nil {
				log.Printf("gbdev: registration refresh failed: %v", err)
			} else {
				log.Printf("gbdev: registration refreshed")
			}
			registerTimer.Reset(registerInterval)
		}
	}
}

func (d *gbDevice) handleMessage(req *sip.Request, tx sip.ServerTransaction) {
	if err := tx.Respond(sip.NewResponseFromRequest(req, sip.StatusOK, "OK", nil)); err != nil {
		log.Printf("gbdev: respond to MESSAGE: %v", err)
		return
	}

	var query gbCatalogQuery
	if err := xml.Unmarshal(req.Body(), &query); err != nil {
		log.Printf("gbdev: ignore invalid XML MESSAGE from %s: %v", req.Source(), err)
		return
	}
	if !strings.EqualFold(query.CmdType, "Catalog") {
		log.Printf("gbdev: ignore unsupported MESSAGE CmdType=%q", query.CmdType)
		return
	}
	if query.DeviceID != "" && query.DeviceID != d.config.Device.ID {
		log.Printf("gbdev: ignore Catalog for device %s", query.DeviceID)
		return
	}

	select {
	case d.events <- gbCatalogEvent{SN: query.SN}:
		log.Printf("gbdev: received Catalog query SN=%d", query.SN)
	default:
		log.Printf("gbdev: Catalog queue is full, drop SN=%d", query.SN)
	}
}

func (d *gbDevice) sendKeepalive(ctx context.Context) error {
	body, err := marshalMANSCDP(gbNotify{
		CmdType:  "Keepalive",
		SN:       d.nextSN(),
		DeviceID: d.config.Device.ID,
		Status:   "OK",
	})
	if err != nil {
		return err
	}
	if err := d.sendMessage(ctx, body); err != nil {
		return err
	}
	log.Printf("gbdev: keepalive sent")
	return nil
}

func (d *gbDevice) sendCatalog(ctx context.Context, sn int) error {
	if sn <= 0 {
		sn = int(d.nextSN())
	}
	body, err := marshalMANSCDP(gbCatalogResponse{
		CmdType:  "Catalog",
		SN:       sn,
		DeviceID: d.config.Device.ID,
		SumNum:   len(d.config.Channels),
		DeviceList: gbCatalogDeviceList{
			Num:   len(d.config.Channels),
			Items: d.config.Channels,
		},
	})
	if err != nil {
		return err
	}
	if err := d.sendMessage(ctx, body); err != nil {
		return err
	}
	log.Printf("gbdev: Catalog response sent, SN=%d channels=%d", sn, len(d.config.Channels))
	return nil
}

func (d *gbDevice) sendMessage(ctx context.Context, body []byte) error {
	req := d.newRequest(sip.MESSAGE, body)
	req.AppendHeader(sip.NewHeader("Content-Type", manscdpContentType))

	requestCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	res, err := d.client.Do(requestCtx, req)
	if err != nil {
		return err
	}
	if res.StatusCode != sip.StatusOK {
		return fmt.Errorf("unexpected SIP response: %d %s", res.StatusCode, res.Reason)
	}
	return nil
}

func (d *gbDevice) newRequest(method sip.RequestMethod, body []byte) *sip.Request {
	req := sip.NewRequest(method, d.serverURI())
	req.SetTransport(strings.ToUpper(d.config.Transport))
	req.SetDestination(d.address)
	req.AppendHeader(&sip.FromHeader{
		Address: d.deviceURI(),
		Params:  sip.NewParams().Add("tag", sip.GenerateTagN(16)),
	})
	req.AppendHeader(&sip.ToHeader{
		Address: d.serverURI(),
		Params:  sip.NewParams(),
	})
	req.SetBody(body)
	return req
}

func (d *gbDevice) deviceURI() sip.Uri {
	return sip.Uri{
		Scheme:    "sip",
		User:      d.config.Device.ID,
		Host:      d.config.Server.Realm,
		UriParams: sip.NewParams(),
		Headers:   sip.NewParams(),
	}
}

func (d *gbDevice) serverURI() sip.Uri {
	return sip.Uri{
		Scheme:    "sip",
		User:      d.config.Server.ID,
		Host:      d.config.Server.Realm,
		UriParams: sip.NewParams(),
		Headers:   sip.NewParams(),
	}
}

func (d *gbDevice) localURI() sip.Uri {
	host, portText, _ := net.SplitHostPort(d.config.Local.AdvertiseAddress)
	port, _ := strconv.Atoi(portText)
	params := sip.NewParams()
	params.Add("transport", d.config.Transport)
	return sip.Uri{
		Scheme:    "sip",
		User:      d.config.Device.ID,
		Host:      host,
		Port:      port,
		UriParams: params,
		Headers:   sip.NewParams(),
	}
}

func (d *gbDevice) nextSN() uint32 {
	return d.sn.Add(1)
}

func marshalMANSCDP(value any) ([]byte, error) {
	body, err := xml.Marshal(value)
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), body...), nil
}

func hostOnly(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return address
	}
	return host
}
