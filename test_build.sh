#!/bin/bash
go mod edit -go=1.24.5
go get tailscale.com@v1.90.8
go mod tidy
sed -i 's/buildGo124Module/buildGo125Module/g' package.nix
sed -i 's/vendorHash = ".*"/vendorHash = "sha256-EkqIQSaD9sL6Y\/6K0lpIo7maO4ZWhgd3d0QtmuAQPRU="/' package.nix
