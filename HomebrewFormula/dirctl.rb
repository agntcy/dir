class Dirctl < Formula
    desc "Command-line interface for AGNTCY directory"
    homepage "https://github.com/agntcy/dir"
    version "v0.2.3"
    license "Apache-2.0"
    version_scheme 1

    url "https://github.com/agntcy/dir/releases/download/#{version}" # NOTE: It is abused to reduce redundancy

    # TODO: Livecheck can be used to brew bump later

    on_macos do
        if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
            url "#{url}/dirctl-darwin-arm64"
            sha256 "d8e8a2e8d64d64fe2b013d14f5c756ee8a7dbb4d3b063bb997c7d69d25d0c876"

            def install
                bin.install "dirctl-darwin-arm64" => "dirctl"

                system "chmod", "+x", bin/"dirctl"
                generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
            end
        end

        if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
            url "#{url}/dirctl-darwin-amd64"
            sha256 "4140a7e8748bf125509678b310f4d0ce0c0ac726b8c20ace683c63617b56f5d9"

            def install
                bin.install "dirctl-darwin-amd64" => "dirctl"

                system "chmod", "+x", bin/"dirctl"
                generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
            end
        end
    end

    on_linux do
        if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
            url "#{url}/dirctl-linux-arm64"
            sha256 "b4eec1661962c5ea69bfc94f751c8471f8307c5e7df0a5d4d88c22ea03f908dc"

            def install
                bin.install "dirctl-linux-arm64" => "dirctl"

                system "chmod", "+x", bin/"dirctl"
                generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
            end
        end

        if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
            url "#{url}/dirctl-linux-amd64"
            sha256 "af4e7023438475519368f89bc255f683cef9db82a4bcc1bdfca4d4c0ba112c94"

            def install
                bin.install "dirctl-linux-amd64" => "dirctl"

                system "chmod", "+x", bin/"dirctl"
                generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
            end
        end
    end
end
