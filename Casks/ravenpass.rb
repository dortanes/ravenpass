cask "ravenpass" do
  version "0.2.0"
  sha256 "8ce4754f1ffd61aceba849cb9c7a402a64c68a4ccdc3733daee4cfe615ed975c"

  url "https://github.com/dortanes/ravenpass/releases/download/v#{version}/Ravenpass-#{version}-macos-arm64.dmg"
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
