#!/bin/bash
function vray_build() {
  mkdir -p build_assets
  go build -v -o build_assets/v2ray -trimpath -ldflags "-s -w -buildid=" ./main
}
