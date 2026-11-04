# js8scan_go
A helper APP written in Go to enable js8Call to act as a radio scanner

Frequency-scanning companion for JS8Call, talking to its TCP API.

1. In JS8Call: File > Settings > Reporting > API: enable the TCP server (default port 2442).
2. `cp config.ini.example config.ini` and edit.
3. `go build -o js8scan . && ./js8scan -config config.ini`

Receive-only by default; set `[tx] enabled = true` to transmit.
