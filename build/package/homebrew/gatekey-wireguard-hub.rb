# typed: false
# frozen_string_literal: true

# Homebrew formula for GateKey WireGuard Mesh Hub
# Install with: brew install dye-tech/tap/gatekey-wireguard-hub
class GatekeyWireguardHub < Formula
  desc "Zero Trust WireGuard mesh hub for GateKey"
  homepage "https://github.com/dye-tech/GateKey"
  version "VERSION_PLACEHOLDER"
  license "Apache-2.0"

  on_macos do
    on_intel do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-wireguard-hub-VERSION_PLACEHOLDER-darwin-amd64.tar.gz"
      sha256 "SHA256_DARWIN_AMD64_PLACEHOLDER"
    end

    on_arm do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-wireguard-hub-VERSION_PLACEHOLDER-darwin-arm64.tar.gz"
      sha256 "SHA256_DARWIN_ARM64_PLACEHOLDER"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-wireguard-hub-VERSION_PLACEHOLDER-linux-amd64.tar.gz"
      sha256 "SHA256_LINUX_AMD64_PLACEHOLDER"
    end

    on_arm do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-wireguard-hub-VERSION_PLACEHOLDER-linux-arm64.tar.gz"
      sha256 "SHA256_LINUX_ARM64_PLACEHOLDER"
    end
  end

  depends_on "wireguard-tools" => :recommended

  def install
    bin.install "gatekey-wireguard-hub"
  end

  def caveats
    <<~EOS
      GateKey WireGuard Mesh Hub has been installed.

      The WireGuard mesh hub provides hub-and-spoke VPN topology using WireGuard protocol.

      Requirements:
        1. WireGuard kernel module or wireguard-go
        2. Registration with a GateKey control plane
        3. Configuration file at /etc/gatekey/wireguard-hub.yaml

      Features:
        - Zero-trust access control per network
        - Full tunnel mode support
        - Push DNS to mesh clients
        - Modern WireGuard protocol with better performance

      For production deployments, consider using Docker or Kubernetes.
      See: https://github.com/dye-tech/GateKey/blob/main/docs/wireguard-mesh-networking.md
    EOS
  end

  test do
    assert_match "gatekey-wireguard-hub", shell_output("#{bin}/gatekey-wireguard-hub version 2>&1", 0)
  end
end
