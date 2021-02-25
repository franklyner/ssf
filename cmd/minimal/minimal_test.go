package main

import (
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
