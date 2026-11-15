cask "ravenpass" do
  version "0.3.0"
  sha256 "1c3e28b69933b685dc1512160a778c83b956746e29a6893d80d852843a3304ba"

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
