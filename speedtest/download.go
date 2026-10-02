package speedtest

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DownloadSpeedTest conducts a download speed test on ip and returns download speed and download latency
func DownloadSpeedTest(nBytes int, proxies map[string]string, timeout time.Duration) (float64, float64, error) {
	startTime := time.Now()

	// Create byte slice of nBytes
	data := make([]byte, nBytes)

	// set proxy to request
	var proxy *url.URL
	// set proxies to req header
	for _, v := range proxies {
		proxy, _ = url.Parse(v)
	}

	// v2rayN style cachefly URL for fast reliable testing (no custom bytes needed, we read up to max timeout anyway)
	targetUrl := "https://cachefly.cachefly.net/10mb.test"

	req, err := http.NewRequest("GET", targetUrl, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("error creating request: %v", err)
	}

	// Set up client
	client := &http.Client{
		Timeout:   timeout * time.Second,
		Transport: &http.Transport{Proxy: http.ProxyURL(proxy)},
	}

	// Send request
	resp, reqErr := client.Do(req)
	if reqErr != nil || resp.StatusCode != 200 {
		if resp != nil {
			_ = resp.Body.Close()
		}
		return 0, 0, fmt.Errorf("error sending request or bad status: %v", reqErr)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	// Read up to nBytes
	_, err = io.ReadFull(resp.Body, data)
	// io.ErrUnexpectedEOF or io.EOF might occur if the file is smaller than nBytes (like the 10mb fallback),
	// but we still want to calculate the speed based on what we actually downloaded.
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return 0, 0, fmt.Errorf("error reading response body: %v", err)
	}

	// Calculate download time and speed
	totalTime := time.Since(startTime).Seconds()
	cfTime := float64(0)
	serverTiming := resp.Header.Get("Server-Timing")
	if serverTiming != "" {
		timings := strings.Split(serverTiming, "=")
		if len(timings) > 1 {
			cfTiming, err := strconv.ParseFloat(timings[1], 64)
			if err == nil {
				cfTime = cfTiming / 1000.0
			}
		}
	}
	downloadTime := totalTime - cfTime
	downloadSpeed := float64(nBytes) * 8 / (downloadTime * 1000000)

	return downloadSpeed, downloadTime, nil
}
