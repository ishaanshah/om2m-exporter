package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"strconv"
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
var timezone string
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
	req, _ := http.NewRequest("GET", url+"/~"+path+"?fu=1&ty=3&lbl=Type/appliance", nil)
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

	ch <- prometheus.MustNewConstMetric(collector.applianceCount,
		prometheus.GaugeValue, float64(len(endpoints)))

	for i := 0; i < len(labels); i += 1 {
		// Get last value
		req, _ := http.NewRequest("GET", url+"/~"+path+"/"+labels[i]+"/la", nil)
		req.Header.Set("X-M2M-Origin", username+":"+password)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("Error while fetching last value for %s: %v", labels[i], err)
			continue
		}

		if resp.StatusCode != 200 {
			if resp.StatusCode == 404 {
				ch <- prometheus.MustNewConstMetric(collector.applianceStatus,
					prometheus.GaugeValue, 0, labels[i])
				ch <- prometheus.MustNewConstMetric(collector.applianceConsumption,
					prometheus.GaugeValue, 0, labels[i])
			} else {
				log.Errorf("Error while fetching last value for %s: %s", labels[i], resp.Status)
			}
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Error while reading last value for %s: %v", labels[i], err)
			continue
		}

		// Read JSON into map[string]interface{}, ideally we should have a correct type defined
		res := make(map[string]interface{})
		err = json.Unmarshal(body, &res)
		if err != nil {
			log.Errorf("Error while reading last value for %s: %v", labels[i], err)
			continue
		}

		tz, _ := time.LoadLocation(timezone)
		ct, _ := time.ParseInLocation("20060102T150405",
			res["m2m:cin"].(map[string]interface{})["ct"].(string), tz)
		con, _ := strconv.ParseFloat(res["m2m:cin"].(map[string]interface{})["con"].(string), 64)

		// Time elapsed is more than the interval, so assume that the appliance is off
		if time.Since(ct) > interval {
			ch <- prometheus.MustNewConstMetric(collector.applianceStatus,
				prometheus.GaugeValue, 0, labels[i])
			ch <- prometheus.MustNewConstMetric(collector.applianceConsumption,
				prometheus.GaugeValue, 0, labels[i])
		} else {
			ch <- prometheus.MustNewConstMetric(collector.applianceStatus,
				prometheus.GaugeValue, 1, labels[i])
			ch <- prometheus.MustNewConstMetric(collector.applianceConsumption,
				prometheus.GaugeValue, con, labels[i])
		}
	}
}

func main() {
	flag.DurationVar(&interval, "interval", 30*time.Second, "The interval at which to probe the OneM2M endpoint. "+
		"If the latest data point is older than the interval then the endpoint is assumed to be dead.")
	flag.StringVar(&url, "url", "", "The URL of the base OneM2M endpoint.")
	flag.StringVar(&path, "path", "", "The path to the base data container.")
	flag.StringVar(&username, "username", "", "The username to access the OneM2M endpoint.")
	flag.StringVar(&password, "password", "", "The password to access the OneM2M endpoint.")
	flag.StringVar(&timezone, "timezone", "Asia/Kolkata", "The timezone where the appliances are located.")
	flag.Parse()

	collector := newCurrentCollector()
	prometheus.MustRegister(collector)

	client = &http.Client{}

	http.Handle("/metrics", promhttp.Handler())

	log.Info("Started serving at localhost:9876/metrics")
	log.Fatal(http.ListenAndServe(":9876", nil))
}
