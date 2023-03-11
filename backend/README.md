```bash
go mod tidy

go install \
    github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
    github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2 \
    google.golang.org/protobuf/cmd/protoc-gen-go \
    google.golang.org/grpc/cmd/protoc-gen-go-grpc

./scripts/build_proto.sh

# 輸出內容填到 server-data/configs/user.yaml 的 cookiekeypairs 欄位
openssl rand -base64 128

openssl rand -out server-data/srvckey -base64 128
```

TODO:

- 登入資訊紀錄(時間、IP、裝置資訊)
- User 對 縮網址(Link)的 CRUD
- 跳轉廣告過場
- User 對 Link 的增添限制(免費、付費... ...)
- Auth secure cookie key
- Link 來源分析(裝置、IP 國家)
- Link 的 dest 修改(要算多使用一次額度)
- 服務間的溝通金鑰
- 支援 QR Code
- 支援自定義域名
- 管理員操作 LOG
- API 的 Unit Test
