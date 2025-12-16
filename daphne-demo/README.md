# Daphne Django Demo Project

演示如何使用 Daphne 作为 ASGI 服务器来运行 Django 应用。

## 快速开始

### 1. 环境准备

确保你已经安装了 Python (建议 3.8+)。

### 2. 安装依赖

在项目根目录下运行以下命令安装所需的 Python 包：

```bash
pip install -r requirements.txt
```

### 3. 初始化数据库

虽然本示例仅包含简单的接口，但为了 Django 正常运行（如 Session/Auth 等中间件），建议运行数据库迁移：

```bash
python manage.py migrate
```

### 4. 启动服务

使用 `daphne` 命令启动 ASGI 服务，监听 8005 端口：

```bash
daphne -p 8005 demo_project.asgi:application
```

启动成功后，你会看到类似以下的输出：

```text
2023-10-27 10:00:00,000 INFO     Starting server at tcp:port=8005:interface=127.0.0.1
2023-10-27 10:00:00,000 INFO     HTTP/2 support not enabled (install the http2 and tls Twisted extras)
2023-10-27 10:00:00,000 INFO     Configuring endpoint tcp:port=8005:interface=127.0.0.1
2023-10-27 10:00:00,000 INFO     Listening on TCP address 127.0.0.1:8005
```

### 5. 验证

打开浏览器或使用 `curl` 访问：

```bash
curl http://127.0.0.1:8005/
```

你应该能看到如下 JSON 响应：

```json
{"message": "Hello from Daphne!", "status": "success"}
```

## 配置 OpenTelemetry 探针

### Installation

```
pip install opentelemetry-distro opentelemetry-exporter-otlp
opentelemetry-bootstrap -a install
```


### Start the server

```
export DJANGO_SETTINGS_MODULE=demo_project.settings
```

```
opentelemetry-instrument --traces_exporter console,otlp_proto_grpc --metrics_exporter otlp_proto_grpc --service_name daphne-demo --resource_attributes "token"="xxx","host.name"="alice" --exporter_otlp_endpoint http://127.0.0.1:4317 daphne -p 8005 demo_project.asgi:application
```

Output in the console

```json
{
    "name": "GET index",
    "context": {
        "trace_id": "0x6c2537d22cc9bd59fd55351f07a71ce5",
        "span_id": "0xc2c9c5ff5b6aba8f",
        "trace_state": "[]"
    },
    "kind": "SpanKind.SERVER",
    "parent_id": null,
    "start_time": "2025-12-16T02:57:25.359113Z",
    "end_time": "2025-12-16T02:57:25.362062Z",
    "status": {
        "status_code": "UNSET"
    },
    "attributes": {
        "http.scheme": "http",
        "http.host": "127.0.0.1:8005",
        "net.host.port": 8005,
        "http.flavor": "1.1",
        "http.target": "/",
        "http.url": "http://127.0.0.1:8005/",
        "http.method": "GET",
        "http.server_name": "127.0.0.1:8005",
        "http.user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36",
        "net.peer.ip": "127.0.0.1",
        "net.peer.port": 58950,
        "http.status_code": 200
    },
    "events": [],
    "links": [],
    "resource": {
        "attributes": {
            "telemetry.sdk.language": "python",
            "telemetry.sdk.name": "opentelemetry",
            "telemetry.sdk.version": "1.39.1",
            "token": "xxx",
            "host.name": "alice",
            "service.name": "daphne-demo",
            "telemetry.auto.version": "0.60b1"
        },
        "schema_url": ""
    }
}
```