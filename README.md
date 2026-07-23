# gowebwrap

A thin, config-driven desktop wrapper for any website — no Electron, no bundled Chromium, no frontend build step.

## What it does

gowebwrap turns any URL into a native desktop app: a real window, its own title/icon, and a real WebKitGTK/WebView2 webview underneath — not a browser tab. Point it at a site via a `gowebwrap.toml` config file or CLI flags, and it launches that site as if it were a dedicated app. Everything is resolved at runtime, so the same generic binary works for any site; nothing about a given site is baked into the binary itself.

## Why

Electron and Tauri (and Wails itself, used directly) all assume you're shipping a bundled frontend and are willing to build a whole app around it. If you just want an existing website to feel like a native app — its own window, its own icon, no browser chrome — that's a lot of machinery for not much:

- **No bundled Chromium.** Wails uses the OS's own webview (WebKitGTK on Linux, WebView2 on Windows), so binaries are megabytes, not hundreds of megabytes.
- **No frontend build step.** There's no frontend to build — gowebwrap just points a webview at a URL you already have.
- **No forking required.** One generic binary, driven by a config file or flags. Shipping your own site as a "real app" (its own name/icon/launcher) is a packaging step, not a code change.

## Install / Quick start

### Try it immediately (Nix, no install)

```sh
nix run github:jim-ww/gowebwrap -- --url=https://example.com
```

### As a package in your own flake

```nix
{
  inputs.gowebwrap.url = "github:jim-ww/gowebwrap";
  # then reference inputs.gowebwrap.packages.${system}.default
}
```

To wrap your own site with its own name/icon, use `lib.mkApp` instead — see [Shipping it as a real app](#shipping-it-as-a-real-app) below.

### Without Nix

Grab a prebuilt binary for your platform from the [Releases](https://github.com/jim-ww/gowebwrap/releases) page.

## Configuration

gowebwrap is driven entirely by a `gowebwrap.toml` file, CLI flags, or both — flags always win over the config file.

```toml
# gowebwrap.toml
name = "My Site"
description = "My site, as an app"
url = "https://example.com"
window-width = 1200
window-height = 800
```

```sh
gowebwrap --url=https://example.com --window-width=1200
```

If `--config` isn't given, gowebwrap looks for `./gowebwrap.toml` in the current directory. Run `gowebwrap --help` for the full list of flags (window geometry, background color, dark theme, GPU acceleration, User-Agent override, etc.) — every flag has a matching config key of the same name.

## Shipping it as a real app

A generic `gowebwrap` process running your config already works, but you can also give it its own launcher — its own name, icon, and entry in your app menu — instead of running it from a terminal.

### Nix

```nix
{
  inputs.gowebwrap.url = "github:jim-ww/gowebwrap";

  outputs = { self, nixpkgs, gowebwrap, ... }: let
    system = "x86_64-linux";
  in {
    packages.${system}.default = gowebwrap.lib.${system}.mkApp {
      name = "my-site";
      config = ./gowebwrap.toml;
      icon = ./icon.png; # optional
    };
  };
}
```

`nix build`/`nix run` on this now gives you a `my-site` binary, complete with a `.desktop` launcher and icon — no separate gowebwrap install step needed.

### Non-Nix

```sh
gowebwrap --config gowebwrap.toml install
```

This writes a canonical config to `~/.config/gowebwrap/<name>.toml` and a `.desktop` launcher to `~/.local/share/applications/`, so the app shows up in your normal application launcher/menu like anything else you installed.

## Build from source

With Nix (recommended — handles all webview/GStreamer dependencies for you):

```sh
nix build github:jim-ww/gowebwrap
./result/bin/gowebwrap --url=https://example.com
```

Without Nix, on Linux, you'll need `pkg-config`, GTK3, and webkitgtk-4.1 dev packages installed, then:

```sh
go build -tags "desktop production gtk3" -o gowebwrap .
```

On Windows, no cgo/extra dependencies are needed — Wails' WebView2 backend calls it via syscalls, so it cross-compiles from Linux too:

```sh
GOOS=windows GOARCH=amd64 go build -tags "desktop production" -ldflags="-H=windowsgui" -o gowebwrap.exe .
```

## Support the project

If gowebwrap is useful to you, consider a small donation.

**Monero (XMR)**
```
83YGRqP8uHed6NeegZQeX9ccCxbzoRHHEEi7pTwk4aqdJZEVXXA6NWtetnsEM2v33zFBBt3Rp6DNhU9qhJEGPspU14yN8t7
```

## License

gowebwrap is free software, licensed under the MIT license — see [LICENSE](LICENSE).
