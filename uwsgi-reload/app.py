from uwsgidecorators import postfork
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor, ConsoleSpanExporter
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.instrumentation.flask import FlaskInstrumentor
from flask import Flask, jsonify

app = Flask(__name__)

@postfork
def init_tracing():
    """在每个 worker 进程 fork 后初始化 OpenTelemetry"""
    print(f"Initializing OpenTelemetry in worker process, PID: {__import__('os').getpid()}")

    resource = Resource.create({
        "service.name": "tapm-flask-demo", # 配置应用名
        "token": "xxxxxxxxxxx", # 这里配置控制台的 Token
        "host.name": "localhost"
    })

    # 创建新的 TracerProvider
    tracer_provider = TracerProvider(resource=resource)
    trace.set_tracer_provider(tracer_provider)

    # 配置控制台导出器
    # console_exporter = ConsoleSpanExporter()
    # console_processor = BatchSpanProcessor(console_exporter)
    # trace.get_tracer_provider().add_span_processor(console_processor)

    # 发送到 APM 后端
    otlp_exporter = OTLPSpanExporter(
         endpoint="http://ap-beijing-fsi.apm.tencentcs.com:4317/v1/traces", # 这里配置控制台的 endpoint，这里以北京金融地域为例，保留后面的/v1/traces
         headers={}
     )
    otlp_processor = BatchSpanProcessor(otlp_exporter)
    trace.get_tracer_provider().add_span_processor(otlp_processor)

    # 自动插桩 Flask
    # 如果有其他的组件，如数据库等，也需要在前面 import 相关的包，然后在这里初始化
    # 查询所有支持的组件：https://github.com/open-telemetry/opentelemetry-python-contrib/tree/main/instrumentation
    FlaskInstrumentor().instrument_app(app)
    print("OpenTelemetry initialization completed!")


# 业务代码，这里不用修改代码，init_tracing 中已经完成了探针的初始化
@app.route('/', methods=['GET'])
def root():
    return jsonify({
        'status': 'success',
        'message': 'Flask application is running',
        'port': 5007
    })

