# QRCode-Decode

QR code decoder based on OpenCV WeChatQRCode, with CLI and HTTP server support.

## Build (Docker)

```bash
docker build -t qrdecode .
```

## Usage

### CLI decode

```bash
# decode from local file
docker run --rm -v /path/to/images:/data qrdecode decode /data/image.png

# custom model paths
docker run --rm qrdecode decode --detect-proto /custom/detect.prototxt /data/image.png
```

### HTTP server

```bash
# default port 80
docker run --rm -p 80:80 qrdecode serve

# custom port via env var
docker run --rm -p 9090:9090 -e PORT=9090 qrdecode serve

# custom port via flag
docker run --rm -p 9090:9090 qrdecode serve --port 9090
```

#### API

```
POST /decode
Content-Type: multipart/form-data
Field: image

Response 200: {"content": "decoded text"}
Response 400: {"error": "..."}
Response 422: {"error": "..."}
```

```bash
curl -F "image=@qrcode.png" http://localhost:80/decode
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT`   | 80    | HTTP server port |

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | 80 | HTTP server port (overrides `PORT` env) |
| `--detect-proto` | /models/detect.prototxt | WeChatQRCode detect prototxt |
| `--detect-model` | /models/detect.caffemodel | WeChatQRCode detect caffemodel |
| `--sr-proto` | /models/sr.prototxt | WeChatQRCode super-resolution prototxt |
| `--sr-model` | /models/sr.caffemodel | WeChatQRCode super-resolution caffemodel |
