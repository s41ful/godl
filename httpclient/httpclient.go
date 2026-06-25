package httpclient

import (
    "fmt"
    "io"
    "net/http"
    "net/http/httputil"
    "time"
)

var tr = &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 20,
    MaxConnsPerHost:     20,
    IdleConnTimeout:     30 * time.Second,
}

type RetryTransport struct {
    Base       http.RoundTripper
    MaxRetries int
    Delay      time.Duration
}

func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    var resp *http.Response
    var err error

    for i := 0; i <= t.MaxRetries; i++ {
	resp, err = t.Base.RoundTrip(req)

	if err == nil && resp.StatusCode < 500 {
	    return resp, nil
	}

	fmt.Printf("Request Failed retrying: %d\n", i)
	if resp != nil && resp.Body != nil {
	    resp.Body.Close()
	}

	// last retry → return error
	if i == t.MaxRetries {
	    break
	}

	time.Sleep(t.Delay)
    }

    return resp, err
}

func DumpRequest(req *http.Request, withBody bool) string {
    dump, err := httputil.DumpRequestOut(req, withBody)
    if err != nil {
	return "dump request error:" + err.Error()
    }
    return string(dump)
}

func DumpResponseHeader(resp *http.Response) string {
    dump, err := httputil.DumpResponse(resp, false)
    if err != nil {
	return "dump response error: " + err.Error() 
    }

    return string(dump)
}

func NewRequest(method, url string, body io.Reader) (*http.Request, error){
    return http.NewRequest(method, url, body)
}

func NewDefaultWebRequest(url string) (*http.Request, error) { 
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
	return nil, err
    }

    req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36")
    req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
    req.Header.Set("Accept-Language", "en-us,en;q=0.5")
    return req, nil
}


type LogTransport struct {
    Base 				http.RoundTripper
    MaxRetries			int
    Delay				time.Duration
}

func (l *LogTransport)RoundTrip(req *http.Request) (*http.Response, error){
    var resp *http.Response
    var err error
    fmt.Printf("SENDING REQUEST:\n%s\n", DumpRequest(req, req.Body != nil))

    for i := 0; i <= l.MaxRetries; i++ {
	resp, err = l.Base.RoundTrip(req)

	if err == nil && resp.StatusCode < 500 {
	    fmt.Printf("RECEIVING HEADERS:\n%s\n", DumpResponseHeader(resp))
	    return resp, nil
	}

	fmt.Printf("Request Failed retrying: %d\n", i)
	if resp != nil && resp.Body != nil {
	    resp.Body.Close()
	}

	// last retry → return error
	if i == l.MaxRetries {
	    break
	}

	time.Sleep(l.Delay)
    }

    if err != nil {
	fmt.Printf("httpclient: %s\n", err)
	return nil, err
    }

    return resp, err
}


func NewClient(debugTraffic bool, maxRetries int) *http.Client {
    if debugTraffic {
	return &http.Client{

	    Transport: &LogTransport{
		Base: 			tr,
		MaxRetries: 	maxRetries,
		Delay: 			1 * time.Second,
	    },
	}
    } else {
	return &http.Client{
	    Transport: &RetryTransport{
		Base: 			  tr,
		MaxRetries: 	maxRetries,
		Delay: 			  1 * time.Second,
	    },
	}
    }
}

