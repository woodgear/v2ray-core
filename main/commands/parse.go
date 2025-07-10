package commands

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	nurl "net/url"
	"os"
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

type ParseFlags struct {
	OutPutPath string
	SSUrl      string
	Base64File string
}

var PARSE_FLAG = ParseFlags{}

func initCmd(c *base.Command, args []string) {
	c.Flag.StringVar(&PARSE_FLAG.OutPutPath, "out-path", "./ss.json", "Path to the output file")
	c.Flag.StringVar(&PARSE_FLAG.SSUrl, "ss-url", "", "Path to the output file")
	c.Flag.StringVar(&PARSE_FLAG.Base64File, "base64-file", "", "Path to the base64 file")
}

func decodebase64(raw string) (string, error) {
	// Try StdEncoding first
	nodes_raw, err := base64.StdEncoding.DecodeString(raw)
	if err == nil {
		return string(nodes_raw), nil
	}

	// Try RawStdEncoding if StdEncoding fails
	nodes_raw, err = base64.RawStdEncoding.DecodeString(raw)
	if err == nil {
		return string(nodes_raw), nil
	}

	// Try URLEncoding if RawStdEncoding fails
	nodes_raw, err = base64.URLEncoding.DecodeString(raw)
	if err == nil {
		return string(nodes_raw), nil
	}

	// Try RawURLEncoding as last resort
	nodes_raw, err = base64.RawURLEncoding.DecodeString(raw)
	if err == nil {
		return string(nodes_raw), nil
	}

	// All decoding methods failed
	return "", fmt.Errorf("failed to decode base64 string with all encoding methods")
}

type ShadowsocksServerTarget struct {
	Address  string `json:"address"`
	Domain   string `json:"domain,omitempty"` // optional, if not set, use address
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

func doParseRaw(raw string) ([]ShadowsocksServerTarget, error) {
	ss := []ShadowsocksServerTarget{}
	nodes, err := decodebase64(raw)
	if err != nil {
		return ss, err
	}
	dns := map[string][]string{}
	for _, n := range strings.Split(nodes, "\n") {
		if strings.TrimSpace(n) == "" {
			continue
		}
		server, err := parse_ss(n)
		if err != nil {
			return ss, err
		}
		ips, err := lookup(server.Address, dns)
		if err != nil {
			return ss, err
		}
		for _, ip := range ips {
			s := server
			s.Address = ip
			s.Domain = server.Address
			ss = append(ss, s)
		}
	}
	ss = lo.UniqBy(ss, func(item ShadowsocksServerTarget) string {
		return fmt.Sprintf("%+v", item)
	})
	return ss, nil
}
func curl(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func read_file(path string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func doParse() error {

	fmt.Printf("opt %+v\n", PARSE_FLAG)
	ss := []ShadowsocksServerTarget{}
	raw := ""
	if PARSE_FLAG.SSUrl != "" {
		body, err := curl(PARSE_FLAG.SSUrl)
		if err != nil {
			return err
		}
		raw = body
	}
	if PARSE_FLAG.Base64File != "" {
		body, err := read_file(PARSE_FLAG.Base64File)
		if err != nil {
			return err
		}
		raw = body
	}
	ss, err := doParseRaw(raw)
	if err != nil {
		return err
	}
	ss = lo.UniqBy(ss, func(item ShadowsocksServerTarget) string {
		return fmt.Sprintf("%+v", item)
	})
	out, err := json.MarshalIndent(ss, "", " ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	os.WriteFile(PARSE_FLAG.OutPutPath, out, 0644)
	return nil
}

func executeParse(cmd *base.Command, args []string) {
	initCmd(cmd, args)
	cmd.Flag.Parse(args)
	if err := doParse(); err != nil {
		panic(err)
	}
}
