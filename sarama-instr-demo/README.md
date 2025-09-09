## Sarama manual instrumentation demo

### Usage

```
go main.go // start producer

go main.go consumer // start consumer
```


### Required span attributes

- "mq.broker"
- "peer.filled"
- "peer.service"

