package update

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

const dockerAPI = "/v1.41"

type dockerClient struct {
	http *http.Client
	sock string
}

func dockerSock() string {
	if v := strings.TrimSpace(os.Getenv("FENGHUOLUN_DOCKER_SOCK")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("DOCKER_HOST")); strings.HasPrefix(v, "unix://") {
		return strings.TrimPrefix(v, "unix://")
	}
	return "/var/run/docker.sock"
}

func newDocker() *dockerClient {
	sock := dockerSock()
	tr := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", sock)
		},
		DisableCompression: true,
	}
	return &dockerClient{
		sock: sock,
		http: &http.Client{Transport: tr},
	}
}

func DockerPing(ctx context.Context) error {
	c := newDocker()
	resp, err := c.do(ctx, http.MethodGet, dockerAPI+"/_ping", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("docker ping %s", resp.Status)
	}
	return nil
}

func InDocker() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	_, ok := selfContainerID()
	return ok
}

func selfContainerID() (string, bool) {
	candidates := []string{"/proc/self/cgroup", "/proc/self/mountinfo"}
	for _, p := range candidates {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if id := parseContainerID(string(b)); id != "" {
			return id, true
		}
	}
	return "", false
}

var containerIDRe = regexp.MustCompile(`(?:docker[-/]|containers/)([0-9a-f]{12,64})`)

func parseContainerID(s string) string {
	m := containerIDRe.FindStringSubmatch(s)
	if len(m) < 2 {
		return ""
	}
	id := m[1]
	if i := strings.IndexByte(id, '.'); i > 0 {
		id = id[:i]
	}
	if len(id) < 12 {
		return ""
	}
	return id
}

type containerInspect struct {
	ID     string          `json:"Id"`
	Name   string          `json:"Name"`
	Image  string          `json:"Image"`
	Config json.RawMessage `json:"Config"`
	State  struct {
		Running bool   `json:"Running"`
		Status  string `json:"Status"`
		Health  *struct {
			Status string `json:"Status"`
		} `json:"Health"`
	} `json:"State"`
	HostConfig      json.RawMessage `json:"HostConfig"`
	NetworkSettings struct {
		Networks json.RawMessage `json:"Networks"`
	} `json:"NetworkSettings"`
}

func (c containerInspect) NameTrim() string {
	return strings.TrimPrefix(c.Name, "/")
}

func inspectSelf(ctx context.Context) (containerInspect, error) {
	cli := newDocker()
	if id, ok := selfContainerID(); ok {
		if ins, err := cli.inspect(ctx, id); err == nil {
			return ins, nil
		}
	}
	if _, err := os.Stat("/.dockerenv"); err != nil {
		return containerInspect{}, fmt.Errorf("not running inside docker")
	}
	if h, err := os.Hostname(); err == nil && h != "" {
		if ins, err := cli.inspect(ctx, h); err == nil {
			return ins, nil
		}
	}
	name := strings.TrimSpace(os.Getenv("FENGHUOLUN_CONTAINER_NAME"))
	if name == "" {
		name = "fenghuolun"
	}
	return cli.inspect(ctx, name)
}

func (d *dockerClient) do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, "http://docker"+path, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return d.http.Do(req)
}

func (d *dockerClient) inspect(ctx context.Context, id string) (containerInspect, error) {
	var zero containerInspect
	resp, err := d.do(ctx, http.MethodGet, dockerAPI+"/containers/"+url.PathEscape(id)+"/json", nil)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return zero, err
	}
	if resp.StatusCode != http.StatusOK {
		return zero, fmt.Errorf("inspect %s: %s %s", id, resp.Status, truncate(string(b), 180))
	}
	var ins containerInspect
	if err := json.Unmarshal(b, &ins); err != nil {
		return zero, err
	}
	return ins, nil
}

func (d *dockerClient) pull(ctx context.Context, image string) error {
	repo, tag := splitImage(image)
	q := url.Values{}
	q.Set("fromImage", repo)
	q.Set("tag", tag)
	resp, err := d.do(ctx, http.MethodPost, dockerAPI+"/images/create?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		return fmt.Errorf("pull %s: %s %s", image, resp.Status, truncate(string(b), 180))
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		var msg struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(line, &msg) == nil && msg.Error != "" {
			return fmt.Errorf("pull %s: %s", image, msg.Error)
		}
	}
	return sc.Err()
}

func (d *dockerClient) tag(ctx context.Context, source, repo, tag string) error {
	q := url.Values{}
	q.Set("repo", repo)
	q.Set("tag", tag)
	path := dockerAPI + "/images/" + url.PathEscape(source) + "/tag?" + q.Encode()
	resp, err := d.do(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		return fmt.Errorf("tag: %s %s", resp.Status, truncate(string(b), 180))
	}
	return nil
}

