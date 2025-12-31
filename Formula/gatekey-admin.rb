# typed: false
# frozen_string_literal: true

# Homebrew formula for GateKey Admin CLI
# Install with: brew install dye-tech/tap/gatekey-admin
class GatekeyAdmin < Formula
  desc "Admin CLI tool for GateKey VPN management"
  homepage "https://github.com/dye-tech/GateKey"
  version "VERSION_PLACEHOLDER"
  license "Apache-2.0"

  on_macos do
    on_intel do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-admin-VERSION_PLACEHOLDER-darwin-amd64.tar.gz"
      sha256 "SHA256_DARWIN_AMD64_PLACEHOLDER"
    end

    on_arm do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-admin-VERSION_PLACEHOLDER-darwin-arm64.tar.gz"
      sha256 "SHA256_DARWIN_ARM64_PLACEHOLDER"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-admin-VERSION_PLACEHOLDER-linux-amd64.tar.gz"
      sha256 "SHA256_LINUX_AMD64_PLACEHOLDER"
    end

    on_arm do
      url "https://github.com/dye-tech/GateKey/releases/download/vVERSION_PLACEHOLDER/gatekey-admin-VERSION_PLACEHOLDER-linux-arm64.tar.gz"
      sha256 "SHA256_LINUX_ARM64_PLACEHOLDER"
    end
  end

  def install
    bin.install "gatekey-admin"
  end

  def caveats
    <<~EOS
      GateKey Admin CLI has been installed.

      This tool is used for administrative tasks such as:
        - Managing users and groups
        - Configuring gateways
        - Managing access rules
        - Viewing audit logs

      To get started:
        gatekey-admin --help

      See: https://github.com/dye-tech/GateKey
    EOS
  end

  test do
    assert_match "gatekey-admin", shell_output("#{bin}/gatekey-admin version 2>&1", 0)
  end
end
