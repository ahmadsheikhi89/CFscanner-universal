package vpn

import (
	configuration "CFScanner/configuration"
	"encoding/json"
	"strings"
	"testing"
)

func TestOutboundSecurityNoneOmitsTLSSettings(t *testing.T) {
	cfg := &configuration.Configuration{
		Config: configuration.ConfigStruct{
			Protocol:     "vless",
			Network:      "xhttp",
			Security:     "none",
			Tls:          false,
			WsHeaderHost: "example.com",
			WsHeaderPath: "/",
			AddressPort:  "80",
			UserId:       "00000000-0000-0000-0000-000000000000",
			XhttpMode:    "stream-up",
		},
	}

	outbounds := createOutbound(cfg, "1.1.1.1")
	if len(outbounds) != 1 {
		t.Fatalf("expected 1 outbound, got %d", len(outbounds))
	}

	outbound := outbounds[0]
	if outbound.StreamSettings.Security != "none" {
		t.Errorf("expected streamSettings.security to be 'none', got '%s'", outbound.StreamSettings.Security)
	}

	data, err := json.Marshal(outbound)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	jsonStr := string(data)
	if strings.Contains(jsonStr, "tlsSettings") {
		t.Errorf("expected marshaled JSON to NOT contain 'tlsSettings', but got: %s", jsonStr)
	}

	if outbound.StreamSettings.XHTTPSettings == nil || outbound.StreamSettings.XHTTPSettings.Mode != "stream-up" {
		t.Errorf("expected XHTTPSettings mode to be 'stream-up', got %+v", outbound.StreamSettings.XHTTPSettings)
	}
}

func TestOutboundSecurityTLSIncludesTLSSettings(t *testing.T) {
	cfg := &configuration.Configuration{
		Config: configuration.ConfigStruct{
			Protocol:     "vless",
			Network:      "xhttp",
			Security:     "tls",
			Tls:          true,
			WsHeaderHost: "example.com",
			WsHeaderPath: "/",
			Sni:          "sni.example.com",
			AddressPort:  "443",
			UserId:       "00000000-0000-0000-0000-000000000000",
			XhttpMode:    "auto",
		},
	}

	outbounds := createOutbound(cfg, "1.1.1.1")
	if len(outbounds) != 1 {
		t.Fatalf("expected 1 outbound, got %d", len(outbounds))
	}

	outbound := outbounds[0]
	if outbound.StreamSettings.Security != "tls" {
		t.Errorf("expected streamSettings.security to be 'tls', got '%s'", outbound.StreamSettings.Security)
	}

	data, err := json.Marshal(outbound)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	jsonStr := string(data)
	if !strings.Contains(jsonStr, "tlsSettings") {
		t.Errorf("expected marshaled JSON to contain 'tlsSettings', but got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, "sni.example.com") {
		t.Errorf("expected tlsSettings.serverName to contain 'sni.example.com', got: %s", jsonStr)
	}
}
