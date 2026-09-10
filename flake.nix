{
  description = "A terminal UI for Jenkins";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = f: nixpkgs.lib.genAttrs supportedSystems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      packages = forAllSystems (pkgs: rec {
        jk = pkgs.buildGoModule rec {
          pname = "jk";
          version = "0.0.36";

          src = pkgs.fetchFromGitHub {
            owner = "avivsinai";
            repo = "jenkins-cli";
            rev = "v${version}";
            sha256 = "010c5nbvsxmzq7jhlrs1pz999ncglpi380bsxgyp9pi9q9wskbla";
          };

          vendorHash = "sha256-C+De7EVziBecP7ifXkychDb2h2mDWw3LidiDEaLnagQ=";

          subPackages = [ "cmd/jk" ];

          meta = with pkgs.lib; {
            description = "GitHub-style CLI for Jenkins";
            homepage = "https://github.com/avivsinai/jenkins-cli";
            license = licenses.mit;
          };
        };

        lazyjenkins = pkgs.buildGoModule rec {
          pname = "lazyjenkins";
          version = "dev";

          src = ./.;

          postPatch = ''
            substituteInPlace go.mod --replace "go 1.27.0" "go 1.26"
          '';

          vendorHash = "sha256-yAmydoJZXlipqhZsjojoPA3uoI8BhaU4sPzs9OZ1+3w=";

          nativeBuildInputs = [ pkgs.makeWrapper ];

          postInstall = ''
            wrapProgram $out/bin/lazyjenkins \
              --prefix PATH : ${pkgs.lib.makeBinPath [ jk ]}
          '';

          meta = with pkgs.lib; {
            description = "A terminal UI for Jenkins";
            homepage = "https://github.com/this/repo";
            license = licenses.mit;
          };
        };

        default = lazyjenkins;
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gnumake
            golangci-lint
            self.packages.${pkgs.system}.jk
          ];
        };
      });
    };
}
