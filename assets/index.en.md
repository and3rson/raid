*(Ukrainian version is [available here](/).)*

This API allows you to query air raid alerts in Ukraine in real-time.

Data is taken from <https://telegram.me/air_alert_ua>.

Events are usually delayed for up to 2 seconds.

Only states are supported at this moment - 24 total plus Kyiv city. Crimea is absent from this list since no information is available. But we all know that Crimea is Ukraine.

## Authentication

You will need a key to use this API.

  - To request a key, please send me an email (<a@dun.ai>) or ping me in Telegram ([\@andunai](https://t.me/andunai)).
  - Include the key with every request in `X-API-Key` header.
  - **When writing front-end code**: you'll need a [polyfill for EventStream](https://github.com/Yaffle/EventSource) since [browser's EventStream API](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events) does not allow sending headers with requests.

Please be aware that this API is rate-limited. If you spam more than 10 RPS, you will be throttled with a HTTP 429 response.

## Endpoints

### `/api/states`

Returns the list of states with their statuses.

```yaml
# $ curl https://alerts.dun.ai/api/states -H "X-API-Key: yourApiKey34421337"

{
  "states": [
	{
	  "id": 1,
	  "name": "Вінницька область",
      "name_en": "Vinnytsia oblast",
	  "alert": false,
	  "changed": "2022-04-05T06:12:52+03:00"
	},
	{
	  "id": 2,
	  "name": "Волинська область",
      "name_en": "Volyn oblast",
	  "alert": false,
	  "changed": "2022-04-05T06:13:06+03:00"
	},
	# ...
  ]
}
```

### `/api/states/live`

[SSE](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events) endpoint which yields alert events in real time.

```yaml
# $ curl https://alerts.dun.ai/api/states/live -H "X-API-Key: yourApiKey34421337"

event: hello
data: null

event: ping
data: null

event: ping
data: null

event: update
data: {"state":{"id":12,"name":"Львівська область","name_en":"Lviv oblast","alert":false,"changed":"2022-04-05T06:14:56+03:00"}}

event: ping
data: null

# ...
```

## Use the source, Luke

This thing was made by [Andrew Dunai](https://github.com/and3rson).

Source code for this service can be found here: <https://github.com/and3rson/raid>

## But why?

I support and preach the principles of open data and FOSS.

I believe that everyone should be allowed to process any **information which is publicly available** in any ways they choose.

> "*But... Doesn't "free" mean "free of charge"? Isn't free and "libre" the same?*"

"Free" (as in beer) and "free" (as in freedom, also called "libre") are totally different concepts.

For example, Instagram is free of charge. However it's not freedom: they force you to use
their own application and refuse to provide you full access over your data.
In fact, they give you some control but it's very limited and heavily supervised.
This is what "non-free" means in the context of computer technologies.

Don't become vendor-locked.

Let's make our world libre.

\*stallman.jpg\*
