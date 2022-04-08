# raid

Air Raid Alerts API (Ukraine)

Production: <https://alerts.dun.ai/>

# Running locally

```sh
# Local:
make run API_KEYS=foo,bar,baz
# Docker:
API_KEYS=foo,bar,baz make run-docker

curl 127.0.0.1:10101/api/states -H 'X-API-Key: foo'
```
