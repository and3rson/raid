*(Ukrainian version is [available here](/).)*

This API allows you to query air raid alerts in Ukraine in real-time.

Data is taken from <https://telegram.me/air_alert_ua>.

Events are usually delayed for up to 2 seconds.

Only regions are supported at this moment - 24 total plus Kyiv city. Crimea is absent from this list since no information is available. But we all know that Crimea is Ukraine.

Service works in two modes: HTTP and TCP.

You can also use our static map: <https://alerts.com.ua/map.png>

![Карта Тривог](/map.png)

## A. HTTP mode

### A1. Authentication

You will need a key to use this API.

  - To request a key, please send me an email (<a@dun.ai>) or ping me in Telegram ([\@andunai](https://t.me/andunai)).
  - Include the key with every request in `X-API-Key` header.
  - **When writing front-end code**: you'll need a [polyfill for EventStream](https://github.com/Yaffle/EventSource) since [browser's EventStream API](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events) does not allow sending headers with requests.

Please be aware that this API is rate-limited:

  - Max request rate from single address: 10 RPS
  - Max request rate per API key: 100 RPS

If you exceed the above limits you will be throttled with a HTTP 429 response.

### A2. Endpoints

#### `/api/states`

Returns the list of regions with their statuses.

```yaml
# $ curl https://alerts.com.ua/api/states -H "X-API-Key: yourApiKey34421337"

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
  ],
  "last_update": "2022-04-05T06:15:10.333210918+03:00"
}
```

#### `/api/state/<ID>`

Returns status for single region.

```yaml
# $ curl https://alerts.com.ua/api/states/12 -H "X-API-Key: yourApiKey34421337"

{
  "state": {
	"id": 12,
	"name": "Львівська область",
    "name_en": "Lviv oblast",
	"alert": false,
	"changed": "2022-04-05T06:13:12+03:00"
  },
  "last_update": "2022-04-05T06:15:10.333210918+03:00"
}
```

#### `/api/states/live`

[SSE](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events) endpoint which yields alert events in real time.

Client example: <https://codesandbox.io/s/goofy-elgamal-mkdkzv?file=/src/App.js>

```yaml
# $ curl https://alerts.com.ua/api/states/live -H "X-API-Key: yourApiKey34421337"

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

## B. TCP Mode

If you want to use this API in embedded systems - e.g. Arduino or ESP8266, you might prefer a more lightweight protocol instead of HTTP.
This is why we offer a simple TCP interface.

TCP-server is running on `tcp.alerts.com.ua` on port `1024`.

Example project for ESP8266: <https://wokwi.com/projects/330842127136195154>

### B1. Packet structure

All messages from server have the following format:

```sh
PacketType:Data\n
```

Every packet to and from server must end with an ASCII line break (`\n`).

| Packet type | Description                                                                | Data                                                                                                                 |
| :--------:  | :------------------------------------------------------------------------- | :------------------------------------------------------------------------------------------------------------------- |
| `a`         | auth packet, contains authentication result                                | `ok`, `timeout` or `wrong_api_key`                                                                                   |
| `p`         | ping packet, server sends this every 15 seconds                            | Random number in range [0;10000)                                                                                     |
| `s`         | state packet, contains information about air raid alert in specific region | Region number and air raid alert value. E.g. during air raid alert activation in Lviv region this will contain `12=1` |

### B2. Communication protocol

1. Client connects and sends its API key (ASCII encoding) within 3 seconds:

    ```
    yourApiKey34421337
    ```

    This is the only packet that client sends to the server.

    You can also request updates for a single region only by appending a comma-separated region number to your key, e. g.:

    ```
    yourApiKey34421337,12
    ```

2. Server sends auth packet which tells whether authentication was successful.

    ```
    a:ok
    ```

    If authentication has failed, error code will be provided instead of `ok` (see previous section).

3. Server initially sends 1 state packet for each region.

4. Server periodically sends ping packets (every 15 seconds).

5. During air raid alert activation or deactivation, server sends state packet.

Sample TCP session (prefix `>` means serverbound, `<` means clientbound, `#` denotes comments):

```js
> yourApiKey34421337     # Client sends API key
< a:ok                   # Authentication successful
< s:1=0                  # Initial data about 25 regions
< s:2=0
< s:3=0
...                      # (20 lines skipped for brevity)
< s:24=0
< s:25=0
< p:1241                 # Ping packet
< p:2508                 # ...
< p:1902
< p:9028
< s:12=1                 # Air raid alert in Lviv region!
< p:3819
< p:9873
< s:12=0                 # Air raid alert in Lviv region has been canceled.
< p:8321                 # Ping packet
< p:3985                 # ...
```

### Use the source, Luke

This thing was made by [Andrew Dunai](https://github.com/and3rson).

Source code for this service can be found here: <https://github.com/and3rson/raid>

### But why?

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
