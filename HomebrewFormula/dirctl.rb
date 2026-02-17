class Dirctl < Formula
    desc "Command-line interface for AGNTCY directory"
    homepage "https://github.com/agntcy/dir"
    version "v1.0.0-rc.4"
    license "Apache-2.0"
    version_scheme 1

    url "https://github.com/agntcy/dir/releases/download/#{version}" # NOTE: It is abused to reduce redundancy

    # TODO: Livecheck can be used to brew bump later

    on_macos do
        if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
            url "#{url}/dirctl-darwin-arm64"
            sha256 "541b49e794883cac455a136fa8c638131f630c52b9148f1f39b65ce0263b2227"

            def install
                bin.install "dirctl-darwin-arm64" => "dirctl"

                system "chmod", "+x", bin/"dirctl"
                generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
            end
        end

        if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
            url "#{url}/dirctl-darwin-amd64"
            sha256 "aabe04c98c5bcacc0bbf718b9548b905be37c06559991570f9eea24c311e553f"

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
            sha256 "e928eccf83dc82785b93c95837d91b99242c7b658408d1e469b74aa5b69e482f"

            def install
                bin.install "dirctl-linux-arm64" => "dirctl"

                system "chmod", "+x", bin/"dirctl"
                generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
            end
        end

        if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
            url "#{url}/dirctl-linux-amd64"
            sha256 "2242ee15f181c3aa46070714a81d8be8e7242c9c6f0320449c7a1d963ed2cffd"

            def install
                bin.install "dirctl-linux-amd64" => "dirctl"

                system "chmod", "+x", bin/"dirctl"
                generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
            end
        end
    end
end
