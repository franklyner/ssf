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
	request := httptest.NewRequest("GET", PREFIX+"/index.html", nil)
	responseRecorder := httptest.NewRecorder()

	serv.GetMainHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != 200 {
		t.Errorf("IndexController returned code %d. Expected 200", responseRecorder.Code)
	}
}

func TestIndexControllerFail(t *testing.T) {
	request := httptest.NewRequest("GET", PREFIX+"/index.html?fail=true", nil)
	responseRecorder := httptest.NewRecorder()

	serv.GetMainHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusBadRequest {
		t.Errorf("IndexController returned code %d. Expected %d", responseRecorder.Code, http.StatusBadRequest)
	}
}

func TestServiceController(t *testing.T) {
	request := httptest.NewRequest("GET", PREFIX+"/service.html", nil)
	responseRecorder := httptest.NewRecorder()

	serv.GetMainHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != 200 {
		t.Errorf("TestServiceController returned code %d. Expected 200", responseRecorder.Code)
	}
}

func TestSecuredController(t *testing.T) {
	baseURI := PREFIX + "/secured.html?"

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
	request := httptest.NewRequest("GET", PREFIX+"/status", nil)
	request.Header.Add("x-request-id", "request-id-from-header")
	responseRecorder := httptest.NewRecorder()

	serv.GetMainHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != 200 {
		t.Errorf("StatusController returned code %d. Expected 200", responseRecorder.Code)
	}

	body, err := ioutil.ReadAll(responseRecorder.Result().Body)
	if err != nil {
		t.Errorf("Failed to read response body: %s", err)
	}
	log.Print(string(body))
}

func TestJWKValidation(t *testing.T) {
	jwt := "Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6InIwTllfTFBiNWxKc0dRPT0iLCJ0eXAiOiJKV1QifQ.eyJhdWQiOlsiaHR0cHM6Ly9jb2NrcGl0Lm1heGJyYWluLmlvL2FwaS8iXSwiZXhwIjoxNzUyMDY2OTYxLCJpYXQiOjE3NTIwNjMzNjEsImlzcyI6Imh0dHBzOi8vbG9naW4uZGV2Lm1heGJyYWluLmlvLyIsImh0dHBzOi8vbWF4YnJhaW4uaW8vdXNlcl9lbWFpbCI6ImZyYW5rLmx5bmVyQG1heGJyYWluLmNvbSIsImh0dHBzOi8vbWF4YnJhaW4uaW8vdGVuYW50X2lkIjoiMTA4IiwiaHR0cHM6Ly9tYXhicmFpbi5pby90ZW5hbnRfc3ViZG9tYWluIjoiZGV2b3NhdXJ1cyIsImh0dHBzOi8vbWF4YnJhaW4uaW8vaXNfY29ja3BpdF91c2VyIjp0cnVlLCJodHRwczovL21heGJyYWluLmlvL2lkZW50aXR5X2lkIjoiM2M1YjBjZDMtNTQ0OS0xMWVkLTkzNDQtMDAxNjNlOGI5YzY3In0.Gi-P285mr1iSNNtpPxue8am1E4ULEL0nlFZbJHjgX2mPOEkV4q6EBDSN0pU-exgJgxJf3vkrq9xZYt5dt1hzdHgP5mzq-0ocRtk3Gd9zxkVKi0EnvDroQyldoigOJ-GtBdDMvUmUje2wh6Nuw4HSrEjvutZ-UyqLy3GN-QecngatD6VqAA5Q0QScbxRo7U-6YZmW2jUwwyr5TLhhUwW6Fllc4BDGqTopfBNm_qH-PRqS900upaMXMmU4SeBnKLz0d9d7DLwVDWhnVZP75VG74ipaZ2FebVy53f4lFRFOuG2QyuOSfM7J_WEzQncv3KvTLMUY2dRiwL4jdX6NTNCWew"
	request := httptest.NewRequest("GET", PREFIX+"/jwt.html", nil)
	request.Header.Add("x-request-id", "request-id-from-header")
	request.Header.Add("Authorization", jwt)
	responseRecorder := httptest.NewRecorder()

	serv.GetMainHandler().ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != 200 {
		t.Errorf("JWTController returned code %d. Expected 200", responseRecorder.Code)
	}
	request = httptest.NewRequest("GET", PREFIX+"/jwt.html", nil)
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
	serv := initServer(config)
	request := httptest.NewRequest("GET", PREFIX+"/loglevel", nil)
	responseRecorder := httptest.NewRecorder()

	serv.GetMainHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != 200 {
		t.Errorf("LogLevelController returned code %d. Expected 200", responseRecorder.Code)
	}
	config.SetProperty(server.ConfigLogLevel, "debug")
	fmt.Println("Loglevel set to debug")
	serv = initServer(config)
	request = httptest.NewRequest("GET", PREFIX+"/loglevel", nil)
	responseRecorder = httptest.NewRecorder()

	serv.GetMainHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != 200 {
		t.Errorf("LogLevelController returned code %d. Expected 200", responseRecorder.Code)
	}
}
