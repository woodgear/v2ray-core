#!/bin/bash
function vray_build() {
  mkdir -p build_assets
  go build -gcflags "all=-N -l" -v -o build_assets/MRAY ./main
}

function vray-test() (
  vray_build
  sudo ./build_assets/MRAY run -c ./build_assets/config.json
)

function vray-debug() (
  vray_build
  sudo dlv --log --api-version=2 --headless=true --listen=:2345 exec ./build_assets/MRAY -- run -c ./build_assets/config.json
)

function vray-clean-tproxy() (
  sudo iptables-save | grep -v MRAY | sudo iptables-restore
)

function vray-init-ipset() (
  sudo ipset create mrayip hash:net maxelem 1000000 || true
  sudo ipset add mrayip 172.245.72.75/32
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

function vray-init-tproxy() (
  sudo ip rule add fwmark 2 table 100
  #   sudo ip route add local 0.0.0.0/0 dev lo table 100

  vray-clean-tproxy
  # 代理局域网设备
  sudo iptables -t mangle -N MRAY
  sudo iptables -t mangle -A MRAY ! -d 172.245.72.75/32 -j RETURN
  #   sudo iptables -t mangle -A MRAY -j TRACE
  sudo iptables -t mangle -A MRAY -p tcp -j TPROXY --on-port 10003 --tproxy-mark 2
  sudo iptables -t mangle -A PREROUTING -j MRAY # 应用规则

  # 代理网关本机
  sudo iptables -t mangle -N MRAY_MASK
  sudo iptables -t mangle -A MRAY_MASK ! -d 172.245.72.75/32 -j RETURN
  #   sudo iptables -t mangle -A MRAY_MASK -j TRACE
  sudo iptables -t mangle -A MRAY_MASK -j RETURN -m mark --mark 0xfe # 有0xfe的包 不走代理
  sudo iptables -t mangle -A MRAY_MASK -p tcp -j MARK --set-mark 2   # 给 TCP 打标记，重路由
  sudo iptables -t mangle -A OUTPUT -j MRAY_MASK                     # 应用规则
)
