package speedtest

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// FrontingTest conducts a fronting test on an ip and return true if status 200 is received
func FrontingTest(ip string, proxies map[string]string, timeout time.Duration) bool {

	var success = false

	// set proxy to request
	var proxy *url.URL
	// set proxies to req header
	for _, v := range proxies {
		proxy, _ = url.Parse(v)
	}

	compatibleIP := ip
	if strings.Contains(ip, ":") {
		compatibleIP = fmt.Sprintf("[%s]", ip)
	} else {
		compatibleIP = ip
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s", compatibleIP), nil)
	if err != nil {
		return false
	}
	req.Host = "speed.cloudflare.com"

	client := &http.Client{
		Timeout: timeout * time.Second,
		Transport: &http.Transport{Proxy: http.ProxyURL(proxy),
			TLSClientConfig: &tls.Config{
				ServerName:         "speed.cloudflare.com",
				InsecureSkipVerify: true,
			},
		},
	}

	resp, err := client.Do(req)

	if err != nil {
		// Return false but could add logging here for debugging
		return false
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Errorf("error occured when closing fronting body %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		// Fronting test failed
	} else {
		success = true
	}

	return success
}
