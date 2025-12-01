# OpenTelemetry Demo

一个简单的 ASP.NET Core 应用，演示 OpenTelemetry 集成。

## 功能

- 集成 OpenTelemetry Tracing
- 使用 gRPC OTLP Exporter 发送到腾讯云 APM
- 自动追踪 ASP.NET Core 请求和 HttpClient 调用

## OpenTelemetry 配置

- **Endpoint**: `http://ap-guangzhou.apm.tencentcs.com:4319`
- **Protocol**: gRPC
- **Resource Attributes**:
  - `service.name`: otel-demo-service
  - `token`: your-token-here

## 运行

```bash
dotnet restore
dotnet run
```

## API 端点

| 路径 | 描述 |
|------|------|
| `/` | 首页 |

## 修改配置

编辑 `Program.cs` 中的配置：

```csharp
.AddAttributes(new Dictionary<string, object>
{
    ["service.name"] = "your-service-name",
    ["token"] = "your-actual-token"
})
```
