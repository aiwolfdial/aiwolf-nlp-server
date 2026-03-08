# aiwolf-nlp-server

[README in Japanese](/README.md)

This is a game server for the AIWolf Contest (Natural Language Division).

For sample agents, please refer to [aiwolfdial/aiwolf-nlp-agent](https://github.com/aiwolfdial/aiwolf-nlp-agent).

## Documentation

- [Configuration File](/doc/en/config.md)
- [Game Logic Implementation](/doc/en/logic.md)
- [Protocol Implementation](/doc/en/protocol.md)

## How to Run

The default server address is `ws://127.0.0.1:8080/ws`. Please specify this address as the connection destination for your agent program.
The self-play mode, which matches only agents with the same team name, is enabled by default. Therefore, if you want to match agents with different team names, please modify the configuration file.
For information on how to modify the configuration file, please refer to [Configuration File](/doc/en/config.md).

### Linux

```bash
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/aiwolf-nlp-server-linux-amd64
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_5.yml
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_9.yml
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_13.yml
curl -Lo .env https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/example.env
chmod u+x ./aiwolf-nlp-server-linux-amd64
./aiwolf-nlp-server-linux-amd64 -c ./default_5.yml # For 5-player games
# ./aiwolf-nlp-server-linux-amd64 -c ./default_9.yml # For 9-player games
# ./aiwolf-nlp-server-linux-amd64 -c ./default_13.yml # For 13-player games
```

### Windows

```bash
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/aiwolf-nlp-server-windows-amd64.exe
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_5.yml
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_9.yml
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_13.yml
curl -Lo .env https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/example.env
.\aiwolf-nlp-server-windows-amd64.exe -c .\default_5.yml # For 5-player games
# .\aiwolf-nlp-server-windows-amd64.exe -c .\default_9.yml # For 9-player games
# .\aiwolf-nlp-server-windows-amd64.exe -c .\default_13.yml # For 13-player games
```

### macOS (Intel)

> [!NOTE]
> The application may be blocked as an unknown developer app.
> Please refer to the following site to grant execution permission:
> <https://support.apple.com/guide/mac-help/mh40616/mac>

```bash
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/aiwolf-nlp-server-darwin-amd64
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_5.yml
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_9.yml
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_13.yml
curl -Lo .env https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/example.env
chmod u+x ./aiwolf-nlp-server-darwin-amd64
./aiwolf-nlp-server-darwin-amd64 -c ./default_5.yml # For 5-player games
# ./aiwolf-nlp-server-darwin-amd64 -c ./default_9.yml # For 9-player games
# ./aiwolf-nlp-server-darwin-amd64 -c ./default_13.yml # For 13-player games
```

### macOS (Apple Silicon)

> [!NOTE]
> The application may be blocked as an unknown developer app.
> Please refer to the following site to grant execution permission:
> <https://support.apple.com/guide/mac-help/mh40616/mac>

```bash
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/aiwolf-nlp-server-darwin-arm64
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_5.yml
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_9.yml
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_13.yml
curl -Lo .env https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/example.env
chmod u+x ./aiwolf-nlp-server-darwin-arm64
./aiwolf-nlp-server-darwin-arm64 -c ./default_5.yml # For 5-player games
# ./aiwolf-nlp-server-darwin-arm64 -c ./default_9.yml # For 9-player games
# ./aiwolf-nlp-server-darwin-arm64 -c ./default_13.yml # For 13-player games
```
