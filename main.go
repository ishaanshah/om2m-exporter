package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var interval time.Duration
var url string
var path string
var username string
var password string
var lastCollectedTs time.Time
var client *http.Client

type currentCollector struct {
	applianceCount       *prometheus.Desc
	applianceStatus      *prometheus.Desc
	applianceConsumption *prometheus.Desc
}

func newCurrentCollector() *currentCollector {
	return &currentCollector{
		applianceCount: prometheus.NewDesc("appliance_count",
			"The number of appliances connected", nil, nil),
		applianceStatus: prometheus.NewDesc("appliance_status",
			"Shows the status of the connected appliance", []string{"appliance"}, nil),
		applianceConsumption: prometheus.NewDesc("appliance_consumption",
			"Shows the current consumed by a device", []string{"appliance"}, nil),
	}
}

func (collector *currentCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.applianceCount
	ch <- collector.applianceStatus
	ch <- collector.applianceConsumption
}

func (collector *currentCollector) Collect(ch chan<- prometheus.Metric) {
	// Get list of appliances
	// TODO: Add label filter later
	req, _ := http.NewRequest("GET", url+"/~"+path+"?fu=1", nil)
	req.Header.Set("X-M2M-Origin", username+":"+password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error while fetching list of appliances: %v", err)
		return
	}

	if resp.StatusCode != 200 {
		log.Errorf("Error while fetching list of appliances: %s", resp.Status)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error while reading list of appliances: %v", err)
		return
	}

	// Read JSON into map[string]interface{}, ideally we should have a correct type defined
	res := make(map[string]interface{})
	err = json.Unmarshal(body, &res)
	if err != nil {
		log.Errorf("Error while reading list of appliances: %v", err)
		return
	}

	endpoints := res["m2m:uril"].([]interface{})
	labels := make([]string, len(endpoints))
	for i := 0; i < len(endpoints); i += 1 {
		labels[i] = strings.ReplaceAll(endpoints[i].(string), path+"/", "")
	}

	applianceStatus := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "appliance_count",
	}, []string{"appliance"})

	ch <- prometheus.MustNewConstMetric(collector.applianceCount,
		prometheus.GaugeValue, float64(len(endpoints)))

	for i := 0; i < len(labels); i += 1 {
		applianceStatus.With(prometheus.Labels{"appliance": labels[i]}).Set(1)
		ch <- prometheus.MustNewConstMetric(collector.applianceStatus,
			prometheus.GaugeValue, float64(i), labels[i])
		ch <- prometheus.MustNewConstMetric(collector.applianceConsumption,
			prometheus.GaugeValue, float64(i), labels[i])

		// Get last value
		req, _ := http.NewRequest("GET", url+"/~"+path+labels[i]+"/", nil)
		req.Header.Set("X-M2M-Origin", username+":"+password)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("Error while fetching last value for %s: %v", labels[i], err)
			continue
		}

		if resp.StatusCode != 200 {
			log.Errorf("Error while fetching list of appliances: %s", resp.Status)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Error while reading list of appliances: %v", err)
		}

		// Read JSON into map[string]interface{}, ideally we should have a correct type defined
		res := make(map[string]interface{})
		err = json.Unmarshal(body, &res)
		if err != nil {
			log.Errorf("Error while reading list of appliances: %v", err)
		}
	}
}

func main() {
	flag.DurationVar(&interval, "interval", 30, "The interval at which to probe the OneM2M endpoint. "+
		"If the latest data point is older than the interval then the endpoint is assumed to be dead.")
	flag.StringVar(&url, "url", "", "The URL of the base OneM2M endpoint.")
	flag.StringVar(&path, "path", "", "The path to the base data container.")
	flag.StringVar(&username, "username", "", "The username to access the OneM2M endpoint.")
	flag.StringVar(&password, "password", "", "The password to access the OneM2M endpoint.")
	flag.Parse()

	collector := newCurrentCollector()
	prometheus.MustRegister(collector)

	client = &http.Client{}

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":9876", nil)
}
