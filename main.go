package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	timestamp_format = "2006-01-02 15:04:05"
	filename_format = "2006-01-02_15:04:05"
	logs_base_path = "logs/"
)

var session_filename = ""

func logEvent(text string) {
	current_time := time.Now()
	timestamp_part := fmt.Sprintf("%s :: ", current_time.Format(timestamp_format))
	spaces := ""
	for _,_ = range timestamp_part {
		spaces += " "
	}
	formatted_text := strings.ReplaceAll(text, "\n", fmt.Sprintf("\n%s\t", spaces))
	formatted_log_str := fmt.Sprintf("%s%s", timestamp_part, formatted_text)

	println(formatted_log_str)

	_, err := os.Stat(logs_base_path)
	if os.IsNotExist(err) {
		err = os.Mkdir(logs_base_path, os.FileMode(0755))
		if err != nil {
			log.Fatal(err)
		}
	}

	if session_filename == "" {
		session_filename = fmt.Sprintf("%s%s.log", logs_base_path, current_time.Format(filename_format))
	}
	file, err := os.OpenFile(session_filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}

	if _, err = file.WriteString(fmt.Sprintf("%s\n", formatted_log_str)); err != nil {
		_ = file.Close()
		panic(err)
	}
	_ = file.Close()
}

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
	logEvent(fmt.Sprintf("Proxy running: %s", getListenAddress()))
	logEvent(fmt.Sprintf("Redirect URL: %s", getProxyAddress()))
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
		logEvent(fmt.Sprintf("Error reading body: %v", err))
		panic(err)
	}

	if len(body) <= 0 {
		return nil
	}

	// Because go lang is a pain in the ass if you read the body then any
	// subsequent calls are unable to read the body again.
	request.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	return json.NewDecoder(request.Body)
}

// Parse the requests body
func parseRequestBody(request *http.Request) requestPayloadStruct {
	decoder := requestBodyDecoder(request)
	var requestPayload requestPayloadStruct
	if decoder != nil {
		err := decoder.Decode(&requestPayload)

		if err != nil {
			panic(err)
		}
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
	url := getProxyAddress()
	go logRequestData(req, url)
	serveReverseProxy(url, res, req)
}

func logRequestData(req *http.Request, url string) {
	requestPayload := parseRequestBody(req)
	logRequestPayload(requestPayload, url)
	logRequest(req)
}

func logRequestPayload(requestPayload requestPayloadStruct, proxyUrl string) {
	logEvent(fmt.Sprintf("proxy_condition: %s, proxy_url: %s", requestPayload.ProxyCondition, proxyUrl))
}

func logRequest(req *http.Request) {
	ipWithPort := req.Header.Get("X-REAL-IP")
	if ipWithPort == "" {
		logEvent("No ip in header, using remoteAddr")
		ipWithPort = req.RemoteAddr
	}
	logIpInfo(ipWithPort)
	logEvent(fmt.Sprintf("Request data:\nUser Agent: '%s'\nIP: '%s'\nRequest URL: '%s'", req.Header.Get("User-Agent"), ipWithPort, req.URL))
}


func logIpInfo(ipWithPort string) {
	ipLookupAddress := getEnv("IP_LOOKUP_ADDRESS", "http://ip-api.com/json/")
	if ipLookupAddress != "" {
		index := strings.Index(ipWithPort, ":")
		ip := ipWithPort
		if index >= 0 {
			ip = ipWithPort[:index]
		}
		logEvent(fmt.Sprintf("IP = '%s'\nIP WITH PORT = '%s'", ip, ipWithPort))

		ipLookupAddress = fmt.Sprintf("%s%s", ipLookupAddress, ip)
		logEvent(fmt.Sprintf("lookup address %s", ipLookupAddress))
		ipMetadata, err := http.Get(ipLookupAddress)
		if err != nil {
			logEvent(fmt.Sprintf("Failed to lookup IP: %s", err))
			return
		}

		ipMetadataJson, err := ioutil.ReadAll(ipMetadata.Body)
		if err != nil {
			logEvent(fmt.Sprintf("%s", err))
			return
		}

		logEvent(fmt.Sprintf("IP metadata: \n%s", ipMetadataJson))
	}
}