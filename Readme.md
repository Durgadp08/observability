# Project

This is a small project about how the observablity and otel works.
Daily we encounter generally encounter production issues or sometimes would like to debug the staging and then we just open the grafana or clickstack dashboard to view the logs, traces and spans.

1. But how they are coming from the production or staging servers to grafana dashboard ?
2. why there are multiple sources for the telemetery data ? 

### Prerequisites

#### Telemetry Data

1. Metrics :- It gives us high level statistical view of how an endpoint is performing across all users.
2. Traces :- A trace tells about the entire life cycle of a single request. A trace contains spans.
these are useful to find exactly where the request is taking more time ( traces are useful only when we have good spans, a trace with no spans gives no information about the request life cycle it will have only start and end time of whole request).
3. Logs :- Once we find a slow request in our traces, we can check the exact place why it is taking long time.

Telemetry data can be two types like pull or push based 
1. Pull based (like Prometeous) is a , Generally we write our metrics to an endpoint and then we need to tell the promteous. Hey go and scrap the data from the endpoint.
2. Push based (OTLP exporter) where the client will send the data to that endpoint

Later grafana can use the data from that endpoint and visualvisize them. 

Generally we can say like this, metrics gives an high level overview of an endpoint, traces tells where the issue and logs tells why it is happening.

When a request comes to an endpoint we will generally starts a trace at the begining of the handler and it will return a new context and span. 