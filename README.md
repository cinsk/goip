
Introduction
============

Simple IP Geolocation command written in [Go](https://golang.org).

Remark
======

Special thanks to [MaxMind](https://www.maxmind.com) for its generosity providing free Gelocation database.
Without their free database available [here](https://dev.maxmind.com/geoip/geoip2/geolite2/), `goip` does not work.

Build
=====

Set up Go environment (esp. `GOPATH` environment variable), and do following:

        $ go get github.com/cinsk/goip

Usage
=====

Database
--------

By default, `goip` will download free MaxMind geolocation database (zip format), unpack it, load it, then it will serve the requests.   This may be helpful if you do not want to download the database manually, but it will take several minutes to become ready.

To speed up, try to download the zip file manually by visiting [MaxMind](https://dev.maxmind.com/geoip/geoip2/geolite2/), and select GeoLite2 City, CSV format, zipped link.  Then unpack the zip file, and provide the directory name using `-u` option:

        $ wget -q 'http://geolite.maxmind.com/download/geoip/database/GeoLite2-City-CSV.zip'
        $ unzip GeoLite2-City-CSV.zip 
        Archive:  GeoLite2-City-CSV.zip
          inflating: GeoLite2-City-CSV_20171205/GeoLite2-City-Blocks-IPv4.csv
          ...
        
        $ goip -d GeoLite2-City-CSV_20171205 ...

Batch mode
----------

`goip` will listen its standard output for the list of IP addresses, and print the information of the IP addresses.  The information includes the name of city with the country code, and the number of occurrance (a.k.a. *population*), and the city's longitude and latitude.   If the country or city name is not known, it will be printed as *UNKNOWN*, unless `-U` option was given.

        $ cat ip.lst
        111.111.111.111
        221.159.164.3
        221.159.164.3
        $ cat ip.lst | goip
        name,pop,lat,lon,group
        "KR: Boseong",2,34.7697,127.0809,1
        "JP: Tokyo",1,35.685,139.7514,1

You can change the output format from csv to text (each line delimited by a tab character):

        $ cat ip.lst | ./goip -t text
        name	pop	lat	lon	group
        KR: Boseong	2	34.7697	127.0809	0
        JP: Tokyo	1	35.685	139.7514	0

All output is sorted by 'pop' field (the number of occurrence), descending order, limited to 1000 entries.  Use `-l xxx` to change the limit to `xxx`.  Use negative limit (e.g. `-l -1`) for the unlimited output.

To change the order of fields, or number of fields, use `-o FIELDS` options where FIELDS are list of fields separated by comma.  Supported names are *name*, *lat*, *lon*, *pop*, and *group*:

        $ cat ip.lst | goip -o name,pop
        name,pop
        "KR: Boseong",2
        "JP: Tokyo",1

Grouping (Clustering)
---------------------

`goip` supports simple clustering of entries, useful for clustering its output (via output field *group*).   It uses simplified [k-means clustering](https://en.wikipedia.org/wiki/K-means_clustering) for grouping, 20 iteration at most.  If you want to change the number of iteration to *XXX*, use `-G XXX` option.

The group(cluster) id begins with zero, upto 5 by default.  To change the number of groups, use `-g NGROUP` option.  Note that the output may contain less number of groups.   If *NGROUP* is negative, grouping/clustering will be disabled.

Server Mode
-----------

If you provide a TCP listening address and port via `-T address:port`, `goip` will listen `address:port` beside reading standard input.   Note that even in this mode, `goip` will still try to read from standard input.  If you close the standard input, `goip` will terminate instantly.

        $ ./goip -T localhost:8888
        server ready

You can use any TCP client such as `nc(1)` or `netcat(1)` to communicate with `goip` server.

Unless special command which begins with `.` or `!` character, `goip` assumes each line (delimited by `\n` character) contains an IP addrese.

        $ echo 221.159.164.3 | nc localhost 8888
        KR:Boseong
        
        [After 5 seconds...]
        $ _

        $ cat ip.lst
        111.111.111.111
        221.159.164.3
        221.159.164.3
        $ cat ip.lst | nc localhost 8888
        JP:Tokyo
        KR:Boseong
        KR:Boseong
        
        [After 5 seconds...]
        $ _

The connection to the `goip` server will be automatically closed after 5 seconds on idle.  Or, you can request explicit disconnect via `.quit` command:

        $ echo -e '221.159.164.3\n.quit' | nc localhost 8888
        KR:Boseong
        $ _

Or, you can create a bash function like this:

        geoip() {
            local IFS=$'\n'
            local args="$*"
            echo -e "${args}\n.quit" | nc localhost 8888
        }
          
And use like this:

        $ geoip 1.1.1.1 2.2.2.2 3.3.3.3
        AU:Research
        FR:UNKNOWN
        US:Fairfield
        $ _

It also supports `.stat` command that will give you the same statisticial output in batch mode, and `.reset` to clear internal data for `.stat` command.

Note that `.stat` command is very expensive, and `goip` does not handle more than 1 request at a time.  If you're looking for a sturdy server for querying geolocation, consider to use other solution such as [freegeoip](https://github.com/fiorix/freegeoip).




