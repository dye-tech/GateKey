# typed: false
# frozen_string_literal: true

# Homebrew formula for GateKey Server
# Install with: brew install dye-tech/tap/gatekey-server
class GatekeyServer < Formula
  desc "Zero Trust VPN control plane server"
  homepage "https://github.com/dye-tech/GateKey"
  version "VERSION_PLACEHOLDER"
  license "Apache-2.0"

  on_macos do
    on_intel do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-server-VERSION_PLACEHOLDER-darwin-amd64.tar.gz"
      sha256 "SHA256_DARWIN_AMD64_PLACEHOLDER"
    end

    on_arm do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-server-VERSION_PLACEHOLDER-darwin-arm64.tar.gz"
      sha256 "SHA256_DARWIN_ARM64_PLACEHOLDER"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-server-VERSION_PLACEHOLDER-linux-amd64.tar.gz"
      sha256 "SHA256_LINUX_AMD64_PLACEHOLDER"
    end

    on_arm do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-server-VERSION_PLACEHOLDER-linux-arm64.tar.gz"
      sha256 "SHA256_LINUX_ARM64_PLACEHOLDER"
    end
  end

  depends_on "postgresql" => :optional

  def install
    bin.install "gatekey-server"
  end

  def caveats
    <<~EOS
      GateKey Server has been installed.

      Before starting the server, you need:
        1. A PostgreSQL database
        2. Configuration file at ~/.gatekey/server.yaml or /etc/gatekey/server.yaml

      To start the server:
        gatekey-server --config /path/to/config.yaml

      For production deployments, consider using Docker or Kubernetes.
      See: https://github.com/dye-tech/GateKey/tree/main/deploy
    EOS
  end

  test do
    assert_match "gatekey-server", shell_output("#{bin}/gatekey-server version 2>&1", 0)
  end
end
