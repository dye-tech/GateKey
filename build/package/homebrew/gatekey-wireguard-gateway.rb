# typed: false
# frozen_string_literal: true

# Homebrew formula for GateKey WireGuard Gateway Agent
# Install with: brew install dye-tech/tap/gatekey-wireguard-gateway
class GatekeyWireguardGateway < Formula
  desc "Zero Trust WireGuard gateway agent for GateKey"
  homepage "https://github.com/dye-tech/GateKey"
  version "VERSION_PLACEHOLDER"
  license "Apache-2.0"

  on_macos do
    on_intel do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-wireguard-gateway-VERSION_PLACEHOLDER-darwin-amd64.tar.gz"
      sha256 "SHA256_DARWIN_AMD64_PLACEHOLDER"
    end

    on_arm do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-wireguard-gateway-VERSION_PLACEHOLDER-darwin-arm64.tar.gz"
      sha256 "SHA256_DARWIN_ARM64_PLACEHOLDER"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-wireguard-gateway-VERSION_PLACEHOLDER-linux-amd64.tar.gz"
      sha256 "SHA256_LINUX_AMD64_PLACEHOLDER"
    end

    on_arm do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-wireguard-gateway-VERSION_PLACEHOLDER-linux-arm64.tar.gz"
      sha256 "SHA256_LINUX_ARM64_PLACEHOLDER"
    end
  end

  depends_on "wireguard-tools" => :recommended

  def install
    bin.install "gatekey-wireguard-gateway"
  end

  def caveats
    <<~EOS
      GateKey WireGuard Gateway Agent has been installed.

      The gateway agent runs alongside WireGuard and requires:
        1. WireGuard kernel module or wireguard-go
        2. Registration with a GateKey control plane
        3. Configuration file at /etc/gatekey/wireguard-gateway.yaml

      For Linux servers, the recommended installation is via the install script:
        curl -sSL https://your-gatekey-server/scripts/install-wireguard-gateway.sh | GATEWAY_TOKEN=<token> bash

      See: https://github.com/dye-tech/GateKey/blob/main/docs/wireguard-gateway-setup.md
    EOS
  end

  test do
    assert_match "gatekey-wireguard-gateway", shell_output("#{bin}/gatekey-wireguard-gateway version 2>&1", 0)
  end
end
