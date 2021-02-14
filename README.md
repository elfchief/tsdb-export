# tsdb-export

This is a quick and dirty way to export metrics out of OpenTSDB in a way
that will make them usable for other, similar systems. Sadly, OpenTSDB
doesn't provide a good (or really, any) way to export this data.

Basically what we have to do is pull a time range for a metric, look for
any places it's aggregating data (via "aggregateTags"), and for those
places, build new queries using "tag=*", and do it again until we don't
have any aggregations happening, and then write that out.

It is not the fastest thing in the world, but it's functional.

The code outputs metrics that can be piped directly into a network
connection to anything that understands the OpenTSDB metric format, e.g.
`./tsdb-export -metric metric.name -start 1606139854 -end 1606399027 | nc localhost 4242`.
If that doesn't work for you, run through sed or adjust `printSingleMetric`
in the code.

To build: `go build`

Usage:

```txt
Usage of ./tsdb-export:
  -end int
    	Unix timestamp of end of metrics, ex. 1503792000 (2017-08-27 00:00:00)
  -list
    	Extract a list of all metrics
  -metric string
    	Metric name to export
  -querylength int
    	Break down queries to increments of this length (in seconds) (default 86400)
  -start int
    	Unix timestamp of start of metrics, ex. 1459814400 (2016-04-05 00:00:00)
  -url string
    	OpenTSDB URL (default "http://localhost:4242")
```

Export example:

```bash
./tsdb-export -metric tsd.compaction.queue.size -start 1606139854 -end 1606399027 -url http://localhost:4242
```

List metric names:

```bash
./tsdb-export -list -url http://localhost:4242
```

Suggested use:

```bash
for metric in $(./tsdb-export -list -url http://localhost:4242)
do
  ./tsdb-export -metric $metric -start 1606139854 -end 1606399027 -url http://localhost:4242 > $metric.txt
done
```
