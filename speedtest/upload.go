package speedtest

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

// UploadSpeedTest conducts a upload speed test on ip and returns upload speed and upload latency
func UploadSpeedTest(nBytes int, proxies map[string]string, timeout time.Duration) (float64, float64, error) {
	startTime := time.Now()
	// Use postman-echo rather than speed.cloudflare.com/__up to prevent EOFs
	// when sending custom payload sizes over Xray/SOCKS5.
	req, err := http.NewRequest("POST", "https://postman-echo.com/post", strings.NewReader(strings.Repeat("0", nBytes)))
	if err != nil {
		return 0, 0, err
	}

	// set proxy to request
	var proxy *url.URL
	// set proxies to req header
	for _, v := range proxies {
		proxy, _ = url.Parse(v)
	}

	client := &http.Client{
		Timeout:   timeout * time.Second,
		Transport: &http.Transport{Proxy: http.ProxyURL(proxy)}}

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	totalTime := time.Since(startTime).Seconds()

	// Since we are no longer using Cloudflare for upload, we just use totalTime as latency.
	latency := totalTime

	var mb float64 = float64(nBytes) * 8 / (1000000.0)
	uploadSpeed := mb / latency

	return uploadSpeed, latency, nil
}
