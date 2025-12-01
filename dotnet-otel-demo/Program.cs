using OpenTelemetry.Resources;
using OpenTelemetry.Trace;

var builder = WebApplication.CreateBuilder(args);

// 配置 OpenTelemetry
builder.Services.AddOpenTelemetry()
    .ConfigureResource(resource => resource
        .AddAttributes(new Dictionary<string, object>
        {
            ["service.name"] = "otel-demo-service",
            ["token"] = "xxxxxxxxx"  // 替换为实际的 token
        }))
    .WithTracing(tracing => tracing
        .AddAspNetCoreInstrumentation()
        .AddHttpClientInstrumentation()
        .AddOtlpExporter(options =>
        {
            options.Endpoint = new Uri("http://ap-guangzhou.apm.tencentcs.com:4319");
            options.Protocol = OpenTelemetry.Exporter.OtlpExportProtocol.Grpc;
        }));

builder.Services.AddHttpClient();

var app = builder.Build();


app.MapGet("/", () => "OpenTelemetry Demo for dotnet");

app.Run();
