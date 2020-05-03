package httpconfig

import (
	"net"
	"net/http"
	"strconv"
	"time"
)

func NewClient(timeoutSeconds int) *http.Client {
	if timeoutSeconds < 1 {
		panic("Attempt to create a New http.Client with timeouts of: " + strconv.Itoa(timeoutSeconds))
	}
	dialerTimeout := duration(timeoutSeconds)
	dialerKeepAlive := duration(timeoutSeconds)
	tLSHandshakeTimeout := duration(timeoutSeconds)
	endlessRedirectsMaxTime := duration(2 * timeoutSeconds)
	expectContinueTimeout := fractionalDuration(0.4, timeoutSeconds)
	responseHeaderTimeout := fractionalDuration(0.3, timeoutSeconds)

	return &http.Client{
		Timeout: endlessRedirectsMaxTime,
		Transport: &http.Transport{
			TLSHandshakeTimeout:   tLSHandshakeTimeout,
			ExpectContinueTimeout: expectContinueTimeout,
			ResponseHeaderTimeout: responseHeaderTimeout,
			DialContext: (&net.Dialer{
				Timeout:   dialerTimeout,
				KeepAlive: dialerKeepAlive,
			}).DialContext,
		},
	}
}

func duration(timeoutSeconds int) time.Duration {
	return time.Duration(timeoutSeconds) * time.Second
}

func fractionalDuration(fraction float32, timeoutSeconds int) time.Duration {
	float := (fraction * float32(timeoutSeconds)) + 0.9
	return duration(int(float))
}
