# TinyParrot

Super small Docker image that serves a single HTTP endpoint returning pre-configured text.

## Run

```sh
docker run --rm -p 8080:8080 -e TINYPARROT_TEXT='hello from TinyParrot' ghcr.io/sygmei/tinyparrot
curl http://localhost:8080/
```

The image is a static Go binary in `scratch`, so it has no shell or package manager in the final runtime image.

## Configuration

TinyParrot listens on port `8080` and serves `/` by default.

| Variable | Default | Description |
| --- | --- | --- |
| `TINYPARROT_TEXT` | `TinyParrot\n` | Text returned by the endpoint. |
| `TINYPARROT_TEXT_FILE` | unset | Path to a file whose contents are returned. Takes precedence over `TINYPARROT_TEXT`. |
| `TINYPARROT_PATH` | `/` | HTTP path to serve. A leading `/` is added if omitted. |
| `TINYPARROT_ADDR` | `:8080` | Listen address. |
| `PORT` | unset | Alternate port-style value, used only when `TINYPARROT_ADDR` is unset. |

You can also mount a file at `/etc/tinyparrot/text`; if neither `TINYPARROT_TEXT_FILE` nor `TINYPARROT_TEXT` is set, TinyParrot will use that file when it exists.

```sh
printf 'hello from a file\n' > ./message.txt
docker run --rm -p 8080:8080 \
  -v "$PWD/message.txt:/etc/tinyparrot/text:ro" \
  ghcr.io/sygmei/tinyparrot
```

For an explicit file path:

```sh
docker run --rm -p 8080:8080 \
  -v "$PWD/message.txt:/message.txt:ro" \
  -e TINYPARROT_TEXT_FILE=/message.txt \
  ghcr.io/sygmei/tinyparrot
```

## Build

```sh
docker build -t tinyparrot .
```

## Publishing

The GitHub workflow builds and publishes `linux/amd64` and `linux/arm64` images to GHCR when a tag is pushed:

- `ghcr.io/sygmei/tinyparrot`
