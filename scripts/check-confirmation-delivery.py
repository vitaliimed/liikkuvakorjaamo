#!/usr/bin/env python3
"""Read-only mail diagnostics. Never prints full log payloads or credentials."""
import json
import subprocess
import sys

QUERY = ('resource.type="cloud_run_revision" '
         'resource.labels.service_name="liikkuva-preview" '
         '("confirmation email" OR "Confirmation email" OR "SMTP" OR "smtp")')

def classify(message):
    value = message.lower()
    if "525" in value or "unauthorized ip" in value:
        return "Почтовый провайдер отклонил IP отправителя. Проверьте разрешения SMTP в Brevo."
    if "535" in value or "authentication failed" in value:
        return "SMTP отклонил авторизацию. Проверьте действующий SMTP-ключ и имя пользователя."
    if "config is incomplete" in value:
        return "Не заполнены обязательные настройки SMTP."
    if "snapshot" in value or "quoted prices" in value:
        return "В заявке отсутствует сохранённая цена; проверьте тарифы и запись заявки."
    if "timeout" in value or "connection refused" in value:
        return "Ошибка соединения с почтовым сервером."
    if "failed" in value or "warning" in value or "not sent" in value:
        return "Есть ошибка отправки; точную причину смотрите в Cloud Logging (исходный текст скрыт)."
    return "Запись почтового обработчика; сама по себе не подтверждает доставку в почтовый ящик."

def main():
    result = subprocess.run(['gcloud','logging','read',QUERY,'--project=liikkuva-korjaamo',
                             '--freshness=7d','--limit=30','--format=json'],capture_output=True,text=True)
    if result.returncode:
        print('Не удалось прочитать журналы. Проверьте вход в gcloud и доступ к Cloud Logging.')
        return 1
    rows = json.loads(result.stdout)
    if not rows:
        print('За последние 7 дней подходящих записей не найдено. Это не подтверждает доставку писем.')
    for row in rows:
        message = row.get('textPayload') or json.dumps(row.get('jsonPayload',{}))
        print(row.get('timestamp',''), classify(message))
    print('Скрипт ничего не меняет и не отправляет письма. Для повторной отправки используйте заявку в админке.')
    return 0

if __name__ == '__main__':
    sys.exit(main())
