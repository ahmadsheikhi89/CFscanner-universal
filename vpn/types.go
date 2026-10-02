package vpn

import (
	"github.com/xtls/xray-core/core"
)

type ScanWorker struct {
	Instance *core.Instance
}
type Log struct {
	Loglevel string `json:"loglevel"`
}

type Inbound struct {
	Port     int    `json:"port"`
	Listen   string `json:"listen"`
	Tag      string `json:"tag"`
	Protocol string `json:"protocol"`
	Settings struct {
		Auth string `json:"auth"`
		UDP  bool   `json:"udp"`
		IP   string `json:"ip"`
	} `json:"settings"`
	Sniffing struct {
		Enabled      bool     `json:"enabled"`
		DestOverride []string `json:"destOverride"`
	} `json:"sniffing"`
}

type User struct {
	ID         string `json:"id"`
	Encryption string `json:"encryption,omitempty"` // "none" for VLESS
	Security   string `json:"security,omitempty"`   // "auto" for VMESS
}

type VNext struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
	Users   []User `json:"users"`
}

type WSSettings struct {
	Host    string            `json:"host,omitempty"`
	Path    string            `json:"path,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type XHTTPSettings struct {
	Host        string `json:"host,omitempty"`
	Path        string `json:"path,omitempty"`
	Mode        string `json:"mode,omitempty"` // auto, packet-up, stream-up, stream-one
	NoSSEHeader bool   `json:"noSSEHeader,omitempty"`
}

type HTTPUpgradeSettings struct {
	Host    string            `json:"host,omitempty"`
	Path    string            `json:"path,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type GRPCSettings struct {
	ServiceName string `json:"serviceName,omitempty"`
	MultiMode   bool   `json:"multiMode,omitempty"`
}

type RealitySettings struct {
	Show        bool   `json:"show,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"` // chrome, firefox, safari, etc.
	ServerName  string `json:"serverName,omitempty"`
	PublicKey   string `json:"publicKey,omitempty"`
	ShortId     string `json:"shortId,omitempty"`
	SpiderX     string `json:"spiderX,omitempty"`
}

type TLSSettings struct {
	ServerName    string   `json:"serverName"`
	AllowInsecure bool     `json:"allowInsecure"`
	ALPN          []string `json:"alpn,omitempty"`
	Fingerprint   string   `json:"fingerprint,omitempty"` // chrome, firefox, safari, ios, edge, 360, qq, random, randomized
}

type StreamSettings struct {
	Network             string               `json:"network"`
	Security            string               `json:"security"`
	WSSettings          *WSSettings          `json:"wsSettings,omitempty"`
	XHTTPSettings       *XHTTPSettings       `json:"xhttpSettings,omitempty"`
	HTTPUpgradeSettings *HTTPUpgradeSettings `json:"httpupgradeSettings,omitempty"`
	TLSSettings         *TLSSettings         `json:"tlsSettings,omitempty"`
	RealitySettings     *RealitySettings     `json:"realitySettings,omitempty"`
	GRPCSettings        *GRPCSettings        `json:"grpcSettings,omitempty"`
}

type TrojanServer struct {
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Password string `json:"password"`
}

type OutboundSettings struct {
	VNext   []VNext        `json:"vnext,omitempty"`
	Servers []TrojanServer `json:"servers,omitempty"`
}

type Outbound struct {
	Protocol       string           `json:"protocol"`
	Settings       OutboundSettings `json:"settings"`
	StreamSettings StreamSettings   `json:"streamSettings"`
}

type XRay struct {
	Log       Log        `json:"log"`
	Inbounds  []Inbound  `json:"inbounds"`
	Outbounds []Outbound `json:"outbounds"`
	Other     struct{}   `json:"other"`
}
