This API allows you to query & listen to air raid alerts in Ukraine.

Data is taken from <https://telegram.me/air_alert_ua> via HTTP API.

Events are usually delayed for up to 5 seconds.

Only states are supported at this moment - 24 total plus Kyiv city. Crimea is absent from this list since no information is available. But we all know that Crimea is Ukraine.

## Authentication

You will need a key to use this API.

  - To request a key, please send me an email (<a@dun.ai>) or ping me in Telegram ([\@andunai](https://t.me/andunai)).
  - Include the key with every request in `X-API-Key` header.

Please be aware that this API is rate-limited. If you spam more than 10 RPS, you will be throttled.

## Endpoints

### `/api/states`

Returns the list of states with their statuses.

Example response:

```
{
  "states": [
	{
	  "id": 1,
	  "name": "Вінницька область",
	  "alert": false,
	  "changed": "2022-04-05T06:12:52+03:00"
	},
	{
	  "id":2,
	  "name": "Волинська область",
	  "alert": false,
	  "changed": "2022-04-05T06:13:06+03:00"
	},
	// ...
  ]
}
```

### `/api/states/live`

[SSE](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events) endpoint which yields alert events in real time.

Example events:

```
event: hello
data: null

event: ping
data: null

event: update
data: {"state":{"id":12,"name":"Львівська область","alert":false,"changed":"2022-04-05T06:14:56+03:00"}}
```

## Use the source, Luke

Made by [Andrew Dunai](https://github.com/and3rson).

Source code for this service can be found here: <https://github.com/and3rson/raid>
