class Unified < Formula
  desc "CLI tool for the Unified.to API"
  homepage "https://unified.to"
  license "MIT"

  # Updated by GoReleaser
  url "https://github.com/unified-to/unified-cli/releases/download/v0.0.0/unified_0.0.0_darwin_arm64.tar.gz"
  sha256 "PLACEHOLDER"

  def install
    bin.install "unified"
  end

  test do
    assert_match "unified", shell_output("#{bin}/unified --help")
  end
end
