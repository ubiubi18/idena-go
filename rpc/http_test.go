// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type PeerInfoService struct{}

func (PeerInfoService) Info(ctx context.Context) PeerInfo {
	return PeerInfoFromContext(ctx)
}

func TestHTTPPeerInfo(t *testing.T) {
	server := NewServer("")
	defer server.Stop()
	if err := server.RegisterName("test", PeerInfoService{}); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "http://node.example", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"test_info","params":[]}`))
	request.Header.Set("content-type", contentType)
	request.Header.Set("User-Agent", "idena-test-client")
	request.Header.Set("Origin", "https://app.example")
	request.RemoteAddr = "192.0.2.1:1234"
	request.Host = "node.example"
	request.Proto = "HTTP/2.0"

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected response status: %d", recorder.Code)
	}

	var response struct {
		Result PeerInfo `json:"result"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Result.Transport != "http" || response.Result.RemoteAddr != request.RemoteAddr {
		t.Fatalf("unexpected peer info: %+v", response.Result)
	}
	if response.Result.HTTP.Version != request.Proto || response.Result.HTTP.Host != request.Host ||
		response.Result.HTTP.UserAgent != "idena-test-client" || response.Result.HTTP.Origin != "https://app.example" {
		t.Fatalf("unexpected HTTP peer info: %+v", response.Result.HTTP)
	}
}

func TestHTTPErrorResponseWithDelete(t *testing.T) {
	testHTTPErrorResponse(t, http.MethodDelete, contentType, "", http.StatusMethodNotAllowed)
}

func TestHTTPErrorResponseWithPut(t *testing.T) {
	testHTTPErrorResponse(t, http.MethodPut, contentType, "", http.StatusMethodNotAllowed)
}

func TestHTTPErrorResponseWithMaxContentLength(t *testing.T) {
	body := make([]rune, maxRequestContentLength+1)
	testHTTPErrorResponse(t,
		http.MethodPost, contentType, string(body), http.StatusRequestEntityTooLarge)
}

func TestHTTPErrorResponseWithEmptyContentType(t *testing.T) {
	testHTTPErrorResponse(t, http.MethodPost, "", "", http.StatusUnsupportedMediaType)
}

func TestHTTPErrorResponseWithValidRequest(t *testing.T) {
	testHTTPErrorResponse(t, http.MethodPost, contentType, "", 0)
}

func testHTTPErrorResponse(t *testing.T, method, contentType, body string, expected int) {
	request := httptest.NewRequest(method, "http://url.com", strings.NewReader(body))
	request.Header.Set("content-type", contentType)
	if code, _ := validateRequest(request); code != expected {
		t.Fatalf("response code should be %d not %d", expected, code)
	}
}
