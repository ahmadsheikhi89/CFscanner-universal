package vpn

import (
	configuration "CFScanner/configuration"
	"CFScanner/utils"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"


)

var logger = []string{"debug",
	"info",
	"warning",
	"error",
	"none"}

func loglevel(level string) string {
	for _, log := range logger {
		if strings.ToLower(level) == log {
			return level
		}
	}
	return logger[4]
}

func createInbound() []Inbound {
	localPortStr := utils.GetFreePort()

	config := Inbound{
		Port:     localPortStr,
		Listen:   "127.0.0.1",
		Tag:      "socks-inbound",
		Protocol: "socks",
		Settings: struct {
			Auth string `json:"auth"`
			UDP  bool   `json:"udp"`
			IP   string `json:"ip"`
		}{"noauth", false, "127.0.0.1"},
		Sniffing: struct {
			Enabled      bool     `json:"enabled"`
			DestOverride []string `json:"destOverride"`
		}{true, []string{"http", "tls"}},
	}
	configSlice := []Inbound{config}
	return configSlice
}

func createOutbound(C *configuration.Configuration, IP string) []Outbound {
	vnextIP, _ := strconv.Atoi(C.Config.AddressPort)
	if C.Config.Sni == "" {
		C.Config.Sni = C.Config.WsHeaderHost
	}

	protocol := C.Config.Protocol
	if protocol == "" {
		protocol = "vless"
	}

	var outboundSettings OutboundSettings

	switch protocol {
	case "vmess":
		outboundSettings = OutboundSettings{
			VNext: []VNext{
				{
					Address: IP,
					Port:    vnextIP,
					Users: []User{
						{
							ID:       C.Config.UserId,
							Security: "auto",
						},
					},
				},
			},
		}
	case "trojan":
		outboundSettings = OutboundSettings{
			Servers: []TrojanServer{
				{
					Address:  IP,
					Port:     vnextIP,
					Password: C.Config.UserId,
				},
			},
		}
	case "vless":
		fallthrough
	default:
		protocol = "vless"
		outboundSettings = OutboundSettings{
			VNext: []VNext{
				{
					Address: IP,
					Port:    vnextIP,
					Users: []User{
						{
							ID:         C.Config.UserId,
							Encryption: "none",
						},
					},
				},
			},
		}
	}

	config := Outbound{
		Protocol: protocol,
		Settings: outboundSettings,
	}

	security := C.Config.Security
	if security == "" {
		if C.Config.Tls {
			security = "tls"
		} else {
			security = "none"
		}
	}

	streamSettings := StreamSettings{
		Network:  C.Config.Network,
		Security: security,
	}

	if security == "tls" {
		serverName := C.Config.Sni
		if C.Config.RandomizeSni {
			serverName = fmt.Sprintf("%d.%s", time.Now().UnixNano(), C.Config.Sni)
		}
		tlsSettings := &TLSSettings{
			ServerName:    serverName,
			AllowInsecure: false,
			Fingerprint:   "chrome",
		}
		if C.Config.Network == "grpc" {
			tlsSettings.ALPN = []string{"h2"}
		}
		streamSettings.TLSSettings = tlsSettings
	} else {
		streamSettings.TLSSettings = nil
	}

	xhttpMode := C.Config.XhttpMode
	if xhttpMode == "" {
		xhttpMode = "auto"
	}

	switch C.Config.Network {
	case "xhttp":
		streamSettings.XHTTPSettings = &XHTTPSettings{
			Host: C.Config.WsHeaderHost,
			Path: C.Config.WsHeaderPath,
			Mode: xhttpMode,
		}
	case "ws":
		streamSettings.WSSettings = &WSSettings{
			Host: C.Config.WsHeaderHost,
			Path: C.Config.WsHeaderPath,
			Headers: map[string]string{
				"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.131 Safari/537.36",
			},
		}
	case "grpc":
		streamSettings.GRPCSettings = &GRPCSettings{
			ServiceName: strings.TrimPrefix(C.Config.WsHeaderPath, "/"),
			MultiMode:   false,
		}
	default:
		// Default to xhttp if unknown
		streamSettings.Network = "xhttp"
		streamSettings.XHTTPSettings = &XHTTPSettings{
			Host: C.Config.WsHeaderHost,
			Path: C.Config.WsHeaderPath,
			Mode: xhttpMode,
		}
	}

	config.StreamSettings = streamSettings
	configSlice := []Outbound{config}
	return configSlice
}

func XRayConfig(IP string, testConfig *configuration.Configuration) string {
	// Create config in memory for this specific IP
	config := XRay{
		Log: Log{
			Loglevel: loglevel(testConfig.LogLevel),
		},
		Inbounds:  createInbound(),
		Outbounds: createOutbound(testConfig, IP),
		Other:     struct{}{},
	}

	configByte, err := json.Marshal(config)
	if err != nil {
		log.Fatal("Marshal error", err)
	}

	// Create a unique config file to avoid race conditions
	// Use timestamp with nanoseconds for uniqueness
	timestamp := time.Now().UnixNano()
	configPath := fmt.Sprintf("%s/xray-%d.json", configuration.DIR, timestamp)

	err = writeJSONToFile(configByte, configPath)
	if err != nil {
		log.Fatal("Failed to write JSON to file", err)
		return ""
	}

	return configPath
}

func writeJSONToFile(jsonBytes []byte, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(jsonBytes)
	if err != nil {
		return err
	}

	return nil
}
