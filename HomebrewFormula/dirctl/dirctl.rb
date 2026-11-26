class Dirctl < Formula
  desc "Command-line interface for AGNTCY directory"
  homepage "https://github.com/agntcy/dir"
  version "v1.9.2"
  license "Apache-2.0"
  version_scheme 1

  url "https://github.com/agntcy/dir/releases/download/#{version}"

  on_macos do
      if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
          url "#{url}/dirctl-darwin-arm64"
          sha256 "ac1911155915c286f765a7c512440691b5c6287a5d9b6c188ffd64db3c257f86"

          def install
              bin.install "dirctl-darwin-arm64" => "dirctl"

              system "chmod", "+x", bin/"dirctl"
              generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
          end
      end

      if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
          url "#{url}/dirctl-darwin-amd64"
          sha256 "bfc73843f147269b6b2b47bdfb012f89c41fa0d57a33b54991a89ffa00c7c557"

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
          sha256 "c94227224ba43f5703a20fff892517de6e7fdd7d2172761f75ec74318d8cfa6c"

          def install
              bin.install "dirctl-linux-arm64" => "dirctl"

              system "chmod", "+x", bin/"dirctl"
              generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
          end
      end

      if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
          url "#{url}/dirctl-linux-amd64"
          sha256 "9659bf980ad1b27ea355737f945d5ff9097985af4ac2298756e366dfcf7d081c"

          def install
              bin.install "dirctl-linux-amd64" => "dirctl"

              system "chmod", "+x", bin/"dirctl"
              generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
          end
      end
  end
end
