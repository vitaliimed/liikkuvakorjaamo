#!/usr/bin/env python3
"""Read-only diagnosis of website lead delivery; never prints raw errors or secrets."""
import argparse
import json
import subprocess
import urllib.error
import urllib.request


def gcloud(*args):
    result = subprocess.run(['gcloud', *args], capture_output=True, text=True)
    if result.returncode:
        raise RuntimeError('Команда gcloud не выполнена. Проверьте аккаунт и права доступа.')
    return result.stdout.strip()


def reason(message):
    value = message.lower()
    checks = [
        ('unauthorized', 'Apps Script отклонил токен: проверьте соответствие WEBHOOK_TOKEN.'),
        ('wrong site', 'Apps Script настроен для другого сайта.'),
        ('storage configuration error', 'Ошибка доступа или настройки таблицы в Apps Script.'),
        ('price snapshot', 'В заявке отсутствует сохранённая цена.'),
        ('missing submission', 'В заявке отсутствует ID для защиты от дублей.'),
        ('invalid helsinki', 'Apps Script отклонил формат даты.'),
        ('is empty', 'Не заполнена настройка интеграции.'),
        ('decode apps script', 'Вместо ожидаемого JSON получен другой ответ: проверьте публикацию Web App.'),
        ('status 403', 'Google отказал в доступе к Web App (403).'),
        ('status 404', 'Публикация Web App не найдена (404).'),
        ('timeout', 'Истекло время ожидания ответа.'),
        ('deadline exceeded', 'Истекло время ожидания ответа.'),
        ('busy', 'Apps Script занят; требуется повторная доставка.'),
    ]
    for needle, description in checks:
        if needle in value:
            return description
    return 'Есть ошибка; точный текст доступен в защищённой админке.' if message else 'Ошибка не записана.'


def scalar(fields, key, default='—'):
    value = fields.get(key, {})
    return value.get('stringValue', value.get('timestampValue', default))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--project', default='liikkuva-korjaamo')
    parser.add_argument('--region', default='europe-north1')
    parser.add_argument('--service', default='liikkuva-preview')
    args = parser.parse_args()
    service = json.loads(gcloud('run', 'services', 'describe', args.service,
                               '--project=' + args.project, '--region=' + args.region, '--format=json'))
    traffic = [r for r in service.get('status', {}).get('traffic', []) if r.get('percent', 0) > 0]
    if len(traffic) != 1 or not traffic[0].get('revisionName'):
        raise RuntimeError('Трафик распределён между ревизиями; нужна отдельная проверка каждой конфигурации.')
    revision = traffic[0]['revisionName']
    data = json.loads(gcloud('run', 'revisions', 'describe', revision,
                            '--project=' + args.project, '--region=' + args.region, '--format=json'))
    env = {v['name']: v for v in data['spec']['containers'][0].get('env', [])}
    print('Рабочая ревизия:', revision)
    for key in ('GOOGLE_APPS_SCRIPT_URL', 'GOOGLE_APPS_SCRIPT_TOKEN'):
        item = env.get(key, {})
        print(key + ':', 'Secret Manager' if item.get('valueFrom') else 'задано' if item.get('value') else 'НЕ ЗАДАНО')
    # Never silently query a different collection if its name is stored as a secret.
    for key in ('FIRESTORE_PROJECT_ID', 'FIRESTORE_LEADS_COLLECTION'):
        if env.get(key, {}).get('valueFrom'):
            raise RuntimeError('Имя проекта/коллекции задано через секрет; автоматическое чтение остановлено.')
    project = env.get('FIRESTORE_PROJECT_ID', {}).get('value') or args.project
    collection = env.get('FIRESTORE_LEADS_COLLECTION', {}).get('value') or 'leads'
    token = gcloud('auth', 'print-access-token')
    query = {'structuredQuery': {'from': [{'collectionId': collection}],
             'orderBy': [{'field': {'fieldPath': 'created_at'}, 'direction': 'DESCENDING'}], 'limit': 60}}
    url = f'https://firestore.googleapis.com/v1/projects/{project}/databases/liikkuvakorjaamo/documents:runQuery'
    request = urllib.request.Request(url, data=json.dumps(query).encode(),
                                    headers={'Authorization': 'Bearer ' + token, 'Content-Type': 'application/json'})
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            rows = json.load(response)
    except urllib.error.HTTPError as exc:
        raise RuntimeError(f'Firestore: HTTP {exc.code}. Проверьте права чтения базы.') from None
    documents = [row['document'] for row in rows if 'document' in row]
    print('Последних заявок:', len(documents))
    for doc in documents:
        fields = doc.get('fields', {})
        lead = fields.get('lead', {}).get('mapValue', {}).get('fields', {})
        print(json.dumps({'date': scalar(lead, 'created_at'), 'plate': scalar(lead, 'car'),
                          'sheets': scalar(fields, 'sheets_status'),
                          'processing': scalar(fields, 'processing_status'),
                          'email': scalar(fields, 'confirmation_email_status')}, ensure_ascii=False))
        if scalar(fields, 'sheets_status') != 'success':
            print('  Диагностика:', reason(scalar(fields, 'last_error', '')))
    print('Ничего не изменено. success означает ответ Apps Script, а не проверку видимости строки в таблице.')
    print('Если у нужной заявки success: проверьте фильтр листа и поиск по всей таблице.')


if __name__ == '__main__':
    try:
        main()
    except (RuntimeError, OSError, ValueError, KeyError) as exc:
        # Avoid leaking request URLs, tokens, raw provider errors or customer details.
        if isinstance(exc, RuntimeError):
            print('Остановлено:', str(exc))
        else:
            print('Остановлено: не удалось прочитать диагностические данные.')
        raise SystemExit(1)
