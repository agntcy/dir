class DirctlAT010 < Formula
  desc "Command-line interface for AGNTCY directory"
  homepage "https://github.com/agntcy/dir"
  version "v0.1.0"
  license "Apache-2.0"
  version_scheme 1

  url "https://github.com/agntcy/dir/releases/download/#{version}"

  on_macos do
      if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
          url "#{url}/dirctl-darwin-arm64"
          sha256 "0c0dda6b76bbeeea9db622e72407fbb14a0394e8374a312e9183b2df50c18703"

          def install
              bin.install "dirctl-darwin-arm64" => "dirctl"

              system "chmod", "+x", bin/"dirctl"
              generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
          end
      end

      if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
          url "#{url}/dirctl-darwin-amd64"
          sha256 "4b0fde01a7b10c7a7331dfeb3edcde2c44ba02e068591ff861e42d12dfba153b"

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
          sha256 "662f19bcb528c7eeb447166706403db4ab85c0eddda90ac2a190d1d608cef346"

          def install
              bin.install "dirctl-linux-arm64" => "dirctl"

              system "chmod", "+x", bin/"dirctl"
              generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
          end
      end

      if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
          url "#{url}/dirctl-linux-amd64"
          sha256 "e622de536104c20fc28ef224dd7ffdfbf724f90944437c5a099aa9122d50dfd8"

          def install
              bin.install "dirctl-linux-amd64" => "dirctl"

              system "chmod", "+x", bin/"dirctl"
              generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
          end
      end
  end
end
