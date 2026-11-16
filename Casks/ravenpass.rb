cask "ravenpass" do
  version "0.4.0"
  sha256 "af0bab29e7cf976dc80dec6d17dcf707cdd79cbb3605a4d1a903f6c74b5752b2"

  url "https://github.com/dortanes/ravenpass/releases/download/v#{version}/Ravenpass-macos-arm64.dmg"
  name "Ravenpass"
  desc "Password manager with local vaults, passkeys, and TOTP codes"
  homepage "https://ravenpass.org/"

  livecheck do
    url :url
    strategy :github_latest
  end

  depends_on arch: :arm64
  depends_on macos: :ventura

  app "Ravenpass.app"

  caveats <<~EOS
    Ravenpass is currently in beta. Keep backups of your vault.
    System AutoFill requires macOS 15 or later.
  EOS
end
