package api

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/tmunzer/mistapi-go/mistapi"
)

// sdkTransport replaces the entire HTTP transport, so no request can reach Mist.
type sdkTransport func(*http.Request) (*http.Response, error)

func (transport sdkTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func offlineSDKClient(t *testing.T, status int, body string, check func(*http.Request)) *Client {
	t.Helper()
	transport := sdkTransport(func(request *http.Request) (*http.Response, error) {
		check(request)
		return &http.Response{
			StatusCode: status,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    request,
		}, nil
	})
	config := mistapi.CreateConfiguration(
		mistapi.WithApiTokenCredentials(mistapi.NewApiTokenCredentials("Token offline-fixture")),
		mistapi.WithHttpConfiguration(mistapi.CreateHttpConfiguration(mistapi.WithTransport(transport))),
	)
	return &Client{sdk: mistapi.NewClient(config)}
}

func checkSDKRequest(t *testing.T, request *http.Request, endpoint string, query url.Values) {
	t.Helper()
	if request.Method != http.MethodGet || request.URL.Path != "/api/v1/orgs/"+uuid.Nil.String()+"/"+endpoint {
		t.Errorf("unexpected request: %s %s", request.Method, request.URL.Path)
	}
	if !reflect.DeepEqual(request.URL.Query(), query) {
		t.Errorf("query = %v; want %v", request.URL.Query(), query)
	}
	if request.Header.Get("Authorization") != "Token offline-fixture" {
		t.Error("SDK did not preserve the token authorization header")
	}
}

func TestSDKSiteHTTPContract(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		status  int
		body    string
		wantLen int
		wantErr bool
	}{
		{"record", http.StatusOK, `[{"id":"00000000-0000-0000-0000-000000000001","name":"Lab, West"}]`, 1, false},
		{"empty", http.StatusOK, `[]`, 0, false},
		{"malformed", http.StatusOK, `[`, 0, true},
		{"forbidden", http.StatusForbidden, `{"detail":"access denied"}`, 0, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := offlineSDKClient(t, test.status, test.body, func(request *http.Request) {
				checkSDKRequest(t, request, "sites", url.Values{"limit": {"1000"}, "page": {"2"}})
			})
			sites, err := client.fetchPageFromSDK(context.Background(), uuid.Nil, 1000, 2)
			if (err != nil) != test.wantErr || len(sites) != test.wantLen {
				t.Fatalf("sites = %d, error = %v; want %d, error %v", len(sites), err, test.wantLen, test.wantErr)
			}
			if len(sites) == 1 {
				rows, err := sitesToMaps(sites)
				if err != nil || rows[0]["name"] != "Lab, West" || rows[0]["id"] != "00000000-0000-0000-0000-000000000001" {
					t.Fatalf("SDK conversion lost fixture fields: %v, %v", rows, err)
				}
			}
		})
	}
}

func TestSDKInventoryHTTPContract(t *testing.T) {
	t.Parallel()
	client := offlineSDKClient(t, http.StatusOK, `[{"mac":"001122334455","serial":"TEST-ONLY","type":"switch","model":"EX4400"}]`, func(request *http.Request) {
		// Exact query comparison also checks that optional device and time filters stay unset.
		checkSDKRequest(t, request, "inventory", url.Values{"limit": {"1000"}, "page": {"3"}, "vc": {"true"}})
	})
	inventory, err := client.fetchInventoryPageFromSDK(context.Background(), uuid.Nil, 1000, 3, true)
	if err != nil || len(inventory) != 1 {
		t.Fatalf("inventory = %v, error = %v", inventory, err)
	}
	rows, err := inventoryToMaps(inventory)
	if err != nil || rows[0]["mac"] != "001122334455" || rows[0]["type"] != "switch" || rows[0]["serial"] != "TEST-ONLY" {
		t.Fatalf("SDK conversion lost inventory fields: %v, %v", rows, err)
	}
}
