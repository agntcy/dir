class Dirctl < Formula
    desc "Command-line interface for AGNTCY directory"
    homepage "https://github.com/agntcy/dir"
    version "v0.2.2"
    license "Apache-2.0"
    version_scheme 1

    url "https://github.com/agntcy/dir/releases/download/#{version}" # NOTE: It is abused to reduce redundancy

    option "with-hub", "CLI with Hub extension"

    # TODO: Livecheck can be used to brew bump later

    on_macos do
        if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
            if build.with? "hub"
                url "#{url}/dirctl-hub-darwin-arm64"
                sha256 ""
            else
                url "#{url}/dirctl-darwin-arm64"
                sha256 "d505890f633f8adc0d6b0b398912ddabf5c4cee66f68c2b0faffe1d048c6b931"
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
                sha256 ""
            else
                url "#{url}/dirctl-darwin-amd64"
                sha256 "3dea803513f326b0ad916f4fa985f8244ca5b9e30723a44d51428f3e42703dcf"
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
                sha256 ""
            else
                url "#{url}/dirctl-linux-arm64"
                sha256 "1bd9a1d81877b0d459ecb4275cd14e7895f60a50d3ad376937d210e914a7d07a"
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
                sha256 ""
            else
                url "#{url}/dirctl-linux-amd64"
                sha256 "ee9929600630a788cb2292c76a60b8e97786f2e9c91132ea952aba2923dc2e5a"
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
