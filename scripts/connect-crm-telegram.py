#!/usr/bin/env python3
"""Connect the existing shared bot. Secrets never enter argv or console output.

Read-only unless --apply is supplied. Run after publishing the CRM release.
"""
import argparse
import datetime
import json
import os
from pathlib import Path
import re
import secrets
import subprocess
import sys
import urllib.error
import urllib.request


class SetupError(Exception):
    pass


def cloud(*args, stdin=None, raw=False):
    p = subprocess.run(['gcloud', *args, '--quiet'], input=stdin,
                       capture_output=True, text=True)
    if p.returncode:
        # gcloud's diagnostics can include environment values. Never echo them.
        raise SetupError('Ошибка gcloud: ' + ' '.join(args[:3]) + '. Проверьте права аккаунта.')
    return p.stdout.strip() if raw else json.loads(p.stdout or '{}')


def request(url, method='GET', body=None, headers=None, missing=False):
    h = dict(headers or {})
    data = None
    if body is not None:
        h['Content-Type'] = 'application/json'
        data = json.dumps(body).encode()
    try:
        req = urllib.request.Request(url, data=data, headers=h, method=method)
        with urllib.request.urlopen(req, timeout=45) as response:
            return json.load(response)
    except urllib.error.HTTPError as exc:
        if missing and exc.code == 404:
            return None
        raise SetupError('HTTP ' + str(exc.code) + ': операция не выполнена; секреты скрыты.') from None
    except (urllib.error.URLError, TimeoutError, ValueError):
        raise SetupError('Сервис недоступен или вернул некорректный ответ; повторите запуск.') from None


def telegram(token, method, body=None):
    result = request('https://api.telegram.org/bot' + token + '/' + method,
                     'POST', body or {})
    if not result.get('ok'):
        raise SetupError('Telegram отклонил ' + method + '; проверьте бота и личный ID.')
    return result['result']


def env_map(revision):
    containers = revision['spec']['containers']
    if len(containers) != 1:
        raise SetupError('Ожидался один контейнер; требуется проверка конфигурации.')
    return {e['name']: e for e in containers[0].get('env', [])}


def serving_revision(service):
    traffic = service.get('status', {}).get('traffic', [])
    live = [x for x in traffic if x.get('percent', 0) > 0]
    if len(live) != 1 or live[0].get('percent') != 100 or not live[0].get('revisionName'):
        raise SetupError('Ожидалась одна опубликованная ревизия со 100% трафика.')
    return live[0]['revisionName']


def check_webhook(info, expected):
    if info.get('url') and info['url'] != expected:
        raise SetupError('У бота уже есть другой webhook. Его адрес сохранится только в приватном backup; '
                         'нужно проверить действующий обработчик, прежде чем заменять его.')


def valid_owner(value):
    if not re.fullmatch(r'[1-9][0-9]{0,15}', value or ''):
        raise SetupError('Нужен положительный личный Telegram user ID владельца, не ID группы.')
    return value


