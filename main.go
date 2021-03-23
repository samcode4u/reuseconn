package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

var connections []*resty.Client
var rrindex = 0
var maxsize = 10
var mutext sync.Mutex

func CreateConnections() {
	mutext.Lock()
	for i := 0; i < maxsize; i++ {
		fmt.Println(i)
		connection := resty.New()
		connection.SetBasicAuth("username", "password")
		connections = append(connections, connection)
	}
	mutext.Unlock()
}

func GetConnection() *resty.Client {
	mutext.Lock()
	rrindex = rrindex % maxsize
	fmt.Println("rrindex = ", rrindex)
	conn := connections[rrindex]
	rrindex++
	mutext.Unlock()
	return conn
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

var netClient *http.Client

func httpClient() *http.Client {
	if netClient == nil {
		t := http.DefaultTransport.(*http.Transport).Clone()

		t.MaxIdleConns = 100
		t.MaxConnsPerHost = 100
		t.MaxIdleConnsPerHost = 100
		t.IdleConnTimeout = 90 * time.Second

		netClient = &http.Client{Transport: t}
	}
	return netClient
}

func sendRequest(client *http.Client, method string, endpoint string, reqbody []byte, username string, password string) []byte {

	req, err := http.NewRequest(method, endpoint, bytes.NewBuffer(reqbody))
	if err != nil {
		log.Fatalf("Error Occured. %+v", err)
	}

	req.Header.Add("Authorization", "Basic "+basicAuth(username, password))

	response, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error sending request to API endpoint. %+v", err)
	}

	// Close the connection to reuse it
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("Couldn't parse response body. %+v", err)
	}

	return body
}

func TestTranportConcurrency() {
	resp := sendRequest(httpClient(), "GET", "https://api.tiniyo.com/v1/Accounts/username/Messages/fcbddeab-5ac3-4281-9dd7-e857ef6625ba", nil, "username", "password")
	fmt.Println(string(resp))
}

func CallAPI(connection *resty.Client) {
	resp, err := connection.R().
		EnableTrace().
		Get("https://api.tiniyo.com/v1/Accounts/username/Messages/fcbddeab-5ac3-4281-9dd7-e857ef6625ba")

	// Explore response object
	// fmt.Println("Response Info:")
	fmt.Println("  Error      :", err)
	// fmt.Println("  Status Code:", resp.StatusCode())
	// fmt.Println("  Status     :", resp.Status())
	// fmt.Println("  Proto      :", resp.Proto())
	// fmt.Println("  Time       :", resp.Time())
	// fmt.Println("  Received At:", resp.ReceivedAt())
	// fmt.Println("  Body       :\n", resp)
	// fmt.Println()

	// Explore trace info
	fmt.Println("Request Trace Info:")
	ti := resp.Request.TraceInfo()
	// fmt.Println("  DNSLookup     :", ti.DNSLookup)
	// fmt.Println("  ConnTime      :", ti.ConnTime)
	// fmt.Println("  TCPConnTime   :", ti.TCPConnTime)
	// fmt.Println("  TLSHandshake  :", ti.TLSHandshake)
	// fmt.Println("  ServerTime    :", ti.ServerTime)
	// fmt.Println("  ResponseTime  :", ti.ResponseTime)
	fmt.Println("  TotalTime     :", ti.TotalTime)
	fmt.Println("  IsConnReused  :", ti.IsConnReused)
	fmt.Println("  IsConnWasIdle :", ti.IsConnWasIdle)
	// fmt.Println("  ConnIdleTime  :", ti.ConnIdleTime)
	// fmt.Println("  RequestAttempt:", ti.RequestAttempt)
	// fmt.Println("  RemoteAddr    :", ti.RemoteAddr.String())

}

func createTransport(localAddr net.Addr) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}
	if localAddr != nil {
		dialer.LocalAddr = localAddr
	}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   100,
		MaxConnsPerHost:       100,
	}
}

