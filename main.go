package vproxy

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

// Get environment variable or default
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getListenAddress() string {
	port := getEnv("PORT", "5555")
	return ":" + port
}

func getProxyAddress() string {
	address := getEnv("PROXY_ADDRESS", "localhost:80")
	return address
}

func logSetup() {
	log.Printf("Proxy running: %s\n", getListenAddress())
	log.Printf("Redirect URL: %s\n", getProxyAddress())
}

type requestPayloadStruct struct {
	ProxyCondition string `json:"proxy_condition"`
}

func main() {
	// Log setup values
	logSetup()

	// Start server
	http.HandleFunc("/", handleRequestAndRedirect)
	if err := http.ListenAndServe(getListenAddress(), nil); err != nil {
		panic(err)
	}
}

// Get a json decoder for a given requests body
func requestBodyDecoder(request *http.Request) *json.Decoder {
	// Read body to buffer
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		panic(err)
	}

	// Because go lang is a pain in the ass if you read the body then any
	// subsequent calls are unable to read the body again.
	request.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	return json.NewDecoder(ioutil.NopCloser())
}

// Parse the requests body
func parseRequestBody(request *http.Request) requestPayloadStruct {
	decoder := requestBodyDecoder(request)
	var requestPayload requestPayloadStruct
	err := decoder.Decode(&requestPayload)

	if err != nil {
		panic(err)
	}

	return requestPayload
}

func serveReverseProxy(target string, res http.ResponseWriter, req *http.Request) {
	// Parse the url
	url, _ := url.Parse(target)

	proxy := httputil.NewSingleHostReverseProxy(url)

	req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = url.Host

	proxy.ServeHTTP(res, req)
}

// Given a request send it to the appropriate url
func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {
	requestPayload := parseRequestBody(req)
	url := getProxyAddress()
	logRequestPayload(requestPayload, url)

	serveReverseProxy(url, res, req)
}

func logRequestPayload(requestPayload requestPayloadStruct, proxyUrl string) {
	log.Printf("proxy_condition: %s, proxy_url: %s\n", requestPayload.ProxyCondition, proxyUrl)
}