func (d *dockerClient) create(ctx context.Context, name string, body []byte) (string, error) {
	q := url.Values{}
	q.Set("name", name)
	resp, err := d.do(ctx, http.MethodPost, dockerAPI+"/containers/create?"+q.Encode(), strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create %s: %s %s", name, resp.Status, truncate(string(b), 180))
	}
	var out struct {
		ID string `json:"Id"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return "", err
	}
	return out.ID, nil
}

func (d *dockerClient) start(ctx context.Context, id string) error {
	resp, err := d.do(ctx, http.MethodPost, dockerAPI+"/containers/"+url.PathEscape(id)+"/start", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		return fmt.Errorf("start: %s %s", resp.Status, truncate(string(b), 180))
	}
	return nil
}

func (d *dockerClient) stop(ctx context.Context, id string, seconds int) error {
	q := url.Values{}
	q.Set("t", fmt.Sprintf("%d", seconds))
	resp, err := d.do(ctx, http.MethodPost, dockerAPI+"/containers/"+url.PathEscape(id)+"/stop?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotModified {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		return fmt.Errorf("stop: %s %s", resp.Status, truncate(string(b), 180))
	}
	return nil
}

func (d *dockerClient) remove(ctx context.Context, id string, force bool) error {
	q := url.Values{}
	if force {
		q.Set("force", "1")
	}
	path := dockerAPI + "/containers/" + url.PathEscape(id)
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	resp, err := d.do(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		return fmt.Errorf("remove: %s %s", resp.Status, truncate(string(b), 180))
	}
	return nil
}

func (d *dockerClient) rename(ctx context.Context, id, name string) error {
	q := url.Values{}
	q.Set("name", name)
	resp, err := d.do(ctx, http.MethodPost, dockerAPI+"/containers/"+url.PathEscape(id)+"/rename?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		return fmt.Errorf("rename: %s %s", resp.Status, truncate(string(b), 180))
	}
	return nil
}

func splitImage(image string) (repo, tag string) {
	image = strings.TrimSpace(image)
	if i := strings.LastIndex(image, ":"); i > 0 && !strings.Contains(image[i+1:], "/") {
		return image[:i], image[i+1:]
	}
	return image, "latest"
}

func buildCreateBody(ins containerInspect, newImage string) ([]byte, error) {
	var cfg map[string]any
	if err := json.Unmarshal(ins.Config, &cfg); err != nil {
		return nil, err
	}
	cfg["Image"] = newImage
	cfg["AttachStdin"] = false
	cfg["AttachStdout"] = false
	cfg["AttachStderr"] = false
	var host map[string]any
	if len(ins.HostConfig) > 0 {
		if err := json.Unmarshal(ins.HostConfig, &host); err != nil {
			return nil, err
		}
	}
	if host == nil {
		host = map[string]any{}
	}
	out := cfg
	out["HostConfig"] = host
	if len(ins.NetworkSettings.Networks) > 0 && string(ins.NetworkSettings.Networks) != "null" {
		out["NetworkingConfig"] = map[string]any{"EndpointsConfig": json.RawMessage(ins.NetworkSettings.Networks)}
	}
	return json.Marshal(out)
}

func buildHelperBody(selfImage, containerName, targetImage, sock string) ([]byte, error) {
	if sock == "" {
		sock = "/var/run/docker.sock"
	}
	body := map[string]any{
		"Image":      selfImage,
		"Entrypoint": []string{"/app/fenghuolun"},
		"Cmd":        []string{"apply-update", "--container=" + containerName, "--image=" + targetImage},
		"Env":        []string{"FENGHUOLUN_DOCKER_SOCK=/var/run/docker.sock"},
		"HostConfig": map[string]any{
			"Binds":         []string{sock + ":/var/run/docker.sock"},
			"AutoRemove":    false,
			"RestartPolicy": map[string]any{"Name": "no"},
			"NetworkMode":   "bridge",
		},
	}
	return json.Marshal(body)
}

func waitRunning(ctx context.Context, d *dockerClient, id string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ins, err := d.inspect(ctx, id)
		if err != nil {
			return err
		}
		if ins.State.Running {
			if ins.State.Health == nil || ins.State.Health.Status == "" || ins.State.Health.Status == "healthy" {
				return nil
			}
			if ins.State.Health.Status == "unhealthy" {
				return fmt.Errorf("new container unhealthy")
			}
		}
		if ins.State.Status == "exited" || ins.State.Status == "dead" {
			return fmt.Errorf("new container %s", ins.State.Status)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return fmt.Errorf("new container did not become ready")
}

func defaultContainerName() string {
	if v := strings.TrimSpace(os.Getenv("FENGHUOLUN_CONTAINER_NAME")); v != "" {
		return v
	}
	return "fenghuolun"
}

func updaterName() string {
	return defaultContainerName() + "-updater"
}

func nextName(name string) string {
	return strings.TrimPrefix(name, "/") + "-next"
}

func sockHostPath() string {
	return dockerSock()
}