func RoutineConfigTest() {
	connection := resty.New()
	tranport := createTransport(nil)
	connection.SetTransport(tranport)
	for i := 0; i < 100; i++ {
		fmt.Println("times", i)
		go CallAPI(connection)
	}
	time.Sleep(5 * time.Second)
	for i := 0; i < 100; i++ {
		fmt.Println("times", i)
		go CallAPI(connection)
	}
}

func RoutineTest() {
	resp, err := GetConnection().R().
		EnableTrace().
		Get("https://api.tiniyo.com/v1/Accounts/username/Messages/fcbddeab-5ac3-4281-9dd7-e857ef6625ba")

	// Explore response object
	// fmt.Println("Response Info:")
	fmt.Println("  Error      :", err)
	// fmt.Println("  Status Code:", resp.StatusCode())
	// fmt.Println("  Status     :", resp.Status())
	// fmt.Println("  Proto      :", resp.Proto())
	// fmt.Println("  Time       :", resp.Time())
	// fmt.Println("  Received At:", resp.ReceivedAt())
	// fmt.Println("  Body       :\n", resp)
	// fmt.Println()

	// Explore trace info
	fmt.Println("Request Trace Info:")
	ti := resp.Request.TraceInfo()
	// fmt.Println("  DNSLookup     :", ti.DNSLookup)
	// fmt.Println("  ConnTime      :", ti.ConnTime)
	// fmt.Println("  TCPConnTime   :", ti.TCPConnTime)
	// fmt.Println("  TLSHandshake  :", ti.TLSHandshake)
	// fmt.Println("  ServerTime    :", ti.ServerTime)
	// fmt.Println("  ResponseTime  :", ti.ResponseTime)
	fmt.Println("  TotalTime     :", ti.TotalTime)
	fmt.Println("  IsConnReused  :", ti.IsConnReused)
	fmt.Println("  IsConnWasIdle :", ti.IsConnWasIdle)
	// fmt.Println("  ConnIdleTime  :", ti.ConnIdleTime)
	// fmt.Println("  RequestAttempt:", ti.RequestAttempt)
	// fmt.Println("  RemoteAddr    :", ti.RemoteAddr.String())

}

func startWebserver() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		time.Sleep(time.Millisecond * 50)

		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	go http.ListenAndServe(":8080", nil)

}

func startLoadTest() {
	count := 0
	for {
		resp, err := http.Get("http://localhost:8080/")
		if err != nil {
			panic(fmt.Sprintf("Got error: %v", err))
		}
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
		log.Printf("Finished GET request #%v", count)
		count += 1
	}

}

func startLoadReuseTest(client *http.Client) {
	count := 0
	for {
		sendRequest(client, "GET", "http://localhost:8080/", nil, "username", "password")
		log.Printf("Finished GET request #%v", count)
		count += 1
		if count > 10 {
			break
		}
	}
}

func startLoadReuseRestyTest(connection *resty.Client) {
	count := 0
	for {
		resp, _ := connection.R().
			EnableTrace().
			Get("http://localhost:8080/")
		log.Printf("Finished GET request #%v", count)
		ti := resp.Request.TraceInfo()
		fmt.Println(ti.IsConnReused)
		count += 1
		if count > 5 {
			break
		}
	}
}

func main() {
	connection := resty.New()
	tranport := createTransport(nil)
	connection.SetTransport(tranport)

	// start a webserver in a goroutine
	startWebserver()

	time.Sleep(time.Second * 5)

	for i := 0; i < 10; i++ {
		// go startLoadTest()
		// go startLoadReuseTest(httpClient())
		go startLoadReuseRestyTest(connection)
	}
	time.Sleep(time.Second * 2)
	fmt.Println("=======================================================")
	for i := 0; i < 10; i++ {
		// go startLoadTest()
		// go startLoadReuseTest(httpClient())
		go startLoadReuseRestyTest(connection)
	}

	time.Sleep(time.Second * 2400)

}
