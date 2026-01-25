# typed: false
# frozen_string_literal: true

# Homebrew formula for GateKey Mesh Gateway (Spoke)
# Install with: brew install dye-tech/tap/gatekey-mesh-gateway
class GatekeyMeshGateway < Formula
  desc "Zero Trust OpenVPN mesh spoke for GateKey"
  homepage "https://github.com/dye-tech/GateKey"
  version "VERSION_PLACEHOLDER"
  license "Apache-2.0"

  on_macos do
    on_intel do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-mesh-gateway-VERSION_PLACEHOLDER-darwin-amd64.tar.gz"
      sha256 "SHA256_DARWIN_AMD64_PLACEHOLDER"
    end

    on_arm do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-mesh-gateway-VERSION_PLACEHOLDER-darwin-arm64.tar.gz"
      sha256 "SHA256_DARWIN_ARM64_PLACEHOLDER"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-mesh-gateway-VERSION_PLACEHOLDER-linux-amd64.tar.gz"
      sha256 "SHA256_LINUX_AMD64_PLACEHOLDER"
    end

    on_arm do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-mesh-gateway-VERSION_PLACEHOLDER-linux-arm64.tar.gz"
      sha256 "SHA256_LINUX_ARM64_PLACEHOLDER"
    end
  end

  depends_on "openvpn" => :recommended

  def install
    bin.install "gatekey-mesh-gateway"
  end

  def caveats
    <<~EOS
      GateKey Mesh Gateway (Spoke) has been installed.

      The mesh gateway connects to a mesh hub for site-to-site VPN connectivity.

      Requirements:
        1. OpenVPN client installation
        2. Registration with a GateKey control plane
        3. Configuration file at /etc/gatekey/mesh-gateway.yaml

      Features:
        - Connects remote sites to the mesh hub
        - Zero-trust network access
        - Automatic route propagation

      For production deployments, consider using Docker or Kubernetes.
      See: https://github.com/dye-tech/GateKey/blob/main/docs/mesh-networking.md
    EOS
  end

  test do
    assert_match "gatekey-mesh-gateway", shell_output("#{bin}/gatekey-mesh-gateway version 2>&1", 0)
  end
end
