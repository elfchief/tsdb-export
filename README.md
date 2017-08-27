This is a quick and dirty way to export metrics out of OpenTSDB in a way
that will make them usable for other, similar systems (in my case,
Wavefront). Sadly, OpenTSDB doesn't provide a good (or really, any) way
to export this data.

Basically what we have to do is pull a time range for a metric, look for
any places it's aggregating data (via "aggregateTags"), and for those
places, build new queries using "tag=*", and do it again until we don't
have any aggregations happening, and then write that out.

It is not the fastest thing in the world, but it's functional.

The code outputs metrics that can be piped directly into a network
connection to anything that understands the OpenTSDB metric format,
e.g. `./tsdb-export metric.name | nc localhost 4242`. If that doesn't
work for you, run through sed or adjust `printSingleMetric` in the code.

Configuration for everything except the name of the metric to export is
done by editing the source (look for "CONFIGURE HERE"). Yes, that's crap.
To be quite honest, this entire chunk of code is crap. It served a need.
If it doesn't serve YOUR need, feel free to send patches.

To build: `go build`

To use:

    ./tsdb-export metric.name
    
or

    ./tsdb-export --list
    
Suggested use:
```
for metric in $(./tsdb-export --list)
do
    ./tsdb-export $metric > $metric.txt
done
```
