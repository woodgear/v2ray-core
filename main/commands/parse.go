package commands

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	nurl "net/url"
	"strconv"
	"strings"

	"github.com/samber/lo"
	"github.com/v2fly/v2ray-core/v5/main/commands/base"
)

var CmdParse = &base.Command{
	CustomFlags: true,
	UsageLine:   "{{.Exec}} parse",
	Short:       "parse subscribe link to v2ray server config",
	Run:         executeParse,
}

func decodebase64(raw string) (string, error) {
	nodes_raw, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", err
	}
	return string(nodes_raw), nil
}

type ShadowsocksServerTarget struct {
	Address  string `json:"address"`
	Port     uint16 `json:"port"`
	Method   string `json:"method"`
	Password string `json:"password"`
	Mark     string `json:"mark"`
}

func parse_ss(url_str string) (server ShadowsocksServerTarget, err error) {
	t := ShadowsocksServerTarget{}
	url, err := nurl.Parse(url_str)
	if err != nil {
		return t, err
	}
	t.Mark = url.Fragment
	if url.User.String() == "" {
		// base64的情况
		infos, err := decodebase64(url.Hostname())
		if err != nil {
			return t, err
		}
		url, err = nurl.Parse("ss://" + infos)
		if err != nil {
			return t, err
		}
		method := url.User.Username()
		passwd, _ := url.User.Password()
		t.Method = method
		t.Password = passwd
	} else {
		cipherInfoString, err := decodebase64(url.User.Username())
		if err != nil {
			return t, err
		}
		cipherInfo := strings.SplitN(cipherInfoString, ":", 2)
		if err != nil {
			return t, err
		}
		method := strings.ToLower(cipherInfo[0])
		passwd := cipherInfo[1]
		t.Method = method
		t.Password = passwd
	}
	address := url.Hostname()
	port, err := strconv.Atoi(url.Port())
	if err != nil {
		return t, err
	}
	t.Address = address
	t.Port = uint16(port)
	return t, nil
}

func lookup(domain string, cache map[string][]string) ([]string, error) {
	if ips, ok := cache[domain]; ok {
		return ips, nil
	}
	ips, err := net.LookupHost(domain)
	if err != nil {
		return nil, err
	}
	cache[domain] = ips
	return ips, nil

}
func doParse(urls []string) error {
	dns := map[string][]string{}
	fmt.Println(urls, len(urls))
	ss := []ShadowsocksServerTarget{}
	for _, url := range urls {
		fmt.Println(url)
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		nodes, err := decodebase64(string(body))
		if err != nil {
			return err
		}
		for _, n := range strings.Split(nodes, "\n") {
			if strings.TrimSpace(n) == "" {
				continue
			}
			server, err := parse_ss(n)
			if err != nil {
				return err
			}
			ips, err := lookup(server.Address, dns)
			if err != nil {
				return err
			}
			for _, ip := range ips {
				s := server
				s.Address = ip
				ss = append(ss, s)
			}
		}
	}
	ss = lo.UniqBy(ss, func(item ShadowsocksServerTarget) string {
		return fmt.Sprintf("%+v", item)
	})
	out, err := json.MarshalIndent(ss, "", " ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func executeParse(cmd *base.Command, args []string) {
	if err := doParse(args); err != nil {
		panic(err)
	}
}
