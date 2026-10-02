package speedtest

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"time"
)

// ConnectivityTest conducts a quick GET request to testUrl to check simple connectivity.
// Modeled after v2rayN's GetRealPingTime: reuses a single HTTP client across retries
// for TLS connection reuse, and does not follow redirects to save round-trips.
func ConnectivityTest(testUrl string, proxies map[string]string, timeout time.Duration) bool {
	var proxy *url.URL
	for _, v := range proxies {
		proxy, _ = url.Parse(v)
	}

	// Single transport reused across attempts — the 2nd attempt reuses the
	// established TLS session through the SOCKS5 proxy, just like v2rayN's
	// single HttpClient with SocketsHttpHandler.
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxy),
		DialContext: (&net.Dialer{
			Timeout: timeout * time.Second, // Fast TCP dial timeout
		}).DialContext,
		TLSHandshakeTimeout: timeout * time.Second,
		DisableKeepAlives:   false, // KEEP connections alive for reuse between attempts
	}

	client := &http.Client{
		Transport: transport,
		// Don't follow redirects — saves a full extra SOCKS5→TLS round-trip.
		// http://google.com/generate_204 redirects to https, wasting time.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	defer transport.CloseIdleConnections()

	// v2rayN does 2 attempts with 100ms gap, takes the fastest.
	for i := 0; i < 2; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
		req, err := http.NewRequestWithContext(ctx, "GET", testUrl, nil)
		if err != nil {
			cancel()
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			cancel()
			if i < 1 {
				time.Sleep(100 * time.Millisecond)
			}
			continue
		}

		status := resp.StatusCode
		_ = resp.Body.Close()
		cancel()

		// Accept 2xx, 3xx (redirects count as success too — connection works)
		if status >= 200 && status < 400 {
			return true
		}

		time.Sleep(100 * time.Millisecond)
	}

	return false
}
