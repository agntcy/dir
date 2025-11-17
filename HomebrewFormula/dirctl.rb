class Dirctl < Formula
    desc "Command-line interface for AGNTCY directory"
    homepage "https://github.com/agntcy/dir"
    version "v0.5.1"
    license "Apache-2.0"
    version_scheme 1

    url "https://github.com/agntcy/dir/releases/download/#{version}" # NOTE: It is abused to reduce redundancy

    # TODO: Livecheck can be used to brew bump later

    on_macos do
        if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
            url "#{url}/dirctl-darwin-arm64"
            sha256 "e9f4dcce5be455aa608ea30cb1362d695569ca0abc37b7df94aef866c43d5bec"

            def install
                bin.install "dirctl-darwin-arm64" => "dirctl"

                system "chmod", "+x", bin/"dirctl"
                generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
            end
        end

        if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
            url "#{url}/dirctl-darwin-amd64"
            sha256 "eb8a71a904c0af7f106fc09fbb0151f352b206899c9b888bbe3e6e343135037f"

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
            sha256 "c076a03cd522710d2c49731a32cb0f224b60cc58dd44a027eb7621600988d826"

            def install
                bin.install "dirctl-linux-arm64" => "dirctl"

                system "chmod", "+x", bin/"dirctl"
                generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
            end
        end

        if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
            url "#{url}/dirctl-linux-amd64"
            sha256 "310a0a56eb61dd8b369d5ef0832fbd241a53c3d0146cf814f82cd51d92a96aef"

            def install
                bin.install "dirctl-linux-amd64" => "dirctl"

                system "chmod", "+x", bin/"dirctl"
                generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
            end
        end
    end
end
