class Dirctl < Formula
    desc "Command-line interface for AGNTCY directory"
    homepage "https://github.com/agntcy/dir"
    version "v0.2.5"
    license "Apache-2.0"
    version_scheme 1

    url "https://github.com/agntcy/dir/releases/download/#{version}" # NOTE: It is abused to reduce redundancy

    # TODO: Livecheck can be used to brew bump later

    on_macos do
        if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
            url "#{url}/dirctl-darwin-arm64"
            sha256 "0653543db8c14fdc7335a75d13e262647d5826cc3fda071377a32289d9f5369c"

            def install
                bin.install "dirctl-darwin-arm64" => "dirctl"

                system "chmod", "+x", bin/"dirctl"
                generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
            end
        end

        if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
            url "#{url}/dirctl-darwin-amd64"
            sha256 "b477590e7244fe22ff9a67925e9035d071e55e7325abdfb72902808d6db5dc28"

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
            sha256 "375de409240e64b231a6a67c00c88c550c201bad2e7a5a99adc7e70b7b482d34"

            def install
                bin.install "dirctl-linux-arm64" => "dirctl"

                system "chmod", "+x", bin/"dirctl"
                generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
            end
        end

        if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
            url "#{url}/dirctl-linux-amd64"
            sha256 "5539a0d14972c62c93e246314eebca6f504176e10d3e0d67c0d4980b85698653"

            def install
                bin.install "dirctl-linux-amd64" => "dirctl"

                system "chmod", "+x", bin/"dirctl"
                generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
            end
        end
    end
end
