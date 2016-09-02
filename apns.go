package apns

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"golang.org/x/net/http2"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
)

const (
	// APNS Server host
	TestHost        = "https://localhost:2197"
	DevelopmentHost = "https://api.development.push.apple.com:2197"
	ProductionHost  = "https://api.push.apple.com:2197"

	// Apns url path format
	ApnsUrlFormat = "%s/3/device/%s"

	// Payload dictionary keys
	Aps = "aps"

	// Aps dictionary keys
	Alert            = "alert"
	Badge            = "badge"
	Sound            = "sound"
	ContentAvailable = "content-available"

	// Default sound value
	Default = "default"

	// Alert dictionary keys
	Title = "title"
	Body  = "body"

	// APNS Request header keys
	ApnsID         = "apns-id"
	ApnsExpiration = "apns-expiration"

	CaChainCert   = "/data/sec/ca/intermediate/certs/ca-chain.cert.pem"
	CertFromCA    = "/data/sec/ca/intermediate/certs/localhost.client.cert.pem"
	KeyFromCA     = "/data/sec/ca/intermediate/private/localhost.client.key.pem"
	CertFromSelf  = "/data/sec/self/certs/cert.pem"
	KeyFromSelf   = "/data/sec/self/private/key.pem"
	CertFromApple = "/data/sec/apple/certs/cert.pem"
	KeyFromApple  = "/data/sec/apple/private/key.pem"
)

func handleErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

type Payload struct {
	Alert             string `json:"alert"`
	Title             string `json:"title,omitempty"`
	Badge             int    `json:"badge,omitempty"`
	Sound             string `json:"sound,omitempty"`
	Content_available bool   `json:"content-available,omitempty"`
}

type Headers struct {
	Apns_id         string `json:"apns-id"`
	Apns_expiration string `json:"apns-expiration"`
}

type Notification struct {
	Payload Payload `json:"payload"`
	Headers Headers `json:"headers"`
}

type payload Payload
type headers Headers

func (payload *Payload) toBytes() []byte {
	aps := map[string]interface{}{}
	if len(payload.Title) > 0 {
		alert := map[string]string{}
		alert[Title] = payload.Title
		alert[Body] = payload.Alert
		aps[Alert] = alert
	} else {
		aps[Alert] = payload.Alert
	}
	aps[Badge] = payload.Badge
	aps[Sound] = payload.Sound
	if payload.Content_available {
		aps[ContentAvailable] = 1
	}
	var bytesBuffer bytes.Buffer
	apsPayload := map[string]interface{}{Aps: aps}
	err := json.NewEncoder(&bytesBuffer).Encode(apsPayload)
	handleErr(err)
	return bytesBuffer.Bytes()
}

type Server struct {
	host   string
	client *http.Client
}

func NewServer(host, cert, key, rootCA string) *Server {
	s := new(Server)
	s.host = host
	var tlsCertificate, err = tls.LoadX509KeyPair(cert, key)
	handleErr(err)

	tlsConfig := new(tls.Config)
	tlsConfig.Certificates = []tls.Certificate{tlsCertificate}
	tlsConfig.InsecureSkipVerify = false // false ensures that the server cert is verified.
	if len(rootCA) > 0 {
		caCert, err := ioutil.ReadFile(rootCA)
		handleErr(err)
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool // providing cert pool ensures that the TLS cert verification passes.
	}

	if len(tlsCertificate.Certificate) > 0 {
		tlsConfig.BuildNameToCertificate()
		fmt.Println(tlsConfig.NameToCertificate)
	}
	s.client = &http.Client{
		Transport: &http2.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
	return s
}

func sendAPSPushToDevicesUsingClient(n Notification, s *Server, wg *sync.WaitGroup, deviceToken string) {
	defer wg.Done()
	// api.development.push.apple.com
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf(ApnsUrlFormat, s.host, deviceToken), bytes.NewReader(n.Payload.toBytes()))
	handleErr(err)
	req.Header.Set(ApnsID, n.Headers.Apns_id)
	req.Header.Set(ApnsExpiration, n.Headers.Apns_expiration)
	resp, err := s.client.Do(req)
	handleErr(err)
	if resp != nil {
		defer resp.Body.Close() // close response body to reuse connection
		fmt.Println(resp)
		// body, err := ioutil.ReadAll(resp.Body)
		io.Copy(ioutil.Discard, resp.Body) // read entire body to reuse connection
	} else {
		fmt.Println("No response from server")
	}
}

func SendPush(n Notification, s *Server, d []string) error {
	fmt.Println(n)
	if len(n.Headers.Apns_id) < 36 {
		return fmt.Errorf("apns-id must be a valid uuid")
	}
	if len(n.Headers.Apns_expiration) < 1 {
		return fmt.Errorf("apns-expiration must be a valid time interval")
	}
	var wg sync.WaitGroup
	for i := 0; i < len(d); i++ {
		wg.Add(1)
		go sendAPSPushToDevicesUsingClient(n, s, &wg, d[i])
	}
	wg.Wait()
	return nil
}
