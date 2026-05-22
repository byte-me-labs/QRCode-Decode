FROM gocv/opencv:4.13.0-ubuntu-24.04 AS builder
WORKDIR /build
COPY . .
RUN curl -sL https://go.dev/dl/go1.24.10.linux-amd64.tar.gz | tar -C /usr/local -xzf - \
    && /usr/local/go/bin/go build -o qrdecode .

FROM gocv/opencv:4.13.0-ubuntu-24.04

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /models && \
    curl -sL -o /models/detect.prototxt \
        "https://raw.githubusercontent.com/WeChatCV/opencv_3rdparty/wechat_qrcode/detect.prototxt" && \
    curl -sL -o /models/detect.caffemodel \
        "https://raw.githubusercontent.com/WeChatCV/opencv_3rdparty/wechat_qrcode/detect.caffemodel" && \
    curl -sL -o /models/sr.prototxt \
        "https://raw.githubusercontent.com/WeChatCV/opencv_3rdparty/wechat_qrcode/sr.prototxt" && \
    curl -sL -o /models/sr.caffemodel \
        "https://raw.githubusercontent.com/WeChatCV/opencv_3rdparty/wechat_qrcode/sr.caffemodel"

COPY --from=builder /build/qrdecode /usr/local/bin/qrdecode

ENV PORT=80
EXPOSE 80

WORKDIR /data
ENTRYPOINT ["/usr/local/bin/qrdecode"]
CMD ["serve"]
