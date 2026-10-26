import re
PHRASES='''Landing admin|Управление сайтом
Admin login|Вход в управление
Login|Войти
Logout|Выйти
Password|Пароль
Save settings|Сохранить настройки
Saved successfully.|Настройки сохранены.
General|Основные настройки
Assets|Изображения и оформление
Landing FI|Главная — финский
Landing EN|Главная — английский
Form FI|Форма — финский
Form EN|Форма — английский
Form — Finnish|Форма — финский
Form — English|Форма — английский
Leads|Заявки
Lead|Заявка
Lead details|Детали заявки
Selected request|Выбранная заявка
Services cards|Карточки услуг
Testimonials|Отзывы
Testimonial|Отзыв
Review moderation|Модерация отзывов
Customer review moderation|Отзывы клиентов
Cases|Примеры работ
Thank-you|После отправки заявки
Privacy|Конфиденциальность
SEO & robots|SEO и индексация
Sections|Разделы
Tracking & consent|Аналитика, реклама и согласие
Business details|Реквизиты компании
Business name|Название компании
Y-tunnus|Y-tunnus (регистрационный номер)
Phone|Телефон
Email|Электронная почта
Name|Имя
Car|Автомобиль
Address|Адрес
Address hint|Как найти автомобиль
Message|Сообщение
Service email|Почта для связи
Base city|Основной город
Header CTA|Кнопка в шапке
Social links|Социальные сети
Hide / show landing screens|Показывать и скрывать блоки
Section visibility|Видимость блоков
All|Все
New|Новая
Pending|Ожидает
Ready|Готово
Completed|Завершено
Cancelled|Отменено
Contacted|Связались
Scheduled|Выезд назначен
In work|В работе
Waiting parts|Ожидаем запчасти
Approved|Опубликовано
Rejected|Отклонено
Approve and publish|Одобрить и опубликовать
Reject|Отклонить
Delete|Удалить
Remove|Удалить
Open|Открыть
Open →|Открыть →
Open lead|Открыть заявку
Open map|Открыть карту
Call client|Позвонить клиенту
Email client|Написать клиенту
Copy phone|Копировать телефон
Copy email|Копировать почту
Copy map URL|Копировать ссылку на карту
Retry processing|Повторить обработку
Update status|Обновить статус
Send custom email|Отправить письмо
Send ready email|Сообщить о готовности
Send vehicle ready|Сообщить о готовности автомобиля
Send request confirmation|Отправить подтверждение заявки
Send review invitation|Попросить отзыв
Ask for a review|Попросить отзыв
Subject|Тема письма
To|Кому
Custom email message|Текст письма клиенту
Client work|Работа с клиентом
Client stage|Этап работы
Client status|Статус клиента
Created|Создано
Updated at|Обновлено
Processed at|Обработано
Task enqueued at|Поставлено в очередь
Last error|Последняя ошибка
Technical details|Технические сведения
Delivery|Доставка уведомлений
Processing status|Статус обработки
Telegram status|Статус Telegram
Sheets status|Статус таблицы
Confirmation email|Подтверждение заявки
Vehicle ready email|Письмо о готовности
Review invitation|Приглашение оставить отзыв
Review message|Текст отзыва
Language|Язык
Source|Источник
Photos|Фотографии
Map visual preview|Предпросмотр карты
Current logo preview|Текущий логотип
Main texts|Основные тексты
Hero|Первый экран
Final CTA|Заключительный призыв
How it works|Как это работает
Process|Этапы работы
Services|Услуги
Promo|Предложение
Promo block|Блок предложения
Enable promo block|Включить блок предложения
Stats strip|Блок показателей
Areas / map|Зона обслуживания и карта
On-site / Workshop block|Выезд и работа в мастерской
Labels|Подписи полей
Placeholders|Подсказки внутри полей
Validation and system messages|Проверка данных и системные сообщения
Buttons, helper texts and upload|Кнопки, пояснения и фотографии
Save stays visible|Сохранение всегда под рукой
Less scrolling, clearer editing and a save button that stays visible.|Разделы сайта, заявки и настройки. Кнопка сохранения всегда доступна.
Enter the admin password to manage content and leads.|Введите пароль для управления сайтом и заявками.
Tabs reduce scrolling. The last opened tab is remembered in your browser.|Выберите раздел. Последний открытый раздел сохраняется в браузере.
Use tabs for faster editing. Your last tab is remembered.|Используйте меню разделов для быстрого перехода.
No leads found.|Заявок пока нет.
No requests in this stage.|На этом этапе заявок нет.
No reviews found for this filter.|Нет отзывов с выбранным статусом.
Client work status was updated.|Статус работы обновлён.
Lead was re-queued successfully.|Заявка снова поставлена в очередь.
Ready email was sent successfully.|Письмо о готовности отправлено.
Request confirmation email was sent successfully.|Подтверждение заявки отправлено.
Review invitation was sent successfully.|Приглашение оставить отзыв отправлено.
Automatic client email is disabled. Configure SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD and MAIL_FROM in Cloud Run.|Отправка писем отключена. Настройте SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD и MAIL_FROM в Cloud Run по инструкции EMAIL-RU.md.
Basic business contact details and header/brand settings. Logo replacement is handled here by URL or static path.|Контакты, реквизиты и оформление шапки. Логотип задаётся ссылкой или путём к файлу.
You can add any number of testimonials. Empty rows are ignored on save.|Добавляйте только реальные отзывы. Пустые карточки не сохраняются.
Reviews submitted through a personal client link appear here first. Only approved reviews are published on the matching FI or EN landing page.|Отзывы по личной ссылке клиента сначала попадают сюда. На сайте публикуются только одобренные отзывы на соответствующем языке.
Each testimonial needs a client name and at least one quote language. Photo is optional. Rating defaults to 5 if empty or invalid.|Укажите имя клиента, реальный текст отзыва и оценку. Фотография необязательна.
Open a request, contact the client and move the job through simple work stages. The selected request is shown first on mobile.|Откройте заявку, свяжитесь с клиентом и обновляйте этап работы. На телефоне выбранная заявка показывается первой.
Uncheck a screen to hide it from the public landing page. The content stays saved in admin and can be shown again later.|Снимите отметку, чтобы скрыть блок сайта. Содержимое сохраняется и может быть показано снова.
Landing SEO, robots meta tags and robots.txt in one place.|Заголовки и описания для поиска, настройки индексации и robots.txt.
Texts and SEO for the success page after form submission.|Тексты страницы после успешной отправки заявки и её SEO-настройки.
All privacy content is grouped here to avoid endless scrolling on one page.|Текст политики конфиденциальности. Проверьте реквизиты оператора и фактически используемые сервисы.
Write plain text. Liikkuva adds the email design automatically.|Введите обычный текст. Оформление письма добавится автоматически.
'''
D=dict(x.split('|',1) for x in PHRASES.strip().splitlines())
WORDS={'Header':'Шапка','CTA':'кнопка','Primary':'Основная','Secondary':'Дополнительная','Final':'Заключительный','Hero':'Первый экран','subtitle':'подзаголовок','title':'заголовок','text':'текст','kicker':'надзаголовок','Badge':'Метка','badge':'метка','Button':'Кнопка','button':'кнопка','Price':'Цена','Tag':'Метка','Quote':'Текст отзыва','Rating':'Оценка','Section':'Раздел','Services':'Услуги','Form':'Форма','intro':'введение','Intro':'Введение','Landing':'Главная','description':'описание','Page':'Страница','page':'страница','Thank-you':'После заявки','Privacy':'Конфиденциальность','Promo':'Предложение','Process':'Этапы','Name':'Имя','name':'имя','Email':'Почта','email':'почта','Phone':'Телефон','phone':'телефон','Address':'Адрес','Car':'Автомобиль','Message':'Сообщение','label':'подпись','placeholder':'подсказка','helper':'пояснение','required':'обязательно','invalid':'некорректно','Submit':'Отправка','loading':'загрузка','Success':'Успех','Error':'Ошибка','Network':'Сеть','Geo':'Геолокация','added':'добавлена','failed':'ошибка','unsupported':'не поддерживается','Photos':'Фотографии','photos':'фотографий','hint':'подсказка','Upload':'Загрузка','Use':'Использовать','location':'местоположение','Pick':'Выбор','map':'карта','Map':'Карта','point':'точка','Point':'Пункт','Points':'Пункты','first':'сначала','Wrong':'Неверный','file':'файл','type':'тип','size':'размер','error':'ошибка','Contact':'Контакт','Registration':'Госномер','Photo':'Фото','Image':'Изображение','image':'изображение','background':'фон','person':'мастер','preview':'предпросмотр','desktop':'на компьютере','Service':'Услуга','service':'услуга','card':'карточка','Case':'Работа','Client':'Клиент','client':'клиент','Current':'Текущий','Logo':'Логотип','logo':'логотип','path':'путь','or':'или','Working':'Рабочие','hours':'часы','number':'номер','Main':'Основной','content':'содержание','Finnish':'финский','English':'английский','Full':'Полный','confirmation':'подтверждение','Requests':'Заявки','statuses':'статусы','retry':'повтор','Open':'Открыть','Call':'Позвонить','pending':'ожидает','approval':'одобрения','Status':'Статус','status':'статус','sent':'отправлено','at':'в','Confirmation':'Подтверждение','Review':'Отзыв','invitation':'приглашение','invite':'приглашение','Ready':'Готовность','Show':'Показать','details':'подробности','New':'Новая','request':'заявка','contacted':'связались','Visit':'Выезд','scheduled':'назначен','Work':'Работа','in':'в','progress':'процессе','Waiting':'Ожидание','for':'для','parts':'запчасти','pickup':'выдача','Completed':'Завершено','Add':'Добавить','case':'работу','testimonial':'отзыв','Latitude':'Широта','Longitude':'Долгота','Location':'Местоположение','source':'источник','Social':'Социальные','links':'ссылки','Too':'Слишком','many':'много','Tracking':'Аналитика','consent':'согласие'}
def tr(t):
 if t.strip() in D:return t.replace(t.strip(),D[t.strip()])
 # Translate short static labels compositionally; preserve technical IDs and template values.
 if len(t.strip())<=95:return re.sub(r'[A-Za-z]+',lambda m:WORDS.get(m[0],m[0]),t)
 return t

