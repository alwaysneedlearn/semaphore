package projects

import (
	"bytes"
	"net/http"
	"testing"
)

func TestParseDiscoveryPutRequest_devicesArray(t *testing.T) {
	body := `{"task_id":7,"devices":[{"hostname":"h1","ip_address":"10.0.0.1","device_status":"healthy","rdp_status":"online","winrm_status":"online","api_status":"online","api_port":9002}]}`
	r, _ := http.NewRequest(http.MethodPut, "/", bytes.NewBufferString(body))
	taskID, devices, err := parseDiscoveryPutRequest(r)
	if err != nil {
		t.Fatal(err)
	}
	if taskID != 7 || len(devices) != 1 || devices[0].IPAddress != "10.0.0.1" {
		t.Fatalf("unexpected: id=%d devices=%+v", taskID, devices)
	}
}

func TestParseDiscoveryPutRequest_devicesAsJSONString(t *testing.T) {
	body := `{"task_id":"7","devices":"[{\"hostname\":\"h1\",\"ip_address\":\"10.0.0.2\",\"device_status\":\"healthy\"}]"}`
	r, _ := http.NewRequest(http.MethodPut, "/", bytes.NewBufferString(body))
	taskID, devices, err := parseDiscoveryPutRequest(r)
	if err != nil {
		t.Fatal(err)
	}
	if taskID != 7 || len(devices) != 1 || devices[0].IPAddress != "10.0.0.2" {
		t.Fatalf("unexpected: id=%d devices=%+v", taskID, devices)
	}
}
