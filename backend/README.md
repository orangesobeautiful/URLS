# URLS 後端

目前透過以下 service 組成:

- Gateway: 將流量發送到特定 service
- User: 處理使用者相關資料與登入功能
- Link:
  - link: 處理短網址的資料
  - redirector: 短網址導向

## 開發前置工作

protobuf 相關工具安裝

```bash
scripts/install_proto_tool.sh
```

建構 protobuf

```bash
scripts/build_proto.sh
```

Cookie Secure Key 生成與設定

透過 openssl 生成隨機字串並將其寫入 server-data/configs/user.yaml 的 cookiekeypairs 欄位

```bash
openssl rand -base64 128
```

生成 Service 內部溝通金鑰

```bash
openssl rand -out server-data/srvckey -base64 128
```
