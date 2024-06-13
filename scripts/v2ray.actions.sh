#!/bin/bash
function vray_build() {
  mkdir -p build_assets
  go build -gcflags "all=-N -l" -v -o build_assets/MRAY ./main
}

function vray-test() (
  vray_build
  sudo ./build_assets/MRAY run -c ./build_assets/config.json
)

function vray-init-cfg() (
    sudo cp /etc/v2ray/config.json ./build_assets/config.json
    local cfg=$(cat ./build_assets/config.json)
    echo "$cfg" | jq '.inbounds[0].port = 10003' | jq '.outbounds[0].streamSettings.sockopt.mark = 254' > ./build_assets/config.json
)

function vray-debug() (
  vray_build
  sudo dlv --log --api-version=2 --headless=true --listen=:2345 exec ./build_assets/MRAY -- run -c ./build_assets/config.json
)

function vray-clean-tproxy() (
  sudo iptables-save | grep -v MRAY | sudo iptables-restore
  sudo iptables-save | grep -v V2RAY | sudo iptables-restore
)

function vray-init-localnet() (
  sudo ipset destroy localnet || true
  sudo ipset create localnet hash:net maxelem 1000000
  sudo ipset add localnet 10.0.0.0/8
  sudo ipset add localnet 127.0.0.0/8
  sudo ipset add localnet 172.16.0.0/12
  sudo ipset add localnet 192.168.0.0/16
)

function vray-init-myvps() (
  sudo ipset destroy myvps || true
  sudo ipset create myvps hash:net maxelem 1000000
  sudo ipset add myvps 172.245.72.75/32
)

function vray-note() (
  local note=$(
    cat <<EOF
1. vray-init-ipset
2. vray-init-tproxy
3. 检查ipset mrayip 检查iptable规则 vray 应避 mrayip的规则 mray 设置的规则正常
EOF
  )
)

function vray-init-mray() (
  # OUTPUT 本机往外的流量
  sudo iptables -t mangle -N MRAY_MASK
  #   sudo iptables -t mangle -A MRAY_MASK -j TRACE
  sudo iptables -t mangle -A MRAY_MASK -m set ! --match-set myvps dst -j RETURN
  sudo iptables -t mangle -A MRAY_MASK -m mark --mark 0xfe -j RETURN # 有0xfe的包 不走代理 0xfe 是 v2ray 配置的 254
  sudo iptables -t mangle -A MRAY_MASK -p tcp -j MARK --set-mark 2   # 给所有走到这里的包打标记2
  sudo iptables -t mangle -A OUTPUT -j MRAY_MASK

  # PREROUTING 局域网发往本机的流量
  sudo iptables -t mangle -N MRAY
  #   sudo iptables -t mangle -A MRAY -j TRACE
  sudo iptables -t mangle -A MRAY -m set ! --match-set myvps dst -j RETURN         # mray 只处理特定的那几个vps的ip
  sudo iptables -t mangle -A MRAY -p tcp -j TPROXY --on-port 10003 --tproxy-mark 2 # 走到这里的只可能是发向vps的包。走tproxy
  sudo iptables -t mangle -A PREROUTING -j MRAY

  sudo ip rule add fwmark 2 table 100                # 有标记2的走100表
  sudo ip route add local 0.0.0.0/0 dev lo table 100 # 在路由表 100 中增加了一条规则，指示所有目的地址为 0.0.0.0/0 的流量都重新走本地回环接口进行处理 -> 重新走到pre-route
)

function vray-init-vray() (
  # vray
  ## output
  sudo iptables -t mangle -N V2RAY_MASK
  sudo iptables -t mangle -A V2RAY_MASK -m set --match-set localnet dst -j RETURN

  sudo iptables -t mangle -A V2RAY_MASK -m set --match-set chinaip dst -j RETURN
  sudo iptables -t mangle -A V2RAY_MASK -m set --match-set myvps dst -j RETURN

  sudo iptables -t mangle -A V2RAY_MASK -m mark --mark 0xff -j RETURN # 直连 SO_MARK 为 0xff 的流量(0xff 是 16 进制数，数值上等同与上面V2Ray 配置的 255)，此规则目的是避免代理本机(网关)流量出现回环问题
  sudo iptables -t mangle -A V2RAY_MASK -p udp -j MARK --set-mark 1   # 给 UDP 打标记,重路由
  sudo iptables -t mangle -A V2RAY_MASK -p tcp -j MARK --set-mark 1   # 给 TCP 打标记，重路由
  sudo iptables -t mangle -A OUTPUT -j V2RAY_MASK                     # 应用规则

  sudo iptables -t mangle -N V2RAY

  sudo iptables -t mangle -A V2RAY -m set --match-set localnet dst -j RETURN
  sudo iptables -t mangle -A V2RAY -m set --match-set chinaip dst -j RETURN
  sudo iptables -t mangle -A V2RAY -m set --match-set myvps dst -j RETURN

  sudo iptables -t mangle -A V2RAY -j TRACE
  sudo iptables -t mangle -A V2RAY -p udp -j TPROXY --on-port 10002 --tproxy-mark 1 # 给 UDP 打标记 1，转发至 10002 端口
  sudo iptables -t mangle -A V2RAY -p tcp -j TPROXY --on-port 10002 --tproxy-mark 1 # 给 TCP 打标记 1，转发至 10002 端口
  sudo iptables -t mangle -A PREROUTING -j V2RAY                                    # 应用规则

  sudo ip rule add fwmark 1 table 100
  sudo ip route add local 0.0.0.0/0 dev lo table 100 # 在路由表 100 中增加了一条规则，指示所有目的地址为 0.0.0.0/0 的流量都重新走本地回环接口进行处理 -> 重新走到pre-route
)

function vray-init-tproxy() (
  set -x
  vray-clean-tproxy
  sudo ipset list localnet | wc -l
  sudo ipset list chinaip | wc -l
  sudo ipset list myvps | wc -l
  sudo ip route add local 0.0.0.0/0 dev lo table 100 # 在路由表 100 中增加了一条规则，指示所有目的地址为 0.0.0.0/0 的流量都重新走本地回环接口进行处理 -> 重新走到pre-route

)
