# js8scan_go
A helper APP written in Go to enable js8Call to act as a radio scanner

Frequency-scanning companion for JS8Call, talking to its TCP API.

# How to install/setup
1. In JS8Call: File > Settings > Reporting > API: enable the TCP server (default port 2442).
2. cp config.ini.example config.ini and edit.
3. Or run ./js8scan -setup to answer prompts and generate config.ini.
4. go build -o js8scan . && ./js8scan -config config.ini

Receive-only by default; set `[tx] enabled = true` to transmit.

# Running post setup
1. go build -o js8scan .
2. ./js8scan -setup

## Assumptions
    this is running on same machine as the js8call instance in order to examine the logs directly
        but there's no reason this couldn't be running on another device since it uses an API
        RX.GET_BAND_ACTIVITY looks like what we want to use

    the log watchers appear to assume that js8call stays running
        when when running on the same machine we can get the js8call instance pid from /tmp/JS8Call.lock
        we can use this to auto determine if log viewers should hibernate or exit

    assumes that js8call doesn't have autoreply on, should check this since we don't want to
        jump to a new frequency if js8 is actively responding to a request

    looks like the config file might only support a single frequency per band
        we definitely want to support multiple per band
