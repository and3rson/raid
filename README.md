# raid

Air Raid Alerts API (Ukraine)

Production: <https://alerts.com.ua/>

[![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/banner-direct-single.svg)](https://stand-with-ukraine.pp.ua/)

# Running locally

```sh
# Local:
make run API_KEYS=foo,bar,baz
# Docker:
make run-docker
# Set DEBUG env var to true to enable verbose logs.
# Set TRACE env var to true to enable VERY verbose logs.

curl 127.0.0.1:10101/api/states -H 'X-API-Key: foo'
```
