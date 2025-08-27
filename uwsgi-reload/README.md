排查 uWSGI reload 造成 OpenTelemetry 插桩失败问题。

pre-fork 模型导致 OpenTelemetry 无法正常插桩。


注意：不要在 uwsgi.ini 中添加 `lazy-apps` 选项。

### Usage

Start:

```
uwsgi --ini uwsgi.ini
```

Reload:

```
uwsgi --reload /tmp/uwsgi.pid
```

### Ref
- https://www.hyperdx.io/blog/opentelemetry-python-server-auto-instrumentation
- https://www.hyperdx.io/docs/install/python#if-you-are-using-gunicorn-uwsgi-or-uvicorn
- https://uptrace.dev/guides/opentelemetry-django
