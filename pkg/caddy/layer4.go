package caddy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sdslabs/beastv4/core/config"
)

const httpTimeout = 30 * time.Second

type layer4ServerBody struct {
	ID     string       `json:"@id"`
	Listen []string     `json:"listen"`
	Routes []routeBlock `json:"routes"`
}

type routeBlock struct {
	Handle []handleBlock `json:"handle"`
}

type handleBlock struct {
	Handler   string         `json:"handler"`
	Upstreams []upstreamDial `json:"upstreams"`
}

type upstreamDial struct {
	Dial []string `json:"dial"`
}

func normalizeAdminBase(adminAPIURL string) string {
	return strings.TrimRight(strings.TrimSpace(adminAPIURL), "/")
}

// Layer4Enabled is true when the Caddy admin API base URL is configured.
func Layer4Enabled(p *config.CaddySshProxyConfig) bool {
	return p != nil && strings.TrimSpace(p.AdminAPIURL) != ""
}

// PutLayer4Server registers or replaces a layer4 TCP server (SSH proxy path) on the Caddy admin API.
func PutLayer4Server(adminAPIURL, instanceID string, hostPort uint32, dialTarget string) error {
	base := normalizeAdminBase(adminAPIURL)
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid caddy admin API URL: %q", adminAPIURL)
	}

	body := layer4ServerBody{
		ID:     instanceID,
		Listen: []string{":" + strconv.FormatUint(uint64(hostPort), 10)},
		Routes: []routeBlock{
			{
				Handle: []handleBlock{
					{
						Handler: "proxy",
						Upstreams: []upstreamDial{
							{Dial: []string{dialTarget}},
						},
					},
				},
			},
		},
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal layer4 server config: %w", err)
	}

	path := fmt.Sprintf("%s/config/apps/layer4/servers/%s", base, url.PathEscape(instanceID))
	req, err := http.NewRequest(http.MethodPut, path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("caddy admin PUT %s: %w", path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("caddy admin PUT %s: status %s: %s", path, resp.Status, strings.TrimSpace(string(b)))
	}
	return nil
}

// DeleteByID removes a config object by @id via the Caddy admin API.
func DeleteByID(adminAPIURL, id string) error {
	base := normalizeAdminBase(adminAPIURL)
	path := fmt.Sprintf("%s/id/%s", base, url.PathEscape(id))
	req, err := http.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("caddy admin DELETE %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("caddy admin DELETE %s: status %s: %s", path, resp.Status, strings.TrimSpace(string(b)))
	}
	return nil
}
