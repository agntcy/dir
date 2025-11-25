class Dirctl < Formula
    desc "Command-line interface for AGNTCY directory"
    homepage "https://github.com/agntcy/dir"
    version "v0.5.3"
    license "Apache-2.0"
    version_scheme 1

    url "https://github.com/agntcy/dir/releases/download/#{version}" # NOTE: It is abused to reduce redundancy

    # TODO: Livecheck can be used to brew bump later

    on_macos do
        if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
            url "#{url}/dirctl-darwin-arm64"
            sha256 "04eb6362f711f0fe8645e099a60cd65690d28da6c71f30ecc3f9613fd629dfd0"

            def install
                bin.install "dirctl-darwin-arm64" => "dirctl"

                system "chmod", "+x", bin/"dirctl"
                generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
            end
        end

        if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
            url "#{url}/dirctl-darwin-amd64"
            sha256 "ff786f7360c512cf2b70e6efe2ed456f6d9d51e9f29494b29a9655663e6ce1e3"

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
            sha256 "24a7db372edb474efe45094ce7d4fccc3f19ce75b47243cfd91fb094d043c141"

            def install
                bin.install "dirctl-linux-arm64" => "dirctl"

                system "chmod", "+x", bin/"dirctl"
                generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
            end
        end

        if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
            url "#{url}/dirctl-linux-amd64"
            sha256 "b591ee011f6dc68ae1a8274501cef05060c50d91497afbe6d6bca4a0fd7c8f13"

            def install
                bin.install "dirctl-linux-amd64" => "dirctl"

                system "chmod", "+x", bin/"dirctl"
                generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
            end
        end
    end
end
