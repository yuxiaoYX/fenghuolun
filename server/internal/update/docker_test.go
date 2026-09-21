package update

import (
	"encoding/json"
	"testing"
)

func TestParseContainerID(t *testing.T) {
	cgroup := "0::/system.slice/docker-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.scope"
	id := parseContainerID(cgroup)
	if len(id) < 12 {
		t.Fatalf("id %q", id)
	}
	mount := "overlay / /var/lib/docker/containers/aaaaaaaaaaaabbbbbbbbbbbbccccccccccccccccdddddddddddd/merged"
	if parseContainerID(mount) == "" {
		t.Fatal("mountinfo")
	}
}

func TestSplitImage(t *testing.T) {
	repo, tag := splitImage("ghcr.io/yuxiaoyx/fenghuolun:v0.1.1")
	if repo != "ghcr.io/yuxiaoyx/fenghuolun" || tag != "v0.1.1" {
		t.Fatalf("%s %s", repo, tag)
	}
}

func TestBuildCreateBody(t *testing.T) {
	ins := containerInspect{
		Config:     json.RawMessage(`{"Image":"old:v0","Env":["A=1"],"Cmd":["/app/fenghuolun"]}`),
		HostConfig: json.RawMessage(`{"Binds":["/opt/fenghuolun/data:/var/lib/fenghuolun"]}`),
		NetworkSettings: struct {
			Networks json.RawMessage `json:"Networks"`
		}{Networks: json.RawMessage(`{"fenghuolun_default":{"NetworkID":"abc"}}`)},
	}
	b, err := buildCreateBody(ins, "ghcr.io/yuxiaoyx/fenghuolun:v0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out["Image"] != "ghcr.io/yuxiaoyx/fenghuolun:v0.2.0" {
		t.Fatalf("image %v", out["Image"])
	}
	if _, ok := out["HostConfig"]; !ok {
		t.Fatal("missing HostConfig")
	}
	if _, ok := out["NetworkingConfig"]; !ok {
		t.Fatal("missing NetworkingConfig")
	}
}

func TestBuildHelperBody(t *testing.T) {
	b, err := buildHelperBody("img:new", "fenghuolun", "img:new", "/var/run/docker.sock")
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	cmd, _ := out["Cmd"].([]any)
	if len(cmd) < 3 || cmd[0] != "apply-update" {
		t.Fatalf("cmd %v", out["Cmd"])
	}
}
