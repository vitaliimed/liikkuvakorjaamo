# Liikkuva — инструкция Google Таблицы, Apps Script, ROAS/CR

Существующая таблица: https://docs.google.com/spreadsheets/d/14HHvj7skyve3K9ugFFgdlobD9De-2BICFdkSZdrVHKU/edit

## 1. Сохраните данные
Откройте таблицу, Файл → Создать копию (резервная копия). Рабочей остаётся существующая таблица. Новый Code.gs не создаёт таблицу заново, не удаляет старые заявки и не очищает доходы.

## Если Apps Script показывает This app is blocked (12.11.2026)

Здесь две независимые проблемы: `spreadsheets.currentonly` недостаточен для `SpreadsheetApp.openById(SHEET_ID)`; требуется полный `https://www.googleapis.com/auth/spreadsheets`. После исправления Ctrl+S и обновите редактор.

Но если Google всё ещё блокирует OAuth даже с полным scope, это может быть ограничение Google OAuth на Apps Script Default-проектах, о котором сообщили в Google Issue Tracker в октябре 2026: https://issuetracker.google.com/issues/571856421 и https://issuetracker.google.com/issues/572515608. Смена scope сама по себе может не помочь.

### Подключение стандартного Cloud Project (официальная поддерживаемая настройка)

1. В Google Cloud Console выберите **существующий** проект `liikkuva-korjaamo` и откройте Google Auth Platform.
2. Заполните Branding (название Liikkuva Sheets, support email) и Audience: **External / Testing**, внесите свой Google-аккаунт в Test users.
3. Получите **Project number**, а не Project ID: `gcloud projects describe liikkuva-korjaamo --format='value(projectNumber)'`.
4. В Apps Script: Project Settings → GCP project → **Change project**. Введите **числовой** номер и Set project. Примечание: обратно на Default project переключиться нельзя, может потребоваться повторное разрешение OAuth. Предварительно настройте Google Auth Platform.
5. Сохраните manifest с полным spreadsheets scope, обновите редактор, запустите `setupExistingLiikkuva` и предоставьте права только своему скрипту, если Google разрешит.
6. Если Google продолжает показывать блокировку, не обходите её. Проверьте настройки Google Auth Platform, разрешения аккаунта и статус ошибки в Google Issue Tracker.

Документы: https://developers.google.com/apps-script/guides/cloud-platform-projects; https://developers.google.com/apps-script/guides/services/authorization; https://developers.google.com/apps-script/reference/spreadsheet/spreadsheet-app#openById(String)

Пока setupExistingLiikkuva не открыл таблицу успешно, свойства SPREADSHEET_ID / SITE_DOMAIN / WEBHOOK_TOKEN не создадутся. Публичный GET /exec сам по себе не доказывает работу записи заявок.

## 2. Откройте Apps Script из таблицы
В рабочем документе: Расширения → Apps Script. Если прежний самостоятельный редактор показывает This app is blocked, создайте отдельный проект через таблицу, чтобы избежать испорченной OAuth-конфигурации старого проекта. В новом редакторе удалите пример myFunction и вставьте ВЕСЬ код из apps-script/Code.gs этой GitHub ветки. Сохраните.

В проекте Apps Script в Project settings можно включить показ appsscript.json; его эталон находится рядом с Code.gs. Он запрашивает право Google Sheets, не доступ к Gmail/Drive.

Выберите функцию setupExistingLiikkuva → Run. При успешном разрешении Google создаются месячные вкладки и вкладка Доход текущего года, данные не стираются.

Если Google снова сообщает This app is blocked, не пытайтесь обойти защиту; проверьте Project Settings → Google Cloud Project и настройки OAuth (тип пользователя, тестовые пользователи, верификация/ограничения Workspace). Старое приложение с кастомным Cloud project может требовать настройку OAuth. Предоставляйте права только собственному проверенному скрипту.

## 3. Получите НАСТОЯЩИЙ /exec
Apps Script → Deploy → New deployment → Web app.
Execute as: Me. Who has access: Anyone (Cloud Run POST без входа в Google).
Нажмите Deploy и сохраните Web app URL вида:
https://script.google.com/macros/s/ВАШ_ID_РАЗВЕРТЫВАНИЯ/exec

Адрес script.google.com/home/projects/.../edit — это только редактор. Он НЕ подойдёт для Cloud Run.
При изменениях кода: Deploy → Manage deployments → Edit → New version → Deploy. Простое сохранение кода не обновляет /exec.

Откройте /exec в браузере: должен прийти JSON с ok:true и site:liikkuvakorjaamo.fi. Это только GET, не доказательство получения заявки.

## 4. Свяжите с Cloud Run
В Project Settings → Script properties найдите WEBHOOK_TOKEN. Он создан функцией setupExistingLiikkuva один раз. Никому не присылайте его и не храните в GitHub.

В Cloud Shell:
cd ~/liikkuva-production-20261112
python3 scripts/activate-leads.py --stage-only

На приглашение Google Apps Script Web App URL вставьте /exec. На скрытый запрос токена введите значение WEBHOOK_TOKEN из свойств скрипта. Скрипт подготовит Bootstrap=0 на НУЛЕВОМ трафике, проверяя Firestore, Cloud Tasks и остальную конфигурацию. До проверки собственной тестовой заявки не переводите трафик.

## 5. Как использовать вкладку Доход 2026

В месячной вкладке, например 2026-11:
A — Статус: В процессе / Успешный / Спам / Отказ.
B — Доход, €: фактически полученная сумма заказа.
C — За что получили: назначение платежа.
W — Канал: Реклама / Органика / Не определён.

UTM medium cpc/paid_social и явные метки google_ads/meta_ads создают рекламный канал. При отсутствии UTM считаем канал неопределённым; W можно отредактировать вручную.

В годовой вкладке Доход 2026 ВРУЧНУЮ вносите только:
D2:D13 — сумма расходов Google Ads + Meta Ads за соответствующий месяц, €.
E2:E13 — количество рекламных кликов за месяц.

Расчётные колонки:
B — общий доход успешных заказов.
C — успешные заказы.
F — рекламные заявки (кроме спама).
G — успешные рекламные заказы.
H — доход от успешных рекламных заказов.
I — ROAS = H / D (в разах, например 3,00×).
J — CR = F / E (процент перехода к заявке).
K — CPL = D / F.
L — CPA = D / G.
M — закрытие заявок = G / F.
Строка 15 содержит суммы и взвешенный общий ROAS/CR.

Пример: бюджет 200 €, 100 кликов, 10 рекламных заявок и 600 € рекламного дохода → CR 10%, ROAS 3,0×. Данные кликов/бюджета не вычисляются из заявки — вводятся из Ads кабинетов или позже через API.

## 6. Диагностика предыдущей ошибки тестов
Команда stage раньше скрывала текст ошибки go test. Теперь выводит его.

cd ~/liikkuva-production-20261112
git pull --ff-only origin release/ux-crm-brevo-20261112
cd source
go test -count=1 ./...

Пока Go-тесты не пройдут, это НЕ готовый релиз сайта. Не отправляйте в чат ключи и приватные токены.

Официальные документы:
https://developers.google.com/apps-script/guides/web
https://developers.google.com/apps-script/guides/bound
https://developers.google.com/apps-script/guides/services/authorization
https://developers.google.com/apps-script/guides/cloud-platform-projects
