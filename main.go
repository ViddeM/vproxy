package vproxy

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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

// Given a request send it to the appropriate url
func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {
	// TODO: Implement this.
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

// Given a request send it to the appropriate url
func handleRequestAndRedirect