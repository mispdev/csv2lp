# csv2lp

## What it is

`csv2lp` is a commandline tool to convert annotated CSV as returned by Flux queries to the InfluxDB line protocol format.

The tool itself just wraps the official ["csv2lp" library of InfluxDB](https://github.com/influxdata/influxdb/tree/master/pkg/csv2lp), reads a given CSV file, feeds it to the library and returns the result. By doing so it makes the library's functionality available as a standalone commandline tool.

I quickly put this tool together for a personal use case, but then decided to share it here, just in case it might be useful for someone else.

## Context

Compared to InfluxQL, [Flux](https://docs.influxdata.com/influxdb/v2.0/reference/flux/) is a more powerful query language (cf. comparison [here](https://docs.influxdata.com/influxdb/v2.0/reference/flux/flux-vs-influxql/)) to query data from an [InfluxDB](https://www.influxdata.com/). So there are many reasons to use Flux instead of InfluxQL - some as simple as Flux supporting [type casts from boolean](https://docs.influxdata.com/influxdb/v1.8/flux/flux-vs-influxql/#cast-booleans-to-integers) to integer or float, which InfluxQL does not support (there are [feature requests](https://github.com/influxdata/influxdb/issues/7562) for this dating back to 2016).

When querying InfluxDB using Flux, the result is usually returned as [annotated CSV](https://docs.influxdata.com/influxdb/cloud/reference/syntax/annotated-csv/). Depending on the version of InfluxDB you use, you might be able to import such annotated CSV, or not. If dealing with InfluxDB versions below 2.0, it is likely that you are not able to do so. In this case, a tool like `csv2lp` can be used to convert the annotated CSV to the [line protocol format](https://docs.influxdata.com/influxdb/cloud/reference/syntax/line-protocol/), which can easily be imported into older 1.x versions of InfluxDB.

## How to build

### Prerequisites

`csv2lp` was developed with Go 1.15. So a basic prerequisite is to have a compatible Go version installed. See the official [Golang](https://golang.org/) website for installation instructions.

### Build

Download or clone this repository to a directory of your choice.

Open a console/terminal within that directory and execute the following command:

```
go install
```

Go should download all required dependencies automatically and compile the tool.

You should be able to find the resulting binary in the default directory `$GOPATH/bin`, and thus should be able to execute it like this:

```
$GOPATH/bin/csv2lp
```

You should now see the following:

```
csv2lp v0.0.1
-------------

Convert annotated CSV as exported via Flux to InfluxDB line protocol.

Using csv2lp[1] library of InfluxDB.
[1] https://github.com/influxdata/influxdb/tree/master/pkg/csv2lp

Syntax: csv2lp <csv-file-name>
```

Use `csv2lp` at your own risk.

### Pre-Build Binary

The release assets of this repository include a pre-built Linux binary of the tool (x86, 64 bit, built with Go 1.15 on Ubuntu 20.04).

## Example use case: boolean conversion

### The problem

Imagine you have a measurement in your InfluxDB v1.8.x that has a field of type boolean. You are using this measurement in a Grafana panel to visualize its state. Now you want to define an Grafana alert on this state, and thus define an alert query on the panel query. You soon realize, that the alert state in Grafana is never "ok" or "alerting", but always stays "undefined". You test the alert query you defined with the "Test Rule" button, and discover that your query always returns no results. Why is this?

When defining the alert query, you define a query with aggregation functions on top of the panel query. As InfluxQL is not able to calculate aggregations of boolean values and is unable to convert boolean values to numerical values like 0 and 1, the query will always fail. In short: You cannot define alert queries on panel queries that return boolean values. If you would have used a numerical type like integer or float, and stored boolean values e.g. as 0 and 1, then this would work like a charm.

So how to convert your boolean field to a number in your already existing measurement? Short answer: You cannot. The [FAQ](https://docs.influxdata.com/influxdb/v1.8/troubleshooting/frequently-asked-questions/#can-i-change-a-fields-data-type) mentions two workarounds, but they do not really solve the underlying issue.

### Solution approaches

If you are lucky enough to have a more recent version of InfluxDB in use (like e.g. 1.8.x), and are able to [activate Flux](https://docs.influxdata.com/influxdb/v1.8/flux/installation/) on it, then you have two options:

 - Add an additional [Flux data source](https://www.influxdata.com/blog/how-grafana-dashboard-influxdb-flux-influxql/) to Grafana and use Flux queries in Grafana that do the required boolean conversion on the fly. But note that you cannot use multiple datasources in a single panel, and thus cannot mix InfluxQL and Flux queries in a single panel.

 - Use Flux to export your measurement with needed type conversion to annotated CSV, drop your measurement in the InfluxDB and import again. If your InfluxDB does not support importing annotated CSV, then use `csv2lp` to convert it to line protocol format and import that.

Let's take a closer look at the last option.

### Converting and exporting via Flux

Let's assume you have a database/bucket called "mydb" in your InfluxDB, and within that your have a measurement called "mymeasurement" with a field named "value" of type boolean. You want to convert the measurement in a way that at the end the field "value" has a number type. One possible Flux query to export your measurement with such a conversion could look like this:

```
    partA = from(bucket: "mydb")
        |> range(start: -730d)
        |> filter(fn: (r) => r._measurement == "mymeasurement" and r._field != "value")

    partB = from(bucket: "iobroker")
        |> range(start: -730d)
        |> filter(fn: (r) => r._measurement == "mymeasurement" and r._field == "value")
        |> toFloat()

    union(tables: [partA, partB])
        |> yield()
```

This queries all fields from your measurement as-is, that have NOT the name "value", and stores the result in a table called "partA". Then it queries the field with the name "value", converts its values to float, and stores the result in a table called "partB". Finally, both tables ("partA" and "partB") are combined again, and the resulting table is given as the final result of the query. By default, Flux will return the result as annotated CSV.

If you write your Flux query to a file called `fluxquery.txt` you can execute the query and store the result in the file `myfluxresult.csv` using `curl` like this:

```
curl -XPOST <host-of-influxdb>:8086/api/v2/query -sS -H 'Accept:application/csv' -H 'Content-type:application/vnd.flux' -d @fluxquery.txt > myfluxresult.csv
```

Please note: When exporting your measurement like this, make sure that you apply a range in the query that spans a large enough time span to cover all your needed data points you want to export. Here in the example we use "-730d", which means "from now go 730 days back in time" (2 years).

### Deleting old measurement

Before we can import the converted measurement, we need to delete (drop) the old measurement in the InfluxDB. If you have anything ingesting new data points to this measurement, you should stop/pause it before doing so. Dropping a measurement can be done e.g. using [`influx` CLI](https://docs.influxdata.com/influxdb/v1.8/tools/shell/):

```
    > USE mydb
    > DROP MEASUREMENT "mymeasurement"
```

### Importing to InfluxDB

Now we need to import the converted measurement from the annotated CSV into InfluxDB. Assuming your InfluxDB is not able to import annotated CSV directly (more on how to check this later), we use the `csv2lp` tool to convert the annotated CSV in file `myfluxresult.csv` to line protocol format and store it in file `mylines.txt`:

```
csv2lp myfluxresult.csv > mylines.txt
```

Now you can import the content of `mylines.txt` into your InfluxDB. One option to do this is using the `-import` parameter of the `influx` CLI (see section "Import data from a file with -import" [here](https://docs.influxdata.com/influxdb/v1.8/tools/shell/#import-data-from-a-file-with--import) for details).

Another option is using the web UI of [Chronograf](https://www.influxdata.com/time-series-platform/chronograf/). In Chronograf choose "Explore" on the left side. Now on the right upper side click the button "Write Data". In the appearing dialog box make sure the correct database/bucket is selected in the drop-down next to "Write Data To". Now you can drag and drop the file `mylines.txt` to the according place in the dialog box to upload it. Finally, click "Write this File" button to write its contents to InfluxDB.

Right after writing the data you can also use the "Explore" mode of Chronograf to check if your measurement was imported correctly. A simple query can be quickly assembled by selecting the database/bucket, the measurement, and a field from the measurement you want to plot in the graph. If plotting your (formerly boolean) "value" field, you should ideally see it display nicely in the graph instead of getting an error message that boolean values cannot be handled.

### Checking if importing annotated CSV works directly

Depending on your version of InfluxDB it might be possible that you can import annotated CSV directly, so you do not need to do the additional step of using `csv2lp` to convert to line protocol format.

To check this, in Chronograf switch from "InfluxQL" to "Flux" in explore mode (via buttons right next to the drop-down for selecting the data source in upper left area). Once in Flux mode, enter the following Flux query:

```
    import "experimental/csv"
    csv.from(url: "http://<host:port>/path/to/myfluxresult.csv")
      |> to(bucket: "mydb")
```

This should import the annotated CSV, and then write it to the given database/bucket. You need to serve your `myfluxresult.csv` via HTTP, as file access is not possible. To do so, you could upload your CSV to a service of your liking that makes it available under a publicly available URL. Alternatively, if your InfluxDB is within your local network, you could quickly spin up a local HTTP server to serve the file. If you have Python 3 installed, this can be done as quickly as writing a single line (which will serve all files in current directory via HTTP on port 12345):

```
python3 -m http.server 12345
```

After you have made available your `myfluxresult.csv` via HTTP, and executed the Flux query above using button "Run Script", you will see if it works. With newer versions of InlfuxDB it should (cf. e.g. this short [YouTube video](https://www.youtube.com/watch?v=wPKZ9i0DulQ) called "How to Write an Annotated CSV with Flux to InfluxDB").

In my case (InfluxDB 1.8.x), it didn't work. I just got the following error message: `Error calling function "to": function "to" is not implemented.`

Note: When removing the `|> to(bucket: "mydb")` part of the Flux query, the data is read from the annotated CSV and shown perfectly fine. So reading the data is possible even with InfluxDB v1.8.x, but writing it to InfluxDB is not.

In such a situation, when you successfully managed to export data via Flux queries to annotated CSV, but are unable to import this annotated CSV again into the exact same InfluxDB it came from... well... perhaps it's time to give `csv2lp` a try. ????

## License

If you find `csv2lp` helpful, feel free to use it **at your own risk. No warranties attached.**

This project is licensed under [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0) (not affecting the licenses of wrapped/used libraries).
