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
	cfg                 = server.CreateConfig("./", "minimal", ConfigProperties)
	serv *server.Server = initServer(cfg)
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
			code:  http.StatusUnauthorized,
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
	jwt := "Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6InIwTllfTFBiNWxKc0dRPT0iLCJ0eXAiOiJKV1QifQ.eyJhdWQiOlsiaHR0cHM6Ly9jb2NrcGl0Lm1heGJyYWluLmlvL2FwaS8iXSwiZXhwIjoxNjM0MDA3NTA1LCJpYXQiOjE2MzQwMDM5MDUsImlzcyI6Imh0dHBzOi8vbG9naW4uZGV2Lm1heGJyYWluLmlvLyIsImh0dHBzOi8vbWF4YnJhaW4uaW8vdXNlcl9lbWFpbCI6ImZyYW5rLmx5bmVyQG1heGJyYWluLmNvbSIsImh0dHBzOi8vbWF4YnJhaW4uaW8vdGVuYW50X2lkIjoiNjEwIiwiaHR0cHM6Ly9tYXhicmFpbi5pby90ZW5hbnRfc3ViZG9tYWluIjoicGFtZHVmIiwiaHR0cHM6Ly9tYXhicmFpbi5pby9pc19jb2NrcGl0X3VzZXIiOnRydWUsImh0dHBzOi8vbWF4YnJhaW4uaW8vaWRlbnRpdHlfaWQiOiI1YmZkNDRjOC05OTM2LTExZWItOTcyNy0wMDBkM2E4MzU2ZTkifQ.bghXkgoYskFuFTMg-SddKTmjHOghQxEDO4h6JUUwTGgS5Ci-GcRQkZX3rMEZYeFE-ZHzzokkpBZYH3xuX0UkuZ6aX7X91S3DUrfne--fgRuuHF0LdIdq_8RkRFpaJT33MgpoRPXD0tmuDM9p-BlFcIICUYw9mOqYfelCLldyUy_9nO04Qi4SXqfzoWfKv6J_AQQslFQNbOgEpLGPQXwQB_v0at9EK9OKYDPUBeCnCxc1HoNOtL0slDpxMPWH24SObXkyAt_I-FgFJ5C7rJG3bCiXU_ELjinH64T3TOKVdyaeDVrt0a-wVKOjE6dJOCtZ6QckZAXKoFJ2kreZZD0nhg"
	request := httptest.NewRequest("GET", "/jwt.html", nil)
	request.Header.Add("x-request-id", "request-id-from-header")
	request.Header.Add("Authorization", jwt)
	responseRecorder := httptest.NewRecorder()

	serv.GetMainHandler().ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != 200 {
		t.Errorf("JWTController returned code %d. Expected 200", responseRecorder.Code)
	}
	request = httptest.NewRequest("GET", "/jwt.html", nil)
	request.Header.Add("x-request-id", "request-id-from-header")
	request.Header.Add("Authorization", jwt)
	responseRecorder = httptest.NewRecorder()

	serv.GetMainHandler().ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != 200 {
		t.Errorf("JWTController returned code %d. Expected 200", responseRecorder.Code)
	}
}

func TestLogLevelController(t *testing.T) {
	config := server.CreateConfig("./", "minimal", ConfigProperties)

	config.SetProperty(server.ConfigLogLevel, "info")
	fmt.Println("Loglevel set to info")
	srv := initServer(config)
	request := httptest.NewRequest("GET", "/loglevel", nil)
	responseRecorder := httptest.NewRecorder()

	srv.GetMainHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != 200 {
		t.Errorf("LogLevelController returned code %d. Expected 200", responseRecorder.Code)
	}
	config.SetProperty(server.ConfigLogLevel, "debug")
	fmt.Println("Loglevel set to debug")
	srv = initServer(config)
	request = httptest.NewRequest("GET", "/loglevel", nil)
	responseRecorder = httptest.NewRecorder()

	srv.GetMainHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != 200 {
		t.Errorf("LogLevelController returned code %d. Expected 200", responseRecorder.Code)
	}
}
