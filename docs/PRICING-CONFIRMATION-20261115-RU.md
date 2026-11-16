# Liikkuva — цены, подтверждение заявки и ожидание механика

Версия: 2026-11-15
Ветка: `fix/price-copy-arrival-20261115` (основана на опубликованной `fix/post-launch-ux-crm-20261114`).
На основной домен **не публиковать без `stage` и проверки**.

## Единая бизнес-логика

- **Выезд механика от 75 €**: только прибытие к месту автомобиля. Диагностика, мелкий ремонт и запчасти в цену выезда **не входят**.
- **Диагностика и подходящий мелкий ремонт от 60 €**: отдельная плата **дополнительно** к стоимости выезда; возможный мелкий ремонт зависит от неисправности, безопасности и условий.
- Более сложные работы, дополнительная работа и запчасти: согласовываются и оцениваются **до выполнения**.
- Отправка заявки не означает согласованный визит. Время устанавливается отдельно через сотрудника/механика. Только **после подтверждения** времени клиент ожидает механика **по адресу/на точке, указанной в заявке, в согласованное время**; должен быть обеспечен доступ к автомобилю.
- Базовая стоимость изменяется через `/admin?tab=general`: `ProVisitEUR` и `ProDiagnosticEUR`. В новых заявках обе цены записываются как `VisitPriceEUR` и `DiagnosticFromEUR`, а в email используется сохранённый на момент заявки тариф; старые заявки не пересчитываются задним числом. Fallback по диагностике — 60 €.
- Значения с пометкой «от / alkaen / from» — стартовые, не гарантированная итоговая сумма.

## Финский: подтверждение заявки

Hei [Asiakkaan nimi],

Kiitos pyynnöstäsi. Olemme vastaanottaneet sen onnistuneesti.

**Automekaanikon käynti alkaen [käyntihinta] €.** Käyntimaksu kattaa vain mekaanikon saapumisen sovittuun paikkaan. Se ei sisällä diagnostiikkaa eikä korjausta.

**Vianmääritys ja pienet korjaustyöt alkaen [diagnoosihinta] €.** Ne veloitetaan erikseen käyntimaksun lisäksi. Hinta sisältää vianmäärityksen ja tilanteeseen sopivan pienen korjauksen, mikäli se on mahdollista.

Varaosat, vaativammat korjaukset ja lisätyöt hinnoitellaan ja sovitaan erikseen ennen työn aloittamista.

Otamme sinuun yhteyttä sopiaksemme käyntiajan. **Kun aika on vahvistettu, odota mekaanikkoa ilmoittamassasi paikassa sovittuna ajankohtana.** Varmista, että autolle on esteetön pääsy.

Ystävällisin terveisin,
Liikkuva Auto Korjaamo

## Русский перевод (для внутренней проверки, клиентам отправляется FI/EN)

Здравствуйте, [Имя]!

Спасибо за заявку. Мы успешно её получили.

**Выезд мастера — от [цена выезда] €.** Эта сумма покрывает только приезд мастера по согласованному адресу. Диагностика и ремонт не входят в стоимость выезда.

**Диагностика и мелкий ремонт — от [цена диагностики] €.** Оплачиваются отдельно, дополнительно к выезду. В стоимость входит диагностика и подходящий мелкий ремонт, если его возможно безопасно выполнить на месте.

Запчасти, дополнительные и более сложные работы оцениваются и согласовываются до начала работ.

Мы свяжемся с вами для согласования времени. **Когда время будет подтверждено, ожидайте мастера на месте, указанном в заявке, в согласованное время.** Пожалуйста, обеспечьте доступ к автомобилю.

С уважением,
Liikkuva Auto Korjaamo

## Изменено в коде
- `source/services_email.go`: HTML и plain-text FI/EN, цены из сохранённого Lead; не обещаем приезд до согласования.
- `source/texts.go`: преимущества, краткое описание тарифов, FAQ, этапы, Thank You FI/EN.
- `source/templates/landing.html`, `source/landing.html`: тарифы и выделенная памятка ожидания; второй такой же текст у нижней формы.
- `source/static/css/footer-form.css`: заметная памятка для light/dark.
- `source/static/seo/hinnasto.html`, `source/static/seo/vikadiagnostiikka.html` и соответствующие `assets/static` копии: обновлённые цены и понятные условия.
- `source/liikkuva_pro.go`: обновление отображения цены при изменениях в админке, по умолчанию диагностика 60 €.
- `source/handlers_admin.go`: дефолт диагностики 60 €; на форме «Выезд мастера от» и «Диагностика и мелкий ремонт от».
- `source/pricing_copy_test.go`: регрессионные тесты сохранённых цен в email, динамических тарифов сайта и ожидания после подтверждения времени.
- `scripts/release-ux-20261112.py`: отдельный no-traffic preview tag `pricing-copy-preview` и состояние `~/.liikkuva-releases/pricing-copy-release.json`.

## Проверки
```bash
cd ~/liikkuva
git fetch origin
git switch --track -c fix/price-copy-arrival-20261115 origin/fix/price-copy-arrival-20261115
python3 scripts/release-ux-20261112.py stage
python3 scripts/release-ux-20261112.py status
```
Если локальная ветка уже есть, используйте `git switch fix/price-copy-arrival-20261115 && git pull --ff-only origin fix/price-copy-arrival-20261115`.

1. В preview подтвердить HTML FI/EN (вход на `/` и `/en`), тарифы/FAQ, `/hinnasto`, `/vikadiagnostiikka`, экран Thank You и обе формы.
2. Изменить цену **только в разрешённом тестовом окружении** или проверить helper через `go test`: новые суммы показываются динамически, цены в сохранённой заявке остаются первоначальными.
3. Проверить письмо на тестовую собственную почту только после корректной SMTP-конфигурации; отправленные ранее клиентские письма не переписываются и повторная рассылка не запускается.
4. При `PREVIEW VERIFIED` выполнить `python3 scripts/release-ux-20261112.py publish`; скрипт переключит 100% только после проверки, иначе предпримет откат. Команда `rollback` сохранена.
