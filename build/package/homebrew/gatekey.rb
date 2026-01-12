# typed: false
# frozen_string_literal: true

# Homebrew formula for GateKey VPN client
# Install with: brew install dye-tech/tap/gatekey
class Gatekey < Formula
  desc "Zero Trust VPN client for GateKey"
  homepage "https://github.com/dye-tech/GateKey"
  version "VERSION_PLACEHOLDER"
  license "Apache-2.0"

  on_macos do
    on_intel do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-VERSION_PLACEHOLDER-darwin-amd64.tar.gz"
      sha256 "SHA256_DARWIN_AMD64_PLACEHOLDER"
    end

    on_arm do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-VERSION_PLACEHOLDER-darwin-arm64.tar.gz"
      sha256 "SHA256_DARWIN_ARM64_PLACEHOLDER"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-VERSION_PLACEHOLDER-linux-amd64.tar.gz"
      sha256 "SHA256_LINUX_AMD64_PLACEHOLDER"
    end

    on_arm do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-VERSION_PLACEHOLDER-linux-arm64.tar.gz"
      sha256 "SHA256_LINUX_ARM64_PLACEHOLDER"
    end
  end

  def install
    bin.install "gatekey"
  end

  def caveats
    <<~EOS
      GateKey VPN client has been installed.

      To get started:
        1. Initialize configuration:
           gatekey config init --server https://your-gatekey-server.com

        2. Log in:
           gatekey login

        3. Connect to VPN:
           gatekey connect

      For more information, see:
        https://github.com/dye-tech/GateKey
    EOS
  end

  test do
    assert_match "gatekey", shell_output("#{bin}/gatekey version 2>&1", 0)
  end
end
