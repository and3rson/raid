*(English version is [available here](/en).)*

Цей API дозволяє вам отримувати інформацію про повітряні тривоги в Україні в режимі реального часу.

Наше джерело даних - Telegram-канал [\@air_alert_ua](https://telegram.me/air_alert_ua).

Події в середньому затримуются до 2-х секунд.

Зараз надаємо інформацію лише про області (24 області та м. Київ). Крим відсутній зі списку, оскільки по ньому відсутня інформація. Але ми всі знаємо, що Крим - це Україна.

## Автентифікація

Вам потрібно ключ для роботи з цим API.

  - Щоб отримати ключ, надішліть мені e-mail (<a@dun.ai>) або повідомлення в Telegram ([\@andunai](https://t.me/andunai)).
  - Надсилайте ключ в кожному запиті в заголовку `X-API-Key`.
  - **Для фронт-ендерів**: вам знадобиться [polyfill для EventStream](https://github.com/Yaffle/EventSource), оскільки [API EventStream в браузерах](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events) не підтримує надсилання заголовків в запиті.

Зверніть увагу: це API має обмеження по частоті запитів. Якщо ви будете спамити зі швидкістю більше ніж 10 запитів на секунду, ви отримаєте HTTP 429.

## Ендпоїнти

### `GET /api/states`

Повертає список областей з їхніми статусами.

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

### `GET /api/states/live`

[SSE](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events)-ендпоінт, який генерує події в режимі реального часу.

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

Цю штуку зробив [Andrew Dunai](https://dun.ai).

Початковий код знаходиться тут: <https://github.com/and3rson/raid>

## Але навіщо?

Я є прихильником відкритих даних та вільного програмного забезпечення.

Я вважаю, що будь-яку **інформацію, яка є відкритою,** будь-хто повинен мати змогу **опрацьовувати так, як йому заманеться**.

> "*Але ж... Хіба "безкоштовний" і "вільний" - не те саме?*"

"Безкоштовний" (*"gratis"*) не є синонімом для "вільний" (*"libre"*).

Для прикладу, Інстаграм - безкоштовний, але він не є вільний: вони змушують вас використовувати
саме їх власний додаток і не дають повного контролю над своїми даними та доступом до них.
Точніше, дають, але дуже обмежений доступ.
Це і означає термін "невільний" в контексті комп'ютерних технологій.

Не ставайте рабами постачальників.

Давайте зробимо світ вільнішим.

\*stallman.jpg\*
