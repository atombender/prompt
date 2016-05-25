# Prompt

Prompt is an exporter for [Prometheus](https://prometheus.io) that spawns a set of registered exporters on demand as subprocesses.

It exposes two endpoints:

* `/metrics` will spawn all exporters concurrently and return their results as a single text document.
* `/metrics/<name>` will spawn a single exporter.

A subprocess is expected to write the Prometheus text file format to standard output. If a subprocess fails, its output is not included in the merged output.

## Installation

```shell
$ go get github.com/atombender/prompt
```

## Configuration

Example YAML configuration:

```yaml
exporters:
- name: elasticsearch
  command: /opt/exporters/elasticsearch localhost:9200
  minInterval: 10
```

The `minInterval` specifies an optional interval during which the output will be cached in memory. This allows per-command caching without requiring that `scrapeInterval` be tuned centrally in the Prometheus configuration.

The `command` is wrapped with `sh -c "<command>"` and will inherit the environment and user context of the parent process. It's recommended that you use `sudo -u nobody` or similar whenever possible.

## Running

Example invocation:

```shell
$ prompt -l localhost:9100 -c prompt.conf
```

The `-c` option can be specified multiple times, and supports directories: `-c conf.d` will read all `.conf` files inside that directory.

To daemonize, run using a process manager such as Systemd, Upstart, Debian's `start-stop-process` or similar.

## License

MIT license. See the `LICENSE` file.
