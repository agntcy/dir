class DirctlAT015 < Formula
  desc "Command-line interface for AGNTCY directory"
  homepage "https://github.com/agntcy/dir"
  version "v0.1.5"
  license "Apache-2.0"
  version_scheme 1

  url "https://github.com/agntcy/dir/releases/download/#{version}"

  on_macos do
      if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
          url "#{url}/dirctl-darwin-arm64"
          sha256 "d481ac5a8a0ba41d9ca32d610789b6e4bfde787948e64534a0bc9198451ea418"

          def install
              bin.install "dirctl-darwin-arm64" => "dirctl"

              system "chmod", "+x", bin/"dirctl"
              generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
          end
      end

      if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
          url "#{url}/dirctl-darwin-amd64"
          sha256 "e6cf264722ed5efbab531cbe593c3464909a7d265fc09efe3c30d278229b1fa2"

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
          sha256 "3c41bd427fbffdd4b47375f0398d36bf832e6976b2c48e9d0407f590d42212b4"

          def install
              bin.install "dirctl-linux-arm64" => "dirctl"

              system "chmod", "+x", bin/"dirctl"
              generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
          end
      end

      if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
          url "#{url}/dirctl-linux-amd64"
          sha256 "51719e35fbb9e73d000acbdfcc3973fc6197aa86c6a5bcf81f6b0a5a0509a7b2"

          def install
              bin.install "dirctl-linux-amd64" => "dirctl"

              system "chmod", "+x", bin/"dirctl"
              generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
          end
      end
  end
end
