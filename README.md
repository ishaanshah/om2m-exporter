# OneM2M Exporter

[![GitHub go.mod Go version of a Go module](https://img.shields.io/github/go-mod/go-version/ishaanshah/om2m-exporter.svg)](https://github.com/ishaanshah/om2m-exporter)
[![GoReportCard example](https://goreportcard.com/badge/github.com/ishaanshah/om2m-exporter)](https://goreportcard.com/report/github.com/ishaanshah/om2m-exporter)
[![GitHub license](https://img.shields.io/github/license/ishaanshah/om2m-exporter.svg)](https://github.com/ishaanshah/om2m-exporter/blob/master/LICENSE)

Prometheus exporter for data collected in OneM2M server

## Usage

```
❯ ./om2m_exporter -h
Usage of ./om2m_exporter:
  -interval duration
        If the latest data point is older than the interval then the appliance is assumed to be inactive. (default 30s)
  -password string
        The password to access the OneM2M endpoint.
  -path string
        The path to the base data container.
  -timezone string
        The timezone where the appliances are located. (default "Asia/Kolkata")
  -url string
        The URL of the base OneM2M endpoint.
  -username string
        The username to access the OneM2M endpoint.
```

## Build

The project is managed using Go Modules. Clone the repository and install the dependencies by running:

```bash
go mod tidy
```

Then build the project using:

```bash
go build
```

## Note

The exporter written is very specialized. It can be used as a good starting point for making something more generalized.
