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
