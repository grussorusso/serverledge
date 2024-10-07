Serverledge relies on [OpenTelemetry](https://opentelemetry.io) for optional
request tracing, aimed at performance investigations.

## Enabling tracing

Tracing is disabled by default. It can enabled with the following configuration
line:

    tracing.enabled: true

By default, JSON-encoded traces are written to `./traces-<timestamp>.json`.
The following line sets a custom output file:

    tracing.outfile: /path/to/file.json