def translate(s):
 chunks=re.split(r'(<script\b.*?</script>|<style\b.*?</style>|<code\b.*?</code>|{{.*?}}|<[^>]*>)',s,flags=re.S)
 for i,c in enumerate(chunks):
  if c.startswith('<script'):
   for k,v in D.items():
    for quote in ['"',"'"]:c=c.replace(quote+k+quote,quote+v+quote)
   chunks[i]=c
  elif c and not c.startswith(('<','{{')):chunks[i]=tr(c)
 return ''.join(chunks)
D.update(dict(x.split('|',1) for x in '''Open FI landing|Открыть сайт — FI
Open EN landing|Открыть сайт — EN
Thank-you FI|После заявки — FI
Privacy FI|Конфиденциальность — FI
Phone, header CTA, logo|Телефон, кнопка, логотип
Favicons, hero, map, service images|Иконки, первый экран, карта, услуги
Main content in Finnish|Основной текст на финском
Main content in English|Основной текст на английском
Finnish form texts and validation|Тексты и проверка формы на финском
English form texts and validation|Тексты и проверка формы на английском
Future offers and campaign block|Акции и рекламные предложения
Client name, quote, photo|Имя клиента, отзыв, фотография
Many cards, site scrolls after 6|Карточки с прокруткой на сайте
FI and EN confirmation page|Подтверждение заявки на FI и EN
Full privacy page text|Политика конфиденциальности
Meta tags and robots.txt|Метатеги и robots.txt
Requests, statuses, retry|Заявки, этапы и повторная обработка
Add all services you need. For a clean mobile landing page, the public page automatically keeps the first view compact: 6 cards are visible on desktop, the rest can be scrolled.|Добавьте услуги. На широком экране сначала видны 6 карточек, остальные доступны прокруткой.
Cases should prove real situations: no-start, warning light, brake noise, battery, urgent inspection. Add many cases here; the public page keeps the first screen compact and scrollable.|Добавляйте реальные примеры работ: проблемы запуска, аккумулятора, тормозов, сигналы на панели. Карточки доступны прокруткой.
Fast asset setup by URL. Today this is enough to build the new hybrid hero and visual map section without waiting for file upload endpoints.|Укажите ссылку на изображение или путь внутри /static/img/. Рекомендуются сжатые WebP и JPG.
If enabled and the title is filled, the promo block appears under the hero section. The button uses|Если блок включён и заголовок заполнен, предложение показывается под первым экраном. Адрес кнопки по умолчанию:
when no custom URL is given.|если собственная ссылка не указана.
Manage all visible form texts, helper texts and validation messages for the english landing page.|Тексты, подсказки и сообщения проверки формы на английском языке.
Manage all visible form texts, helper texts and validation messages for the finnish landing page.|Тексты, подсказки и сообщения проверки формы на финском языке.
Manage any number of case cards. The public site shows 6 cards at once and lets visitors scroll the rest.|Добавляйте карточки примеров работ. Остальные карточки доступны прокруткой.
Manage any number of service cards. The public site shows 6 cards at once and lets visitors scroll the rest.|Добавляйте карточки услуг. Остальные карточки доступны прокруткой.
Optional. If filled together with Google Ads Tag ID, successful form submissions send a Google Ads conversion event.|Необязательно. Вместе с Google Ads Tag ID включает конверсию после сохранения заявки при согласии на рекламу.
Optional. If filled, successful form submit triggers LinkedIn event-specific conversion tracking.|Необязательно. ID конверсии LinkedIn для сохранённой заявки при согласии на рекламу.
Optional. Loads only after marketing consent and tracks PageView + Lead on successful form submit.|Загружается при согласии на рекламу. События PageView и Lead после сохранения заявки.
Optional. Loads only after marketing consent and tracks PageView + SubmitForm on successful form submit.|Загружается при согласии на рекламу. События PageView и SubmitForm после сохранения заявки.
Optional. Loads the LinkedIn Insight Tag only after marketing consent.|Необязательно. LinkedIn Insight Tag загружается при согласии на рекламу.
Optional. Main analytics ID for GA4. Consent Mode will control analytics_storage for this tag.|ID счётчика GA4. Сбор аналитики зависит от выбора посетителя в настройках cookies.
Optional. Recommended if you use Google Ads audiences or direct Ads conversion tracking.|ID Google Ads (AW-…). Заполните для рекламных аудиторий и прямой передачи конверсий.
Optional. Use optimized WebP/JPG. If empty, the card is still shown without image.|Необязательно. Используйте сжатый WebP или JPG. Без ссылки карточка показывается без изображения.
Recommended for the later “coverage / how it works” section. Use a light SVG or optimized WebP instead of a live map background.|Изображение зоны обслуживания. Используйте компактный SVG или WebP.
The site now uses Consent Mode style behavior for Google tags and a design-matched consent banner. GA4 and Google Ads use consent states. Meta, LinkedIn and TikTok load only after marketing consent. All configured tags are rendered on landing, thank-you and privacy pages from the same settings source.|Настройки общие для страниц сайта. Consent Mode v2 учитывает выбор посетителя. Meta, LinkedIn и TikTok загружаются при согласии на рекламу. Конверсия отправляется после сохранения заявки; данные формы в аналитику не передаются.
Use a clean PNG or WebP with transparent background if possible. This image is shown on desktop only, between the text and the form.|Для дополнительного изображения мастера используйте PNG или WebP. Оно показывается на широком экране.
Use the same visual format for all service images. Recommended: 1200×900, same color grading, same camera style.|Для услуг используйте единый стиль изображений. Рекомендуемый размер — 1200×900.
Optional. Keep it dark and lightweight. The page still works without this image.|Необязательное изображение. Используйте сжатый файл; страница работает и без него.
If the field is empty, the icon will not be shown in the footer or thank-you page.|Если ссылка не задана, иконка внизу сайта и на странице подтверждения скрыта.
Main landing copy in English.|Основной текст сайта на английском языке.
Main landing copy in Finnish.|Основной текст сайта на финском языке.
Logo preview could not be loaded from this path.|Не удалось загрузить логотип по указанному пути.
Hero background preview is empty or could not be loaded from this URL.|Фоновое изображение не задано или недоступно по ссылке.
Hero person preview could not be loaded from this URL.|Не удалось загрузить изображение мастера.
Map preview is empty or could not be loaded from this URL.|Изображение карты не задано или недоступно.
Production rule|Правило публикации
How this works now|Как работает раздел
Prepared for future offers, campaign pushes and seasonal messages on the landing page.|Акции, сезонные предложения и рекламные сообщения на сайте.
You can use an anchor like|Можно указать якорь, например
You can use|Можно использовать
Use a static path like|Укажите путь к файлу, например
or a full public URL.|или полную общедоступную ссылку.
, an internal path or a full external URL.|, внутренний путь или полную внешнюю ссылку.
in the text. The server will replace it with the configured site URL when serving robots.txt.|в тексте. Сервер заменит его адресом сайта в robots.txt.
'''.strip().splitlines()))
WORDS.update({'landing':'сайт','robots':'индексация','meta':'метатег','text':'текст','Cards':'Карточки','cards':'карточки','strip':'блок','slots':'файлы','screens':'блоки','Header':'Шапка','Reverse':'Обратное','geocode':'геокодирование','Phone':'Телефон','hint':'подсказка','Contact':'Контакт','Preview':'Предпросмотр','Favicons':'Иконки','favicons':'иконки','icons':'иконки','touch':'касания','Apple':'Apple','app':'приложения','Ask':'Попросить','review':'отзыв','form':'форма','texts':'тексты','validation':'проверка','buttons':'кнопки','Upload':'Загрузить'})
