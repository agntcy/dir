class Dirctl < Formula
  desc "Command-line interface for AGNTCY directory"
  homepage "https://github.com/agntcy/dir"
  version "v1.4.0"
  license "Apache-2.0"
  version_scheme 1

  url "https://github.com/agntcy/dir/releases/download/#{version}"

  on_macos do
      if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
          url "#{url}/dirctl-darwin-arm64"
          sha256 "d46c4104247d7f90bc4a2acea2ad4fb62a65ab2b392870784f1ea6554de31431"

          def install
              bin.install "dirctl-darwin-arm64" => "dirctl"

              system "chmod", "+x", bin/"dirctl"
              ENV["DIRECTORY_CLIENT_SERVER_ADDRESS"] = "127.0.0.1:8888"
              generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
          end
      end

      if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
          url "#{url}/dirctl-darwin-amd64"
          sha256 "4a190bcc4f2eac24ee5b3d00e0ab25efe3954b62c1ed5fb1814ec870dfa65299"

          def install
              bin.install "dirctl-darwin-amd64" => "dirctl"

              system "chmod", "+x", bin/"dirctl"
              ENV["DIRECTORY_CLIENT_SERVER_ADDRESS"] = "127.0.0.1:8888"
              generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
          end
      end
  end

  on_linux do
      if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
          url "#{url}/dirctl-linux-arm64"
          sha256 "069c9ec217ef749acefab5ca31db3ba864b9daf2f673d867e1eefbcce11baae6"

          def install
              bin.install "dirctl-linux-arm64" => "dirctl"

              system "chmod", "+x", bin/"dirctl"
              ENV["DIRECTORY_CLIENT_SERVER_ADDRESS"] = "127.0.0.1:8888"
              generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
          end
      end

      if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
          url "#{url}/dirctl-linux-amd64"
          sha256 "229ec5391032b577cd1056dd67970d92191bc27fb6988300a5d3d9e114226124"

          def install
              bin.install "dirctl-linux-amd64" => "dirctl"

              system "chmod", "+x", bin/"dirctl"
              ENV["DIRECTORY_CLIENT_SERVER_ADDRESS"] = "127.0.0.1:8888"
              generate_completions_from_executable(bin/"dirctl", "completion", shells: [:bash, :zsh, :fish])
          end
      end
  end
end
