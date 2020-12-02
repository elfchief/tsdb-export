package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// Configuration moved to command line arguments
var tsdbEndpoint string

// Useful utility code
func check(e error) {
	if e != nil {
		panic(e)
	}
}

func log(format string, vals ...interface{}) {
	logmsg := fmt.Sprintf(format, vals...)
	fmt.Fprint(os.Stderr, logmsg)
}

func copymap(from map[string]string) map[string]string {
	newmap := make(map[string]string)
	for k, v := range from {
		newmap[k] = v
	}

	return newmap
}

// Actual code
type metricList []string

func getMetricList() []string {
	// 16.777.216 is the max number of metric name UIDs in OpenTSDB
	url := fmt.Sprintf("%s/api/suggest?type=metrics&max=16777216", tsdbEndpoint)

	resp, err := http.Get(url)
	check(err)

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	check(err)

	var ml metricList
	fmt.Printf("%v\n", ml)
	err = json.Unmarshal(body, &ml)
	check(err)

	return ml
}

// Add in: Start time, end time, aggregation function, metric name, tags
var metricURI = "%s/api/query?start=%s&end=%s&m=%s:%s{%s}"

type metricResponse struct {
	Metric        string                 `json:"metric"`
	Tags          map[string]string      `json:"tags"`
	AggregateTags []string               `json:"aggregateTags"`
	Dps           map[string]json.Number `json:"dps"`
}

// Given a metric URL, get it and unmarshal it
type metricResponses []metricResponse

func getMetricFromURL(url string) metricResponses {
	resp, err := http.Get(url)
	check(err)

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	check(err)

	var mr metricResponses
	err = json.Unmarshal(body, &mr)
	if err != nil {
		log("ERROR: Can't unmarshal json body: %v\n", err)
		log("Body:\n%s\n", body)
		panic(err)
	}
	// check(err)

	return mr
}

// Turn a map of tags & values into a slice of strings that we can then
// fix up using strings.Join()
func tagFold(tagList map[string]string) []string {
	foldedTagList := []string{}

	for k, v := range tagList {
		foldedTagList = append(foldedTagList, fmt.Sprintf("%s=%s", k, v))
	}

	return foldedTagList
}

// Get one or more metric series given a specific set of tags
func getMetricSet(metricName string, start string, end string, tagList map[string]string) metricResponses {
	url := fmt.Sprintf(metricURI, tsdbEndpoint, start, end, "sum", metricName, strings.Join(tagFold(tagList), ","))
	m := getMetricFromURL(url)

	return m
}

// Print the actual metric line
// Format: put metric.name timestamp value tag1=value2 tag2=value2
func printSingleMetric(name string, time string, value json.Number, tags string) {
	fmt.Printf("put %s %s %s %s\n", name, time, value, tags)
}

// Given a single metric series in an opentsdb response, print it out
func printMetric(m metricResponse) {
	// Create the specific tag combination for this series
	tagstring := strings.Join(tagFold(m.Tags), " ")

	// Dump all the datapoints
	for k, v := range m.Dps {
		printSingleMetric(m.Metric, k, v, tagstring)
	}
}

// Recurse through this metric and suss out all the tag combination
// Basically, if we have anything in the aggregate tag list, descend another
// layer, and use a wildcard for the aggregated metrics.
func drillMetric(name string, start string, end string, tagList map[string]string) {
	mr := getMetricSet(name, start, end, tagList)
	for _, m := range mr {
		// If we have aggregate tags, dig deeper
		if len(m.AggregateTags) > 0 {
			newTagList := copymap(tagList)
			// Add new tag name to list as a wildcard
			for _, v := range m.AggregateTags {
				newTagList[v] = "*"
			}
			// log("INFO: %d aggregate tags, digging deeper on metric %s (%v)\n", len(m.AggregateTags), name, newTagList)
			drillMetric(name, start, end, newTagList)
		} else {
			// log("INFO: %d aggregate tags, gone as deep as we can on metric %s (%v)\n", len(m.AggregateTags), name, tagList)
			printMetric(m)
		}
	}
}

func main() {

	start := flag.Int("start", 0, "Unix timestamp of start of metrics, ex. 1459814400 (2016-04-05 00:00:00)")
	end := flag.Int("end", 0, "Unix timestamp of end of metrics, ex. 1503792000 (2017-08-27 00:00:00)")
	timeRange := flag.Int("querylength", 86400, "Break down queries to increments of this length (in seconds)")
	flag.StringVar(&tsdbEndpoint, "url", "http://localhost:4242", "OpenTSDB URL")
	list := flag.Bool("list", false, "Extract a list of all metrics")
	metric := flag.String("metric", "", "Metric name to export")

	var Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if len(os.Args) < 5 && *list == false {
		Usage()
		os.Exit(1)
	}

	if *list == true {
		ml := getMetricList()
		for _, v := range ml {
			fmt.Println(v)
		}
		os.Exit(0)
	}

	if *end == 0 || *start == 0 || *start >= *end {
		fmt.Fprintf(os.Stderr, "Both -start and -end needs to be specified, also -start needs to be less than -end.\n")
		os.Exit(1)
	}

	if *metric == "" {
		fmt.Fprintf(os.Stderr, "-metric needs to be specified.\n")
		os.Exit(1)
	}

	log("INFO: start: %d, end: %d\n", *start, *end)
	for currentPosition, lastPosition := *end-*timeRange, *end; *start <= currentPosition+*timeRange; lastPosition, currentPosition = currentPosition, currentPosition-*timeRange {
		// safe guard, so we don't query before specified time range.
		if currentPosition < *start {
			currentPosition = *start - 1
		}
		log("INFO: Processing metric %s, time range %d to %d\n", *metric, currentPosition, lastPosition)
		drillMetric(*metric, strconv.Itoa(currentPosition), strconv.Itoa(lastPosition), map[string]string{})
	}
}
