{
  description = "gowebwrap - thin Wails desktop wrapper around any URL";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs {inherit system;};

      # Wails v3's Linux backend defaults to GTK4 + webkitgtk-6.0; this
      # nixpkgs only ships the older GTK3 + webkitgtk-4.1 ABI, so both the
      # devShell and the package build select that backend via the `gtk3` Go
      # build tag (see wails' internal/assetserver/webview's *_linux_gtk3.go
      # files) instead.
      webkitDeps = pkgs.lib.optionals pkgs.stdenv.hostPlatform.isLinux [
        pkgs.gtk3
        pkgs.webkitgtk_4_1
      ];

      # WebKitGTK's media pipeline (audio/video elements, getUserMedia) goes
      # through GStreamer, not directly through PulseAudio/PipeWire — without
      # these plugins it can't find basic elements like appsink/autoaudiosink
      # ("GStreamer element appsink not found. Please install it.").
      # gst-plugins-base: appsink/audioconvert/playbin etc.
      # gst-plugins-good: pulsesrc/autoaudiosink, the actual PulseAudio/
      # PipeWire (via its Pulse shim) playback/capture elements.
      # gst-plugins-bad: fakevideosink, WebVTT subtitle encoder, and most
      # modern container/codec glue (VP9, Opus, etc).
      # gst-plugins-ugly + gst-libav: real H.264/AAC/MP3 decoders — without
      # these, sites like YouTube fail to play video ("browser can't play
      # this") since their default streams are H.264, and WebKit's
      # GStreamer backend can crash outright (heap corruption) rather than
      # cleanly fail when it negotiates a codec with no matching decoder.
      gstDeps = [
        pkgs.gst_all_1.gstreamer
        pkgs.gst_all_1.gst-plugins-base
        pkgs.gst_all_1.gst-plugins-good
        pkgs.gst_all_1.gst-plugins-bad
        pkgs.gst_all_1.gst-plugins-ugly
        pkgs.gst_all_1.gst-libav
      ];

      # GStreamer doesn't scan Nix store paths by default the way it would
      # FHS system dirs — needs GST_PLUGIN_SYSTEM_PATH_1_0 pointed at each
      # plugin package's lib/gstreamer-1.0 explicitly.
      gstPluginPath = pkgs.lib.makeSearchPathOutput "lib" "lib/gstreamer-1.0" gstDeps;

      # nixpkgs has no wails3 package yet (v3 is alpha) — run the pinned CLI
      # straight from its module cache instead. Version must match go.mod's
      # github.com/wailsapp/wails/v3 requirement.
      wails3Version = "v3.0.0-alpha2.117";
      wails3 = pkgs.writeShellScriptBin "wails3" ''
        exec env GOFLAGS=-tags=gtk3 ${pkgs.go}/bin/go run github.com/wailsapp/wails/v3/cmd/wails3@${wails3Version} "$@"
      '';
    in {
      packages.default = pkgs.buildGoModule {
        pname = "gowebwrap";
        version = "0.0.4";
        src = ./.;

        vendorHash = "sha256-3/QiXWNrQSGvkiEe8zfva9iPHAkeBSJ8kAOdr4IFfoQ=";

        # `go mod vendor` (buildGoModule's default) unconditionally resolves
        # every dependency's go:embed patterns for every GOOS/GOARCH, and
        # Wails v3's alpha releases ship a Windows-only embed
        # (internal/webview2/webviewloader: arm64/WebView2Loader.dll) that's
        # missing from the published module zip — this fails even though
        # we're building for linux and never touch that package. proxyVendor
        # uses `go mod download` instead, which doesn't do that resolution.
        proxyVendor = true;

        # "desktop" and "production" mirror what `wails build` normally
        # passes itself; without them the binary panics at startup with
        # "Wails applications will not build without the correct build tags".
        # "gtk3" selects the GTK3/webkitgtk-4.1 backend — see webkitDeps above.
        tags = ["desktop" "production" "gtk3"];

        nativeBuildInputs = [pkgs.pkg-config pkgs.makeWrapper];
        buildInputs = webkitDeps;

        # Same runtime env the devShell sets (fonts, TLS certs, GStreamer
        # plugins for the webview) — needed here too since `nix run`/an
        # installed binary doesn't go through the devShell's shellHook.
        postFixup = with pkgs; ''
          wrapProgram $out/bin/gowebwrap \
            --suffix XDG_DATA_DIRS : "${gsettings-desktop-schemas}/share/gsettings-schemas/${gsettings-desktop-schemas.name}:${gtk3}/share/gsettings-schemas/${gtk3.name}" \
            --set GIO_EXTRA_MODULES "${glib-networking}/lib/gio/modules" \
            --set SSL_CERT_FILE "${cacert}/etc/ssl/certs/ca-bundle.crt" \
            --set NIX_SSL_CERT_FILE "${cacert}/etc/ssl/certs/ca-bundle.crt" \
            --set GST_PLUGIN_SYSTEM_PATH_1_0 "${gstPluginPath}"
        '';

        meta = {
          description = "gowebwrap desktop app";
          mainProgram = "gowebwrap";
        };
      };

      # For Nix users: wraps the same generic, runtime-config-driven binary
      # from packages.default with a fixed --config so it behaves like a
      # dedicated app — its own store-path binary name, .desktop entry, and
      # icon. Nothing about the wrapped site is baked into the Go binary
      # itself; this only fixes which config file it's pointed at, the same
      # way desktopentry.Install does for non-Nix users (see
      # internal/desktopentry).
      #
      # Usage from a downstream flake:
      #   gowebwrap.lib.${system}.mkApp {
      #     name = "my-site";
      #     config = ./gowebwrap.toml;
      #     icon = ./icon.png; # optional
      #   }
      lib.mkApp = {
        name,
        config,
        icon ? null,
        description ? "",
        categories ? ["Network"],
      }:
        pkgs.stdenv.mkDerivation {
          pname = name;
          version = "0.0.1";
          dontUnpack = true;

          nativeBuildInputs = [pkgs.makeWrapper];

          installPhase =
            ''
              mkdir -p $out/bin $out/share/applications
              makeWrapper ${self.packages.${system}.default}/bin/gowebwrap $out/bin/${name} \
                --add-flags "--config ${config}"
            ''
            + pkgs.lib.optionalString (icon != null) ''
              mkdir -p $out/share/icons/hicolor/512x512/apps
              cp ${icon} $out/share/icons/hicolor/512x512/apps/${name}.png
            ''
            + ''
              cat > $out/share/applications/${name}.desktop <<DESKTOPENTRY
              [Desktop Entry]
              Type=Application
              Name=${name}
              Comment=${description}
              Exec=${name}
              Icon=${
                if icon != null
                then name
                else ""
              }
              Terminal=false
              Categories=${pkgs.lib.concatStringsSep ";" categories};
              DESKTOPENTRY
            '';

          meta = {
            inherit description;
            mainProgram = name;
          };
        };

      devShells.default = pkgs.mkShell {
        buildInputs =
          [
            pkgs.go
            wails3
            pkgs.go-task
            pkgs.pkg-config
            pkgs.gh
          ]
          ++ webkitDeps
          ++ gstDeps;

        # Picked up by plain `go build`/`go run` in this shell. `wails3
        # build`/`wails3 dev` add their own "desktop"/"production"/"dev"
        # tags themselves, so only the webkit backend tag is needed here.
        GOFLAGS = "-tags=gtk3";

        # Without GSettings schemas on XDG_DATA_DIRS, GTK/WebKitGTK fall back
        # to a default font config that renders text tiny/wrong-sized in the
        # webview. GIO_EXTRA_MODULES is needed for glib-networking's TLS gio
        # module to load at all, and SSL_CERT_FILE gives it an actual CA
        # bundle to validate against — NixOS has no /etc/ssl/certs.
        shellHook = with pkgs; ''
          export XDG_DATA_DIRS=${gsettings-desktop-schemas}/share/gsettings-schemas/${gsettings-desktop-schemas.name}:${gtk3}/share/gsettings-schemas/${gtk3.name}:$XDG_DATA_DIRS;
          export GIO_EXTRA_MODULES="${pkgs.glib-networking}/lib/gio/modules";
          export SSL_CERT_FILE="${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt";
          export NIX_SSL_CERT_FILE="${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt";
          export GST_PLUGIN_SYSTEM_PATH_1_0="${gstPluginPath}";
        '';
      };
    });
}
