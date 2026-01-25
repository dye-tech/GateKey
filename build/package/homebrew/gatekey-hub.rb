# typed: false
# frozen_string_literal: true

# Homebrew formula for GateKey Mesh Hub
# Install with: brew install dye-tech/tap/gatekey-hub
class GatekeyHub < Formula
  desc "Zero Trust OpenVPN mesh hub for GateKey"
  homepage "https://github.com/dye-tech/GateKey"
  version "VERSION_PLACEHOLDER"
  license "Apache-2.0"

  on_macos do
    on_intel do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-hub-VERSION_PLACEHOLDER-darwin-amd64.tar.gz"
      sha256 "SHA256_DARWIN_AMD64_PLACEHOLDER"
    end

    on_arm do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-hub-VERSION_PLACEHOLDER-darwin-arm64.tar.gz"
      sha256 "SHA256_DARWIN_ARM64_PLACEHOLDER"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-hub-VERSION_PLACEHOLDER-linux-amd64.tar.gz"
      sha256 "SHA256_LINUX_AMD64_PLACEHOLDER"
    end

    on_arm do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-hub-VERSION_PLACEHOLDER-linux-arm64.tar.gz"
      sha256 "SHA256_LINUX_ARM64_PLACEHOLDER"
    end
  end

  depends_on "openvpn" => :recommended

  def install
    bin.install "gatekey-hub"
  end

  def caveats
    <<~EOS
      GateKey Mesh Hub has been installed.

      The mesh hub provides hub-and-spoke VPN topology for site-to-site connectivity.

      Requirements:
        1. OpenVPN server installation
        2. Registration with a GateKey control plane
        3. Configuration file at /etc/gatekey/hub.yaml

      Features:
        - Zero-trust access control per network
        - Full tunnel mode support
        - Push DNS to mesh clients

      For production deployments, consider using Docker or Kubernetes.
      See: https://github.com/dye-tech/GateKey/blob/main/docs/mesh-networking.md
    EOS
  end

  test do
    assert_match "gatekey-hub", shell_output("#{bin}/gatekey-hub version 2>&1", 0)
  end
end
