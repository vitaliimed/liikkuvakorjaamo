# Liikkuva CRM — Meta WhatsApp webhook (первый безопасный этап)

## Где расположен код

- `source/crm/whatsapp_webhook.go`: GET handshake, POST HMAC-SHA256, Firestore inbox, статусы.
- `source/crm/whatsapp_webhook_test.go`: проверки подлинности и ошибок.
- `source/crm/server.go`: публичный webhook до CRM session/CSRF; защищённый API владельца.
- `source/crm/web/app.js`: раздел «WhatsApp» только для владельца.

## Что сделано / что пока НЕ сделано

Сделано: приём подписанных Meta событий, идемпотентная запись в `crm_whatsapp_inbox` и `crm_whatsapp_status_events`, список последних 50 сообщений для владельца через `GET /crm/api/whatsapp-inbox`.

**Не сделано:** отправка сообщений клиентам, кнопки подтверждения владельца/механика, получение медиафайлов по API Meta, машинный перевод, привязка новых WhatsApp-диалогов к существующему заказу по верифицированному номеру телефона, миграция AutoHelp. Эти функции требуют отдельной реализации и тестов. Система НЕ отправляет сообщения автоматически.

Не назначать новые входящие номера случайным автомобилям по извлечённому госномеру. Перенос AutoHelp выполнять отдельной идемпотентной миграцией, не копировать клиентские данные в исходники и Git.

## Настройки Cloud Run

Секреты Secret Manager:
- `whatsapp-app-secret` -> переменная `WHATSAPP_APP_SECRET`;
- `whatsapp-verify-token` -> переменная `WHATSAPP_VERIFY_TOKEN`.

У service account `liikkuva-crm-runtime@liikkuva-korjaamo.iam.gserviceaccount.com` должны быть права `roles/secretmanager.secretAccessor` к двум конкретным секретам.

Webhook URL для нового независимого preview-тега:
`https://crm-wa-preview---liikkuva-preview-knrjxcx2bq-lz.a.run.app/crm/webhooks/whatsapp`

Публиковать только с `--no-traffic --tag=crm-wa-preview`, сохранять `CRM_MAIL_SEND_ENABLED=0`. Не обновлять production-трафик основного сайта. Провести `(cd source && go test ./...)` до развёртывания.

## Meta Developers

1. В приложении Liikkuva CRM Bot открыть настройки WhatsApp / Webhooks.
2. В Callback URL указать тегированный HTTPS-адрес выше.
3. Verify Token: фактическое секретное значение `whatsapp-verify-token` (не публиковать в переписке).
4. Subscribe to webhook field `messages`.
5. Прислать сообщение с тестового зарегистрированного получателя на тестовый номер Meta.
6. Проверить список в разделе «WhatsApp» только владельца CRM.
7. Проверить повторную доставку одного message ID: вторую запись не создаёт.
8. Для тестов исходящих сообщений нужен действующий Meta API access token и зарегистрированный отправитель; тестовый номер Meta ограничен списком тестовых получателей.

## Следующие этапы

A. Реальный номер компании и Meta access token в Secret Manager (без публикации токена в GitHub).

B. Сопоставление подтверждённого WhatsApp sender с существующим CRM пользователем и ролью, кнопки механика / владельца. Для сообщений вне сервисного окна Meta использовать одобренные шаблоны.

C. Согласие клиента, шаблон fi, контроль отправки владельцем, запись доставки, подтверждение от клиента, календарь, сообщение механику.

D. AI-перевод на русский с сохранением оригинала и разграничением доступа.

E. Безопасная миграция AutoHelp с дедупликацией автомобиля и историей по VIN/номеру, отдельным полем brand.
