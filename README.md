# QRCode-Decode

基于 OpenCV WeChatQRCode 的二维码解码工具，支持命令行和 HTTP 服务两种方式。

[English](README.en.md)

## 构建（Docker）

```bash
docker build -t qrdecode .
```

## 使用

### 命令行解码

```bash
# 解码本地图片
docker run --rm -v /path/to/images:/data qrdecode decode /data/image.png

# 自定义模型路径
docker run --rm qrdecode decode --detect-proto /custom/detect.prototxt /data/image.png
```

### HTTP 服务

```bash
# 默认端口 80
docker run --rm -p 80:80 qrdecode serve

# 通过环境变量指定端口
docker run --rm -p 9090:9090 -e PORT=9090 qrdecode serve

# 通过命令行参数指定端口
docker run --rm -p 9090:9090 qrdecode serve --port 9090
```

#### 接口

```
POST /decode
Content-Type: multipart/form-data
字段: image

成功响应 200: {"content": "解码内容"}
错误响应 400: {"error": "..."}
错误响应 422: {"error": "..."}
```

```bash
curl -F "image=@qrcode.png" http://localhost:80/decode
```

## 环境变量

| 变量   | 默认值 | 说明          |
|--------|--------|---------------|
| `PORT` | 80     | HTTP 服务端口 |

## 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--port` | 80 | HTTP 服务端口（优先级高于 `PORT` 环境变量） |
| `--detect-proto` | /models/detect.prototxt | WeChatQRCode 检测模型配置 |
| `--detect-model` | /models/detect.caffemodel | WeChatQRCode 检测模型权重 |
| `--sr-proto` | /models/sr.prototxt | WeChatQRCode 超分模型配置 |
| `--sr-model` | /models/sr.caffemodel | WeChatQRCode 超分模型权重 |
