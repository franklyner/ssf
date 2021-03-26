package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/franklyner/ssf/server"
)

var (
	serv *server.Server = initServer(".")
)

func TestIndexController(t *testing.T) {
	request := httptest.NewRequest("GET", "/index.html", nil)
	responseRecorder := httptest.NewRecorder()

	serv.GetMainHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != 200 {
		t.Errorf("IndexController returned code %d. Expected 200", responseRecorder.Code)
	}
}

func TestIndexControllerFail(t *testing.T) {
	request := httptest.NewRequest("GET", "/index.html?fail=true", nil)
	responseRecorder := httptest.NewRecorder()

	serv.GetMainHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusBadRequest {
		t.Errorf("IndexController returned code %d. Expected %d", responseRecorder.Code, http.StatusBadRequest)
	}
}

func TestServiceController(t *testing.T) {
	request := httptest.NewRequest("GET", "/service.html", nil)
	responseRecorder := httptest.NewRecorder()

	serv.GetMainHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != 200 {
		t.Errorf("TestServiceController returned code %d. Expected 200", responseRecorder.Code)
	}
}

func TestSecuredController(t *testing.T) {
	baseURI := "/secured.html?"

	ts := []struct {
		name  string
		param string
		code  int
	}{
		{
			name:  "success",
			param: "secure=true",
			code:  200,
		},
		{
			name:  "fail",
			param: "secure=false",
			code:  http.StatusBadRequest,
		},
	}

	for _, tc := range ts {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", baseURI+tc.param, nil)
			responseRecorder := httptest.NewRecorder()

			serv.GetMainHandler().ServeHTTP(responseRecorder, request)

			if responseRecorder.Code != tc.code {
				t.Errorf("test case %s returned code %d. Expected %d", tc.name, responseRecorder.Code, tc.code)
			}
		})
	}
}

func TestStatusController(t *testing.T) {
	request := httptest.NewRequest("GET", "/status", nil)
	request.Header.Add("x-request-id", "request-id-from-header")
	responseRecorder := httptest.NewRecorder()

	serv.GetMainHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != 200 {
		t.Errorf("StatusController returned code %d. Expected 200", responseRecorder.Code)
	}

	body, err := ioutil.ReadAll(responseRecorder.Result().Body)
	if err != nil {
		t.Errorf("Failed to read response body: %w", err)
	}
	log.Print(string(body))
}

func TestJWKValidation(t *testing.T) {
	jwt := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6Ik1qVkRORFl4UkVFNU5VSTNPRFJETlRGR05ETkRRVFpHTTBVMU9EZ3lNakV6TUROQ05UVTVPQSJ9.eyJodHRwczovL21heGJyYWluLmlvL3VzZXJfZW1haWwiOiJrcmlzdGluYS5zZWt1bGljQHEtc29mdHdhcmUuY29tIiwiaHR0cHM6Ly9tYXhicmFpbi5pby90ZW5hbnRfaWQiOiI2MTAiLCJodHRwczovL21heGJyYWluLmlvL3RlbmFudF9zdWJkb21haW4iOiJwYW1kdWYiLCJodHRwczovL21heGJyYWluLmlvL2lzX2NvY2twaXRfdXNlciI6dHJ1ZSwiaXNzIjoiaHR0cHM6Ly9tYXhicmFpbi1kZXYuZXUuYXV0aDAuY29tLyIsInN1YiI6ImF1dGgwfDVjM2M1ZGY5NDFhZmQ5N2NmYWZlNzQ1NyIsImF1ZCI6WyJodHRwczovL2NvY2twaXQubWF4YnJhaW4uaW8vYXBpLyIsImh0dHBzOi8vbWF4YnJhaW4tZGV2LmV1LmF1dGgwLmNvbS91c2VyaW5mbyJdLCJpYXQiOjE2MTY3NzgzOTMsImV4cCI6MTYxNjc4NTU5MywiYXpwIjoidmRnbWtRM25xamJXSDJZRE8wNnNPb1c1RXF2UGF4SngiLCJzY29wZSI6Im9wZW5pZCBwcm9maWxlIGVtYWlsIn0.QxBWAu8jdmoV48uKpZ7H54v8_dHWPLVp84f5PqRgn3sbslbwIts9z65Z3v2ZpjXAEsCLAC1tZqNh5iF_Gn0YjpvYv0cL3wMrp0JXFqybJG-59mbBxlSuFIgRcp4v2LncSrXoTixO1Yg0YS-J1KDrBtfAV2VjFAVP4u-CrH7_fdESO66TxUeX6oIgdEo1SKh-FTyLGyZvNHWrE6IHpJ2t_ROW5iCBqwF4qaHxOIFu3yasUBFlGwQ8l_i10vOSkBpme32Htmv0mnvvPHQdS72DWb3DgqZ-kEaZ-6QjGEgqA-pRKNRGm2kcru1s3L_4cWfyKU9JqtJxV8WtOBNSLLLQ7Q"
	request := httptest.NewRequest("GET", "/jwt.html", nil)
	request.Header.Add("x-request-id", "request-id-from-header")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwt))
	responseRecorder := httptest.NewRecorder()

	serv.GetMainHandler().ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != 200 {
		t.Errorf("JWTController returned code %d. Expected 200", responseRecorder.Code)
	}
	request = httptest.NewRequest("GET", "/jwt.html", nil)
	request.Header.Add("x-request-id", "request-id-from-header")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwt))
	responseRecorder = httptest.NewRecorder()

	serv.GetMainHandler().ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != 200 {
		t.Errorf("JWTController returned code %d. Expected 200", responseRecorder.Code)
	}
}