class Connector:
    def __init__(self, args):
        self.args = args
        self.project = args.project
        self.region = args.region
        self.service = args.service
        self.url = 'https://liikkuvakorjaamo.fi'
        self.webhook = self.url + '/crm/bot/telegram'
        self.fs = ('https://firestore.googleapis.com/v1/projects/' + self.project
                   + '/databases/liikkuvakorjaamo/documents')

    def google(self, url, method='GET', body=None, missing=False):
        access = cloud('auth', 'print-access-token', raw=True)
        return request(url, method, body, {'Authorization': 'Bearer ' + access}, missing)

    def describe(self, project, service):
        state = cloud('run', 'services', 'describe', service, '--project='+project,
                      '--region='+self.region, '--format=json')
        revision = serving_revision(state)
        deployed = cloud('run', 'revisions', 'describe', revision, '--project='+project,
                         '--region='+self.region, '--format=json')
        return state, deployed

    def value(self, entries, name, project):
        item = entries.get(name, {})
        ref = item.get('valueFrom', {}).get('secretKeyRef')
        if ref:
            secret = ref['name']
            if secret.startswith('projects/'):
                parts = secret.split('/')
                project, secret = parts[1], parts[3]
            return cloud('secrets', 'versions', 'access', ref.get('key', 'latest'),
                         '--secret='+secret, '--project='+project, raw=True)
        return item.get('value', '').strip()

    def users(self, field, value, transaction=None):
        body = {'structuredQuery': {'from': [{'collectionId': 'crm_users'}],
                'where': {'fieldFilter': {'field': {'fieldPath': field}, 'op': 'EQUAL',
                                         'value': {'stringValue': value}}}, 'limit': 3}}
        if transaction:
            body['transaction'] = transaction
        return [row['document'] for row in self.google(self.fs+':runQuery', 'POST', body)
                if 'document' in row]

    def check_owner(self, transaction=None):
        owners = self.users('Email', self.email, transaction)
        if len(owners) != 1:
            raise SetupError('Не найдена единственная существующая запись владельца по CRM_OWNER_EMAIL.')
        owner = owners[0]
        fields = owner.get('fields', {})
        if fields.get('Role', {}).get('stringValue') != 'owner' or not fields.get('Active', {}).get('booleanValue'):
            raise SetupError('Запись владельца неактивна или имеет другую роль.')
        previous = fields.get('TelegramID', {}).get('stringValue', '')
        if previous and previous != self.owner_id:
            raise SetupError('У владельца уже привязан другой Telegram ID. Автоматическая замена остановлена.')
        for user in self.users('TelegramID', self.owner_id, transaction):
            if user['name'] != owner['name']:
                raise SetupError('Этот Telegram ID уже принадлежит другому сотруднику CRM.')
        return owner

    def preflight(self):
        sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
        import update
        self.updater = update.Updater(update.parse_args(['--project='+self.project, '--region='+self.region, '--service='+self.service]))
        try:
            self.updater.discover()
        except update.UpdateError:
            raise SetupError('Не удалось проверить привязку рабочего домена. Облачные настройки не изменены.') from None
        self.state, self.revision = self.describe(self.project, self.service)
        self.before = serving_revision(self.state)
        if self.state['status'].get('latestReadyRevisionName') != self.before or self.state['status'].get('latestCreatedRevisionName') != self.before:
            raise SetupError('Сначала опубликуйте готовую CRM-ревизию: последняя сборка отличается от рабочей.')
        entries = env_map(self.revision)
        self.email = self.value(entries, 'CRM_OWNER_EMAIL', self.project).lower()
        if not self.email:
            raise SetupError('Не задан CRM_OWNER_EMAIL; нельзя определить владельца.')
        token = self.value(entries, 'TELEGRAM_BOT_TOKEN', self.project)
        if not token:
            raise SetupError('В рабочем Liikkuva отсутствует токен существующего бота.')
        bot = telegram(token, 'getMe')
        _, autohelp = self.describe('seventh-chassis-247718', 'autohelp-landing')
        other = self.value(env_map(autohelp), 'TELEGRAM_BOT_TOKEN', 'seventh-chassis-247718')
        if not other or telegram(other, 'getMe').get('id') != bot.get('id'):
            raise SetupError('У сайтов разные боты. Требуется выбрать общий бот; настройки не изменены.')
        current = self.value(entries, 'CRM_TELEGRAM_BOT_TOKEN', self.project)
        if current and telegram(current, 'getMe').get('id') != bot.get('id'):
            raise SetupError('CRM уже настроена на другого бота; настройки не изменены.')
        self.token = token
        self.webhook_info = telegram(token, 'getWebhookInfo')
        check_webhook(self.webhook_info, self.webhook)
        supplied = self.args.owner_telegram_id or self.value(entries, 'CRM_OWNER_TELEGRAM_ID', self.project)
        if not supplied:
            if not sys.stdin.isatty():
                raise SetupError('Укажите --owner-telegram-id с вашим личным Telegram user ID.')
            supplied = input('Ваш личный Telegram user ID (не номер телефона и не ID группы): ').strip()
        self.owner_id = valid_owner(supplied)
        chat = telegram(token, 'getChat', {'chat_id': self.owner_id})
        if chat.get('type') != 'private' or str(chat.get('id')) != self.owner_id:
            raise SetupError('ID не является личным чатом. Сначала запустите существующего бота.')
        self.owner = self.check_owner()
        self.hook_secret = self.value(entries, 'CRM_TELEGRAM_WEBHOOK_SECRET', self.project)
        self.tick_secret = self.value(entries, 'CRM_INTERNAL_TOKEN', self.project)
        if self.hook_secret and not re.fullmatch(r'[A-Za-z0-9_-]{32,256}', self.hook_secret):
            raise SetupError('Существующий секрет webhook имеет неподдерживаемый формат.')
        if self.tick_secret and len(self.tick_secret) < 32:
            raise SetupError('Существующий CRM_INTERNAL_TOKEN слишком короткий.')
        self.runtime = self.revision['spec'].get('serviceAccountName')
        if not self.runtime:
            raise SetupError('В ревизии не указан runtime service account.')
        self.scheduler = ('https://cloudscheduler.googleapis.com/v1/projects/' + self.project
                          + '/locations/' + self.args.scheduler_region + '/jobs/crm-telegram-tick')
        self.old_schedule = self.google(self.scheduler, missing=True)
        if self.old_schedule and self.old_schedule.get('httpTarget', {}).get('uri') != self.url+'/crm/internal/tick':
            raise SetupError('Имя crm-telegram-tick занято другой задачей; она не изменена.')
        print('Проверено: оба сайта используют @' + bot.get('username', '') + '; владелец найден.')
        print('Рабочая ревизия: ' + self.before + '. База CRM: liikkuvakorjaamo (задана в коде).')

    def save_secret(self, name, value):
        root = 'https://secretmanager.googleapis.com/v1/projects/'+self.project+'/secrets'
        existing = self.google(root+'/'+name, missing=True)
        if existing is None:
            self.google(root+'?secretId='+name, 'POST', {'replication': {'automatic': {}}})
        version = None
        if existing is not None:
            latest = self.google(root+'/'+name+'/versions/latest', missing=True)
            if latest and latest.get('state') == 'ENABLED':
                number = latest['name'].split('/')[-1]
                current = cloud('secrets', 'versions', 'access', number,
                                '--secret='+name, '--project='+self.project, raw=True)
                if current == value:
                    version = number
        if version is None:
            version = cloud('secrets', 'versions', 'add', name, '--project='+self.project,
                            '--data-file=-', '--format=value(name)', stdin=value, raw=True).split('/')[-1]
        cloud('secrets', 'add-iam-policy-binding', name, '--project='+self.project,
              '--member=serviceAccount:'+self.runtime, '--role=roles/secretmanager.secretAccessor',
              '--format=json')
        return name+':'+version

    def bind_owner(self):
        transaction = self.google(self.fs+':beginTransaction', 'POST', {})['transaction']
        try:
            owner = self.check_owner(transaction)
            self.google(self.fs+':commit', 'POST', {'transaction': transaction, 'writes': [{
                'update': {'name': owner['name'], 'fields': {'TelegramID': {'stringValue': self.owner_id}}},
                'updateMask': {'fieldPaths': ['TelegramID']},
                'currentDocument': {'updateTime': owner['updateTime']}}]})
        except Exception:
            try:
                self.google(self.fs+':rollback', 'POST', {'transaction': transaction})
            except SetupError:
                pass
            raise

    def backup(self):
        directory = Path.home()/'.liikkuva-releases'
        directory.mkdir(mode=0o700, exist_ok=True)
        name = 'telegram-before-'+datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%d-%H%M%S')+'.json'
        path = directory/name
        fd = os.open(path, os.O_WRONLY|os.O_CREAT|os.O_EXCL, 0o600)
        with os.fdopen(fd, 'w') as out:
            json.dump({'service': self.state, 'revision': self.revision, 'owner': self.owner,
                       'webhook': self.webhook_info, 'scheduler': self.old_schedule}, out)
        print('Приватная резервная копия: '+str(path))

    def apply(self):
        self.backup()
        self.hook_secret = self.hook_secret or secrets.token_urlsafe(48)
        self.tick_secret = self.tick_secret or secrets.token_urlsafe(48)
        refs = {name: self.save_secret(secret_name, value) for name, secret_name, value in [
            ('CRM_TELEGRAM_BOT_TOKEN', 'crm-shared-telegram-token', self.token),
            ('CRM_TELEGRAM_WEBHOOK_SECRET', 'crm-telegram-webhook-secret', self.hook_secret),
            ('CRM_INTERNAL_TOKEN', 'crm-internal-token', self.tick_secret)]}
        # Detect deployments made while secrets and permissions were prepared.
        state, _ = self.describe(self.project, self.service)
        if serving_revision(state) != self.before or state['status'].get('latestCreatedRevisionName') != self.before:
            raise SetupError('Обнаружено параллельное обновление. Повторите запуск.')
        print('Применяю настройки к рабочему сервису без пересборки образа…', flush=True)
        deployed = cloud('run', 'services', 'update', self.service, '--project='+self.project,
                         '--region='+self.region, '--update-secrets='+','.join(k+'='+v for k,v in refs.items()),
                         '--update-env-vars=CRM_OWNER_TELEGRAM_ID='+self.owner_id, '--format=json')
        revision = deployed['status']['latestReadyRevisionName']
        if revision == self.before or deployed['status'].get('latestCreatedRevisionName') != revision:
            raise SetupError('Новая конфигурация не стала готовой. Webhook не изменён.')
        cloud('run', 'services', 'update-traffic', self.service, '--project='+self.project,
              '--region='+self.region, '--to-revisions='+revision+'=100', '--format=json')
        # Hosting may route to a pinned tag. Use the project's existing updater logic.
        import update
        updater = self.updater
        if updater.hosting_run and updater.hosting_run.get('tag'):
            cloud('run', 'services', 'update-traffic', self.service, '--project='+self.project,
                  '--region='+self.region, '--update-tags='+updater.hosting_run['tag']+'='+revision, '--format=json')
        try:
            updater.verify_public_revision(revision)
        except update.UpdateError:
            raise SetupError('Конфигурация опубликована, но основной домен не подтвердил ревизию. Webhook не изменён.') from None
        # Negative update_id is ignored before any user or job is accessed.
        request(self.webhook, 'POST', {'update_id': -1},
                {'X-Telegram-Bot-Api-Secret-Token': self.hook_secret})
        self.bind_owner()
        check_webhook(telegram(self.token, 'getWebhookInfo'), self.webhook)
        telegram(self.token, 'setWebhook', {'url': self.webhook, 'secret_token': self.hook_secret,
                  'allowed_updates': ['message'], 'max_connections': 1, 'drop_pending_updates': False})
        # Secrets travel only in the authenticated HTTPS body, not CLI arguments.
        schedule = {'name': self.scheduler.split('/v1/', 1)[1], 'schedule': '* * * * *',
                    'timeZone': 'Europe/Helsinki', 'attemptDeadline': '180s',
                    'httpTarget': {'uri': self.url+'/crm/internal/tick', 'httpMethod': 'POST',
                                   'headers': {'Authorization': 'Bearer '+self.tick_secret}},
                    'retryConfig': {'minBackoffDuration': '30s', 'maxBackoffDuration': '300s', 'maxDoublings': 3}}
        if self.old_schedule:
            self.google(self.scheduler+'?updateMask=schedule,timeZone,attemptDeadline,httpTarget,retryConfig', 'PATCH', schedule)
            if self.old_schedule.get('state') == 'PAUSED':
                self.google(self.scheduler+':resume', 'POST', {})
        else:
            self.google(self.scheduler.rsplit('/', 1)[0], 'POST', schedule)
        # This processes real pending notifications; no synthetic customer/job is created.
        tick = request(self.url+'/crm/internal/tick', 'POST', {}, {'Authorization': 'Bearer '+self.tick_secret})
        info = telegram(self.token, 'getWebhookInfo')
        if info.get('url') != self.webhook:
            raise SetupError('Telegram не подтвердил адрес webhook.')
        if tick.get('failed', 0):
            raise SetupError('Подключение сохранено, но часть уведомлений не доставлена. Проверьте личные ID сотрудников и запуск бота.')
        print('Подключено: существующий бот, владелец, webhook и ежеминутная очередь уведомлений.')
        print('Рабочая ревизия: '+revision+'. Откройте бота и отправьте /start.')


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--project', default='liikkuva-korjaamo')
    p.add_argument('--region', default='europe-north1')
    p.add_argument('--service', default='liikkuva-preview')
    p.add_argument('--scheduler-region', default='europe-west1')
    p.add_argument('--owner-telegram-id')
    p.add_argument('--apply', action='store_true')
    args = p.parse_args(argv)
    connector = Connector(args)
    connector.preflight()
    if args.apply:
        connector.apply()
    else:
        print('Проверка завершена; изменений нет. Для подключения добавьте --apply.')


if __name__ == '__main__':
    try:
        main()
    except (SetupError, KeyboardInterrupt) as exc:
        print('Остановлено: '+str(exc), file=sys.stderr)
        sys.exit(1)
