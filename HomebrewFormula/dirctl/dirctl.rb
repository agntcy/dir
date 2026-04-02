class Dirctl < Formula
  desc "Command-line interface for AGNTCY directory"
  homepage "https://github.com/agntcy/dir"
  version "v1.1.0"
  license "Apache-2.0"
  version_scheme 1

  url "https://github.com/agntcy/dir/releases/download/#{version}"

  on_macos do
      if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
          url "#{url}/dirctl-darwin-arm64"
          sha256 "2afe529383948ead288d95d772225e9d389acee2ea15bbc03091857ea96110d1"

          def install
              bin.install "dirctl-darwin-arm64" => "dirctl"

              system "chmod", "+x", bin/"dirctl"
              generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
          end
      end

      if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
          url "#{url}/dirctl-darwin-amd64"
          sha256 "ed0c51b25d1ba30738762c741712a260a8e3e52a516cee94ad003190dbb5aa13"

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
          sha256 "0c8aee7d5ce721df8d863e17380be42b8b0bc46b94861666a3fac4dd78dd1172"

          def install
              bin.install "dirctl-linux-arm64" => "dirctl"

              system "chmod", "+x", bin/"dirctl"
              generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
          end
      end

      if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
          url "#{url}/dirctl-linux-amd64"
          sha256 "14b0be9229bfc8cdd6a7e4b61ab62d04ec958f96e62b28718f02b0ce7c04ba38"

          def install
              bin.install "dirctl-linux-amd64" => "dirctl"

              system "chmod", "+x", bin/"dirctl"
              generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
          end
      end
  end
end
