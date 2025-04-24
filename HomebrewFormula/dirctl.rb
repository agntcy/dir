class Dirctl < Formula
    desc "Command-line interface for AGNTCY directory"
    homepage "https://github.com/agntcy/dir"
    version "v0.2.1"
    license "Apache-2.0"
    version_scheme 1

    url "https://github.com/agntcy/dir/releases/download/#{version}" # NOTE: It is abused to reduce redundancy

    option "with-hub", "CLI with Hub extension"

    # TODO: Livecheck can be used to brew bump later

    on_macos do
        if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
            if build.with? "hub"
                url "#{url}/dirctl-hub-darwin-arm64"
                sha256 "0d2d2a74059ce348ba2b2e0f7328246ab0c6f7c98f2aefb396efdf1eb6779f08"
            else
                url "#{url}/dirctl-darwin-arm64"
                sha256 "7de6a450f7c57e320da5f759fd616982138231db26402f775b4f1a1caa49e9d5"
            end

            def install
                if build.with? "hub"
                    bin.install "dirctl-hub-darwin-arm64" => "dirctl"
                else
                    bin.install "dirctl-darwin-arm64" => "dirctl"
                end

                system "chmod", "+x", bin/"dirctl"
                generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
            end
        end

        if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
            if build.with? "hub"
                url "#{url}/dirctl-hub-darwin-amd64"
                sha256 "0a356a09102cd18bba193a8835b0a562f8786be32c4998c87505c2ab56450f77"
            else
                url "#{url}/dirctl-darwin-amd64"
                sha256 "0097077942494c75831dc7272d715c526a24e6699c6c97ab8e5c945664235788"
            end

            def install
                if build.with? "hub"
                    bin.install "dirctl-hub-darwin-amd64" => "dirctl"
                else
                    bin.install "dirctl-darwin-amd64" => "dirctl"
                end

                system "chmod", "+x", bin/"dirctl"
                generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
            end
        end
    end

    on_linux do
        if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
            if build.with? "hub"
                url "#{url}/dirctl-hub-linux-arm64"
                sha256 "8cf32863438315910c680b27e63c315eda4987a84ffe356ea741d723b0a0a4f3"
            else
                url "#{url}/dirctl-linux-arm64"
                sha256 "c58d3712bc57c63b251b70a2e9f7866dad058cf490a3c210455b68026fc5eb7c"
            end
            def install
                if build.with? "hub"
                    bin.install "dirctl-hub-linux-arm64" => "dirctl"
                else
                    bin.install "dirctl-linux-arm64" => "dirctl"
                end

                system "chmod", "+x", bin/"dirctl"
                generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
            end
        end

        if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
            if build.with? "hub"
                url "#{url}/dirctl-hub-linux-amd64"
                sha256 "80d74b602c9f724c444a4ef28d5460a2108a3f20711b425e293db0369045f015"
            else
                url "#{url}/dirctl-linux-amd64"
                sha256 "fefdd4d705c654719cacc0d4f687b1d1e2e726c7340d04b28623b3bad75411cb"
            end

            def install
                if build.with? "hub"
                    bin.install "dirctl-hub-linux-amd64" => "dirctl"
                else
                    bin.install "dirctl-linux-amd64" => "dirctl"
                end

                system "chmod", "+x", bin/"dirctl"
                generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
            end
        end
    end
end
