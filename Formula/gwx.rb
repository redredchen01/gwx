class Gwx < Formula
  desc "Google Workspace CLI for humans and agents"
  homepage "https://github.com/redredchen01/gwx"
  version "0.7.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/redredchen01/gwx/releases/download/v0.7.0/gwx_0.7.0_darwin_arm64"
      sha256 "" # TODO: fill after release
    end
    on_intel do
      url "https://github.com/redredchen01/gwx/releases/download/v0.7.0/gwx_0.7.0_darwin_amd64"
      sha256 "" # TODO: fill after release
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/redredchen01/gwx/releases/download/v0.7.0/gwx_0.7.0_linux_arm64"
      sha256 "" # TODO: fill after release
    end
    on_intel do
      url "https://github.com/redredchen01/gwx/releases/download/v0.7.0/gwx_0.7.0_linux_amd64"
      sha256 "" # TODO: fill after release
    end
  end

  def install
    binary = Dir["gwx*"].first
    bin.install binary => "gwx"
  end

  test do
    assert_match "gwx", shell_output("#{bin}/gwx version")
  end
end
