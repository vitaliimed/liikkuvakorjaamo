#!/usr/bin/env python3
"""Update the existing Liikkuva service; preserve configuration and check before promotion."""
from __future__ import annotations

import argparse
import email.utils
import getpass
import json
import os
from pathlib import Path
import re
import shlex
import shutil
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request

ROOT = Path(__file__).resolve().parent
DOMAIN = 'https://liikkuvakorjaamo.fi'
VERSION = '2026.11.01-pine-settings'
HEALTH_PATH = '/api/health'
DATABASE = 'liikkuvakorjaamo'
SITE = 'liikkuva-liikkuva-korjaamo'
SERVICES = {'liikkuva-preview', 'liikkuva-landing', 'liikkuvakorjaamo'}
RULES = """rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    // All client access is mediated by the authenticated Go application.
    // Server SDK operations authenticate with IAM and do not use these rules.
    match /{document=**} { allow read, write: if false; }
  }
}
"""


class UpdateError(RuntimeError):
    pass


class EndpointError(UpdateError):
    """A serving URL error, distinct from the Google management API."""
    def __init__(self, code, url):
        self.code = code
        self.url = url
        detail = 'сетевая ошибка' if code == 0 else f'HTTP {code}'
        super().__init__(f'{detail}, {url}')


class NoEndpointRedirect(urllib.request.HTTPRedirectHandler):
    # Never forward an operator identity token to a redirect destination.
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


class APIError(UpdateError):
    def __init__(self, code, url):
        self.code = code
        super().__init__(f'Google API: HTTP {code}, {urllib.parse.urlsplit(url).path}.')


def run(*args, input=None, cwd=None, env=None, check=True):
    result = subprocess.run(args, input=input, cwd=cwd, env=env, text=True,
                            stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    if check and result.returncode:
        # Arguments never contain raw keys/passwords. Do not echo response
        # bodies, environment snapshots or API credentials into terminal logs.
        command = ' '.join(str(x) for x in args[:4])
        reason = (result.stderr or '').strip()
        if Path(str(args[0])).name == 'go':
            reason = (result.stdout + '\n' + result.stderr).strip()
        raise UpdateError(f'Не выполнена команда: {command}\n{reason[-2500:]}')
    return result


def env_entries(service):
    return {entry['name']: entry for entry in
            service.get('spec', {}).get('template', {}).get('spec', {}).get('containers', [{}])[0].get('env', [])}


def select_service(services, hosting_run=None, requested=None, region='europe-north1'):
    names = {s['metadata']['name'] for s in services}
    if hosting_run and hosting_run.get('region', region) != region:
        raise UpdateError('Firebase использует другой регион. Укажите его через --region.')
    bound = hosting_run.get('serviceId') if hosting_run else None
    if requested:
        if requested not in SERVICES and requested != bound:
            raise UpdateError('Указанный сервис не распознан как Liikkuva. Проверьте привязку Hosting.')
        if requested not in names:
            raise UpdateError(f'Сервис {requested} не найден в выбранном проекте/регионе.')
        if bound and requested != bound:
            raise UpdateError(f'Домен привязан к {bound}, а указан {requested}. Обновление остановлено.')
        return requested
    if bound:
        if bound not in names:
            raise UpdateError(f'Сервис из Firebase ({bound}) не найден. Проверьте проект и регион.')
        return bound
    candidates = sorted(names & SERVICES)
    if len(candidates) == 1:
        return candidates[0]
    raise UpdateError('Не удалось однозначно выбрать сервис. Повторите с --service ИМЯ. Найдены: ' + ', '.join(candidates))


def validate_password(password):
    if len(password) < 12 or len(password.encode('utf-8')) > 72 or '\n' in password or '\r' in password:
        raise UpdateError('Пароль: от 12 символов, до 72 байт, одна строка.')


def prompt_password():
    password = getpass.getpass('Новый пароль кабинета (от 12 символов, ввод скрыт): ')
    validate_password(password)
    if password != getpass.getpass('Повторите новый пароль: '):
        raise UpdateError('Пароли не совпадают.')
    return password


def update_command(*flags):
    # A subshell returns to the caller's previous directory. Always print a
    # command for this checkout, not a same-named script in an older folder.
    return shlex.join(['python3', str(ROOT / 'update.py'), *flags])


class Updater:
    def __init__(self, args):
        self.args = args
        self.project = args.project
        self.region = args.region
        self.service = None
        self.token = None
        self.project_number = None
        self.updates = {}
        self.secret_updates = {}
        self.hosting_run = None
        self.snapshot = None
        self.entries = {}
        self.identity_token = None
        self.identity_attempted = False

    def gc(self, *args, **kwargs):
        return run('gcloud', *args, '--project=' + self.project, '--quiet', **kwargs)

    def api(self, url, method='GET', data=None, authenticated=True):
        headers = {'Accept': 'application/json'}
        if authenticated:
            if not self.token:
                self.token = self.gc('auth', 'print-access-token').stdout.strip()
            headers['Authorization'] = 'Bearer ' + self.token
        payload = None
        if data is not None:
            headers['Content-Type'] = 'application/json'
            payload = json.dumps(data).encode('utf-8')
        request = urllib.request.Request(url, data=payload, headers=headers, method=method)
        try:
            with urllib.request.urlopen(request, timeout=45) as response:
                content = response.read()
                return json.loads(content) if content else {}
        except (json.JSONDecodeError, UnicodeDecodeError):
            raise UpdateError('Сервер вернул неожиданный ответ вместо JSON: ' + urllib.parse.urlsplit(url).hostname) from None
        except urllib.error.HTTPError as exc:
            code = exc.code
            exc.close()
            raise APIError(code, url) from None
        except urllib.error.URLError as exc:
            raise UpdateError(f'Нет соединения с {urllib.parse.urlsplit(url).hostname}: {exc.reason}') from None

    def describe(self):
        return json.loads(self.gc('run', 'services', 'describe', self.service,
                                 '--region=' + self.region, '--format=json').stdout)

    def discover(self):
        services = json.loads(self.gc('run', 'services', 'list', '--region=' + self.region, '--format=json').stdout)
        base = f'https://firebasehosting.googleapis.com/v1beta1/sites/{self.args.site}'
        try:
            live = self.api(base + '/channels/live')
            version = live.get('release', {}).get('version', {})
            if isinstance(version, str):
                version = {'name': version}
            if version.get('name') and 'config' not in version:
                version = self.api('https://firebasehosting.googleapis.com/v1beta1/' + version['name'])
            routes = [r['run'] for r in version.get('config', {}).get('rewrites', []) if 'run' in r]
            service_ids = {r.get('serviceId') for r in routes}
            if len(service_ids) > 1:
                raise UpdateError('Hosting использует несколько Cloud Run сервисов. Укажите --service явно.')
            if routes:
                self.hosting_run = routes[0]
        except APIError as exc:
            if exc.code not in (403, 404):
                raise
            print('Не удалось прочитать привязку Hosting; проверяю список сервисов Cloud Run.', flush=True)
        self.service = select_service(services, self.hosting_run, self.args.service, self.region)
        self.snapshot = self.describe()
        self.entries = env_entries(self.snapshot)
        site_url = self.entries.get('SITE_URL', {}).get('value', '').rstrip('/')
        service_urls = {self.snapshot.get('status', {}).get('url', '').rstrip('/')}
        service_urls.update(r.get('url', '').rstrip('/') for r in self.snapshot.get('status', {}).get('traffic', []))
        if site_url and site_url != DOMAIN and site_url not in service_urls:
            raise UpdateError(f'SITE_URL выбранного сервиса отличается от {DOMAIN} и его технического адреса. Проверьте --service.')
        db_project = self.entries.get('FIRESTORE_PROJECT_ID', {}).get('value', self.project)
        if db_project and db_project != self.project:
            raise UpdateError('База выбранного сервиса находится в другом проекте. Требуется проверить конфигурацию.')
        self.project_number = self.gc('projects', 'describe', self.project, '--format=value(projectNumber)').stdout.strip()
        self.runtime_account = self.snapshot['spec']['template']['spec'].get('serviceAccountName') or self.project_number + '-compute@developer.gserviceaccount.com'
        self.updates = {'SITE_URL': DOMAIN}
        if not self.entries.get('ADMIN_RECOVERY_EMAIL', {}).get('value'):
            self.updates['ADMIN_RECOVERY_EMAIL'] = 'medynikv@gmail.com'
        if not self.entries.get('FIRESTORE_PROJECT_ID', {}).get('value'):
            self.updates['FIRESTORE_PROJECT_ID'] = self.project
        print(f'Проект: {self.project}; регион: {self.region}; существующий сервис: {self.service}.', flush=True)

    def prepare_storage(self):
        self.gc('services', 'enable', 'firestore.googleapis.com', 'firebaserules.googleapis.com')
        databases = json.loads(self.gc('firestore', 'databases', 'list', '--format=json').stdout)
        if isinstance(databases, dict):
            databases = databases.get('databases', [])
        if not any(d.get('name', '').endswith('/' + DATABASE) for d in databases):
            print('Создаю хранилище кабинета в europe-north1…', flush=True)
            self.gc('firestore', 'databases', 'create', '--database=' + DATABASE,
                    '--location=europe-north1', '--type=firestore-native')
        # IAM is used by the server SDK. The web client has no direct DB access.
        self.gc('projects', 'add-iam-policy-binding', self.project,
                '--member=serviceAccount:' + self.runtime_account,
                '--role=roles/datastore.user', '--condition=None', '--format=none')
        self.secure_database()

    def configure_media(self):
        """Use a private, isolated bucket; website assets are served through the app."""
        bucket = self.entries.get('GCS_BUCKET_NAME', {}).get('value') or self.project + '-liikkuva-files'
        if not bucket.endswith('-liikkuva-files') or not re.fullmatch(r'[a-z0-9][a-z0-9.-]{2,221}', bucket):
            raise UpdateError('Хранилище не распознано как отдельное хранилище Liikkuva. Настройки не изменены.')
        self.gc('services', 'enable', 'storage.googleapis.com')
        root = 'https://storage.googleapis.com/storage/v1/b'
        try:
            metadata = self.api(root + '/' + bucket)
        except APIError as exc:
            if exc.code != 404:
                raise
            print('Создаю закрытое хранилище изображений Liikkuva…', flush=True)
            metadata = self.api(root + '?project=' + urllib.parse.quote(self.project), method='POST', data={
                'name': bucket, 'location': self.region, 'storageClass': 'STANDARD',
                'iamConfiguration': {'uniformBucketLevelAccess': {'enabled': True},
                                     'publicAccessPrevention': 'enforced'}})
        if str(metadata.get('projectNumber')) != str(self.project_number):
            raise UpdateError('Хранилище принадлежит другому проекту. Доступ не изменён.')
        self.gc('storage', 'buckets', 'add-iam-policy-binding', 'gs://' + bucket,
                '--member=serviceAccount:' + self.runtime_account,
                '--role=roles/storage.objectUser', '--format=none')
        self.updates['GCS_BUCKET_NAME'] = bucket
        print('Изображения сайта: отдельная папка site-assets; фото заявок остаются закрытыми.', flush=True)

    def secure_database(self):
        root = f'https://firebaserules.googleapis.com/v1/projects/{self.project}'
        release_name = f'projects/{self.project}/releases/cloud.firestore/{DATABASE}'
        current = None
        try:
            current = self.api('https://firebaserules.googleapis.com/v1/' + release_name)
            existing = self.api('https://firebaserules.googleapis.com/v1/' + current['rulesetName'])
            files = existing.get('source', {}).get('files', [])
            if len(files) == 1 and files[0].get('content') == RULES:
                return self.check_database_private()
        except APIError as exc:
            if exc.code != 404:
                raise
        print('Закрываю прямой клиентский доступ к базе Liikkuva; приложение работает через IAM…', flush=True)
        ruleset = self.api(root + '/rulesets', 'POST', {
            'source': {'files': [{'name': 'firestore.rules', 'content': RULES}]},
            'attachment_point': f'firestore.googleapis.com/projects/{self.project_number}/databases/{DATABASE}'})
        release = {'name': release_name, 'rulesetName': ruleset['name']}
        if current:
            self.api('https://firebaserules.googleapis.com/v1/' + release_name,
                     'PATCH', {'release': release, 'updateMask': 'rulesetName'})
        else:
            self.api(root + '/releases', 'POST', release)
        for attempt in range(7):
            try:
                return self.check_database_private()
            except UpdateError:
                if attempt == 6:
                    raise
                print('Ожидаю применения правил доступа…', flush=True)
                time.sleep(10)

    def check_database_private(self):
        collection = self.entries.get('SITE_SETTINGS_COLLECTION', {}).get('value', 'site_settings') + '_private'
        path = f'https://firestore.googleapis.com/v1/projects/{self.project}/databases/{DATABASE}/documents/{collection}/admin_access'
        try:
            self.api(path, authenticated=False)
        except APIError as exc:
            if exc.code in (401, 403):
                return
            raise UpdateError('Правила доступа ещё применяются. Повторите обновление через несколько минут.') from None
        raise UpdateError('Прямой доступ к данным кабинета пока открыт. Обновление остановлено до защиты базы.')

    def configure_email(self):
        print('Укажите SMTP-данные подтверждённого отправителя. Пересылка входящих писем не заменяет SMTP.', flush=True)
        host = input('SMTP host [smtp-relay.brevo.com]: ').strip() or 'smtp-relay.brevo.com'
        port = input('SMTP port [587]: ').strip() or '587'
        login = input('SMTP login: ').strip()
        password = getpass.getpass('SMTP key / password (ввод скрыт): ')
        address = input('Подтверждённый адрес отправителя [service@liikkuvakorjaamo.fi]: ').strip() or 'service@liikkuvakorjaamo.fi'
        if not re.fullmatch(r'[A-Za-z0-9.-]+', host) or port not in ('465', '587') or not login or not password or '\n' in login or not re.fullmatch(r'[^\s@,<>]+@[^\s@,<>]+\.[^\s@,<>]+', address):
            raise UpdateError('Проверьте SMTP host, порт, логин, ключ и адрес отправителя.')
        self.gc('services', 'enable', 'secretmanager.googleapis.com')
        for key, value in [('SMTP_USERNAME', login), ('SMTP_PASSWORD', password)]:
            self.save_secret(key, value)
        self.updates.update(SMTP_HOST=host, SMTP_PORT=port,
                            MAIL_FROM=email.utils.formataddr(('Liikkuva Auto Korjaamo', address)), MAIL_REPLY_TO=address)

    def save_secret(self, key, value):
        name = 'liikkuva-' + key.lower().replace('_', '-')
        found = self.gc('secrets', 'describe', name, '--format=none', check=False).returncode == 0
        if not found:
            self.gc('secrets', 'create', name, '--replication-policy=automatic')
        version = self.gc('secrets', 'versions', 'add', name, '--data-file=-',
                          '--format=value(name)', input=value).stdout.strip().split('/')[-1]
        if not version.isdigit():
            raise UpdateError('Secret Manager не вернул номер версии секрета. Настройка остановлена.')
        self.gc('secrets', 'add-iam-policy-binding', name,
                '--member=serviceAccount:' + self.runtime_account,
                '--role=roles/secretmanager.secretAccessor', '--format=none')
        self.secret_updates[key] = name + ':' + version

    @staticmethod
    def telegram_metadata(token, method, data=None):
        # These read-only methods cannot change the shared bot's webhook,
        # consume AutoHelp updates, or send a message to the shared chat.
        if method not in ('getMe', 'getChat'):
            raise UpdateError('Разрешена только проверка данных Telegram.')
        request = urllib.request.Request('https://api.telegram.org/bot' + token + '/' + method,
                                         data=json.dumps(data or {}).encode(),
                                         headers={'Content-Type': 'application/json'}, method='POST')
        try:
            with urllib.request.urlopen(request, timeout=20) as response:
                result = json.loads(response.read(128 * 1024))
        except urllib.error.HTTPError as exc:
            exc.close()
            raise UpdateError('Telegram отклонил проверку. Проверьте токен бота, Chat ID и доступ бота к чату.') from None
        except (urllib.error.URLError, TimeoutError, ConnectionError, ValueError):
            raise UpdateError('Не удалось проверить Telegram. Проверьте соединение и повторите настройку.') from None
        if not isinstance(result, dict) or result.get('ok') is not True or not isinstance(result.get('result'), dict):
            raise UpdateError('Telegram не подтвердил бота или чат. Настройки не сохранены.')
        return result['result']

    def configure_telegram(self):
        print('Для общего с AutoHelp бота укажите существующий токен и тот же Chat ID.', flush=True)
        token = getpass.getpass('Telegram bot token (ввод скрыт): ').strip()
        chat = input('Telegram Chat ID: ').strip()
        if not re.fullmatch(r'\d{5,}:[A-Za-z0-9_-]{20,}', token) or not re.fullmatch(r'-?\d+|@[A-Za-z0-9_]{5,}', chat):
            raise UpdateError('Проверьте формат Telegram token и Chat ID.')
        bot = self.telegram_metadata(token, 'getMe')
        if bot.get('is_bot') is not True:
            raise UpdateError('Telegram не подтвердил учётную запись бота.')
        destination = self.telegram_metadata(token, 'getChat', {'chat_id': chat})
        if not destination.get('id'):
            raise UpdateError('Telegram не подтвердил целевой чат.')
        print('Проверен бот @' + str(bot.get('username', 'без username')) + '; чат доступен. '
              'Доставка сообщения проверяется отдельной тестовой заявкой.', flush=True)
        self.gc('services', 'enable', 'secretmanager.googleapis.com')
        self.save_secret('TELEGRAM_BOT_TOKEN', token)
        self.save_secret('TELEGRAM_CHAT_ID', str(destination['id']))

    def configured(self, key, runtime_only=False):
        if not runtime_only and key in self.secret_updates:
            return bool(self.secret_updates[key])
        if not runtime_only and key in self.updates:
            return bool(self.updates[key])
        entry = self.entries.get(key, {})
        return bool(entry.get('value') or entry.get('valueFrom', {}).get('secretKeyRef'))

    def integration_report(self):
        groups = [
            ('Отправка почты', ('SMTP_HOST', 'SMTP_USERNAME', 'SMTP_PASSWORD', 'MAIL_FROM')),
            ('Telegram', ('TELEGRAM_BOT_TOKEN', 'TELEGRAM_CHAT_ID')),
            ('Google Таблица', ('GOOGLE_APPS_SCRIPT_URL', 'GOOGLE_APPS_SCRIPT_TOKEN')),
            ('Обработка заявок', ('FIRESTORE_PROJECT_ID', 'INTERNAL_TASK_TOKEN', 'REVIEW_TOKEN_SECRET',
                                  'CLOUD_TASKS_PROJECT_ID', 'CLOUD_TASKS_LOCATION', 'CLOUD_TASKS_QUEUE',
                                  'GCS_BUCKET_NAME', 'INTERNAL_PROCESS_LEAD_URL')),
        ]
        print('Наличие настроек (без проверки доставки и без вывода секретов):', flush=True)
        for title, keys in groups:
            missing = [key for key in keys if not self.configured(key, runtime_only=True)]
            print(title + ': ' + ('задано' if not missing else 'не заданы ' + ', '.join(missing)), flush=True)
        paused = self.entries.get('LIIKKUVA_BOOTSTRAP', {}).get('value') == '1'
        print('Приём заявок: ' + ('режим настройки, отправка приостановлена' if paused else 'режим настройки не включён'), flush=True)
        print('Проверка входящей пересылки выполняется в Cloudflare Email Routing. '
              'Проверка доставки — своей заявкой после подключения очереди, хранилища и таблицы.', flush=True)

    def check_intake(self):
        required = ['INTERNAL_TASK_TOKEN', 'REVIEW_TOKEN_SECRET', 'TELEGRAM_BOT_TOKEN', 'TELEGRAM_CHAT_ID', 'GOOGLE_APPS_SCRIPT_URL', 'GOOGLE_APPS_SCRIPT_TOKEN', 'CLOUD_TASKS_PROJECT_ID', 'CLOUD_TASKS_LOCATION', 'CLOUD_TASKS_QUEUE', 'GCS_BUCKET_NAME', 'INTERNAL_PROCESS_LEAD_URL']
        missing = [k for k in required if not self.configured(k)]
        if missing:
            raise UpdateError('Приём заявок пока нельзя включить: отсутствуют ' + ', '.join(missing))
        self.updates['LIIKKUVA_BOOTSTRAP'] = '0'

    def build_cli(self, folder):
        go = shutil.which('go')
        if not go:
            raise UpdateError('Go не найден. Выполните команду в Google Cloud Shell, где доступен Go.')
        print('Проверяю Go-код и собираю локальный инструмент доступа…', flush=True)
        run(go, 'test', './...', cwd=ROOT / 'source')
        binary = Path(folder) / 'liikkuva-server'
        run(go, 'build', '-buildvcs=false', '-o', str(binary), '.', cwd=ROOT / 'source')
        return binary

    def set_password(self, binary, password):
        self.token = self.gc('auth', 'print-access-token').stdout.strip()
        env = dict(os.environ)
        # Only the CLI sees this short-lived operator credential. The server
        # never reads it, and it is not included in Cloud Run configuration.
        for key, default in [('SITE_SETTINGS_COLLECTION', 'site_settings'), ('ADMIN_RECOVERY_EMAIL', 'medynikv@gmail.com')]:
            env[key] = self.updates.get(key) or self.entries.get(key, {}).get('value') or default
        env.update(FIRESTORE_PROJECT_ID=self.project, LIIKKUVA_OPERATOR_TOKEN=self.token,
                   SITE_URL=DOMAIN, ADMIN_PASSWORD='')
        env.setdefault('ADMIN_RECOVERY_EMAIL', 'medynikv@gmail.com')
        run(str(binary), 'admin-password', input=password + '\n', env=env, cwd=ROOT / 'source')
        print('Новый пароль сохранён. Старые сеансы завершены.', flush=True)

    def deploy(self):
        stamp = time.strftime('%Y%m%d-%H%M%S', time.gmtime())
        tag = 'check-' + stamp
        args = ['run', 'deploy', self.service, '--region=' + self.region,
                '--source=' + str(ROOT / 'source'), '--no-traffic', '--tag=' + tag,
                '--format=json']
        # Update only the named keys. Preserve unrelated variables and secret refs.
        delimiter = '~'
        if any(delimiter in str(v) for v in self.updates.values()):
            raise UpdateError('Недопустимый символ в публичных настройках.')
        args.append('--update-env-vars=^' + delimiter + '^' + delimiter.join(k + '=' + v for k, v in self.updates.items()))
        plain_to_secret = [k for k in self.secret_updates if 'value' in self.entries.get(k, {})]
        secret_to_plain = [k for k in self.updates if 'valueFrom' in self.entries.get(k, {})]
        if plain_to_secret:
            args.append('--remove-env-vars=' + ','.join(plain_to_secret))
        if secret_to_plain:
            args.append('--remove-secrets=' + ','.join(secret_to_plain))
        if self.secret_updates:
            args.append('--update-secrets=' + ','.join(k + '=' + v for k, v in self.secret_updates.items()))
        print('Собираю новую ревизию без пользовательского трафика. Сборка может занять несколько минут…', flush=True)
        result = self.gc(*args)
        try:
            deployed = json.loads(result.stdout)
        except (ValueError, TypeError):
            deployed = {}
        expected = deployed.get('status', {}).get('latestCreatedRevisionName')
        if not expected:
            expected = self.describe().get('status', {}).get('latestCreatedRevisionName')
        if not expected:
            raise UpdateError('Cloud Run не вернул имя созданной ревизии. Трафик не переключён.')
        self.check_and_promote(tag, expected)

    def endpoint(self, url, identity_token=None, timeout=12):
        headers = {'Cache-Control': 'no-cache', 'Pragma': 'no-cache'}
        if identity_token:
            parsed = urllib.parse.urlsplit(url)
            if parsed.scheme != 'https' or not parsed.hostname or not parsed.hostname.endswith('.run.app'):
                raise UpdateError('Токен проверки разрешён только для HTTPS-адреса Cloud Run.')
            headers['X-Serverless-Authorization'] = 'Bearer ' + identity_token
        request = urllib.request.Request(url, headers=headers)
        opener = urllib.request.build_opener(NoEndpointRedirect())
        try:
            with opener.open(request, timeout=timeout) as response:
                return response.read(512 * 1024).decode('utf-8')
        except urllib.error.HTTPError as exc:
            code = exc.code
            exc.close()
            raise EndpointError(code, url) from None
        except (urllib.error.URLError, TimeoutError, socket.timeout, ConnectionError):
            raise EndpointError(0, url) from None
        except UnicodeDecodeError:
            raise UpdateError('Проверочный адрес вернул ответ в неожиданной кодировке.') from None

    def operator_identity(self):
        if not self.identity_attempted:
            self.identity_attempted = True
            result = self.gc('auth', 'print-identity-token', check=False)
            if result.returncode == 0:
                self.identity_token = result.stdout.strip() or None
        return self.identity_token

    def candidate_urls(self, route, deployed):
        candidates = [route.get('url')]
        status = deployed.get('status', {})
        for base in [status.get('url'), status.get('address', {}).get('url')]:
            if base:
                parsed = urllib.parse.urlsplit(base)
                candidates.append('https://' + route['tag'] + '---' + parsed.netloc)
        if self.project_number and str(self.project_number).isdigit():
            label = route['tag'] + '---' + self.service + '-' + str(self.project_number)
            if len(label) <= 63:
                candidates.append(f'https://{label}.{self.region}.run.app')
        result = []
        for url in candidates:
            if not url:
                continue
            parsed = urllib.parse.urlsplit(url)
            if (parsed.scheme != 'https' or not parsed.hostname or not parsed.hostname.endswith('.run.app')
                    or parsed.username or parsed.password or parsed.port or parsed.query or parsed.fragment
                    or parsed.path not in ('', '/')):
                continue
            value = url.rstrip('/')
            if value not in result:
                result.append(value)
        return result

    def route_ready(self, deployed, tag, revision):
        state = deployed.get('status', {})
        created = state.get('latestCreatedRevisionName')
        if created and created != revision:
            raise UpdateError('Во время проверки появилась другая ревизия. Трафик не переключён; проверьте параллельное обновление.')
        conditions = {c['type']: c.get('status') for c in state.get('conditions', [])}
        if conditions.get('Ready') != 'True' or conditions.get('RoutesReady') != 'True':
            return None
        if state.get('latestReadyRevisionName') != revision:
            return None
        generation = deployed.get('metadata', {}).get('generation')
        observed = state.get('observedGeneration')
        if generation is not None and observed is not None and int(observed) < int(generation):
            return None
        return next((r for r in state.get('traffic', []) if r.get('tag') == tag and r.get('revisionName') == revision), None)

    @staticmethod
    def health_matches(body, revision):
        try:
            health = json.loads(body)
        except (ValueError, TypeError):
            return False
        if not isinstance(health, dict):
            return False
        if health.get('revision') == revision and health.get('version') != VERSION:
            raise UpdateError('Ревизия отвечает, но версия приложения отличается от ожидаемой. Трафик не переключён.')
        return (health.get('ok') is True and health.get('site') == 'liikkuvakorjaamo.fi'
                and health.get('version') == VERSION and health.get('revision') == revision)

    def probe_revision(self, tag, revision):
        deadline = time.monotonic() + self.args.check_timeout
        attempt = 0
        last = 'маршрут ещё не готов'
        deployed = self.snapshot or {}
        while time.monotonic() < deadline:
            deployed = self.describe()
            route = self.route_ready(deployed, tag, revision)
            if route:
                metadata = deployed.get('metadata', {}).get('annotations', {})
                if str(metadata.get('run.googleapis.com/default-url-disabled', '')).lower() == 'true':
                    last = 'в настройках сервиса выключены технические адреса run.app'
                    break
                urls = self.candidate_urls(route, deployed)
                if not urls:
                    last = 'Cloud Run не вернул допустимый адрес проверочного тега'
                    break
                for url in urls:
                    remaining = deadline - time.monotonic()
                    if remaining <= 0:
                        break
                    token = None
                    try:
                        try:
                            body = self.endpoint(url + HEALTH_PATH, timeout=min(12, remaining))
                        except EndpointError as exc:
                            if exc.code not in (401, 403, 404):
                                raise
                            token = self.operator_identity()
                            if not token:
                                raise
                            body = self.endpoint(url + HEALTH_PATH, identity_token=token,
                                                 timeout=min(12, max(0.1, deadline - time.monotonic())))
                        if self.health_matches(body, revision):
                            for path, marker in [('/admin', 'Войти в кабинет'),
                                                 ('/static/css/responsive.css?v=20261031-compact2', '.hero-form-panel')]:
                                page = self.endpoint(url + path, identity_token=token)
                                if marker not in page:
                                    raise UpdateError(f'Ревизия отвечает, но не прошла проверка {path}. Трафик не переключён.')
                            return url
                        last = 'адрес пока не подтверждает ожидаемые версию и ревизию: ' + url
                    except EndpointError as exc:
                        last = str(exc)
                        if exc.code not in (0, 401, 403, 404, 408, 429, 500, 502, 503, 504):
                            raise UpdateError('Не прошла проверка ревизии: ' + last) from None
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                break
            print('Ожидаю готовности проверочного маршрута: ' + last, flush=True)
            time.sleep(min(10, 2 ** min(attempt + 1, 4), remaining))
            attempt += 1
        self.print_probe_diagnostics(deployed, tag, revision)
        raise UpdateError('Ревизия создана, но её проверочный адрес недоступен. Трафик не переключён. '
                          + last + '\nДля повторной проверки той же версии без новой сборки: '
                          + update_command('--resume', '--set-password')
                          + '\nЕсли ревизия создана старым кодом, соберите актуальную версию: '
                          + update_command('--set-password'))

    def print_probe_diagnostics(self, deployed, tag, revision):
        # Whitelist only routing fields; never print the environment or tokens.
        metadata = deployed.get('metadata', {})
        annotations = metadata.get('annotations', {})
        state = deployed.get('status', {})
        safe = {'project': self.project, 'service': self.service, 'revision': revision, 'tag': tag,
                'ingress': annotations.get('run.googleapis.com/ingress', 'not reported'),
                'default_url_disabled': annotations.get('run.googleapis.com/default-url-disabled', 'not reported'),
                'conditions': [{'type': c.get('type'), 'status': c.get('status'), 'reason': c.get('reason')}
                               for c in state.get('conditions', [])],
                'routes': [{'tag': r.get('tag'), 'revision': r.get('revisionName'), 'url': r.get('url')}
                           for r in state.get('traffic', []) if r.get('tag') == tag]}
        print('Диагностика маршрута (без паролей и ключей):\n' + json.dumps(safe, ensure_ascii=False, indent=2), flush=True)

    def verify_public_revision(self, revision):
        deadline = time.monotonic() + min(90, self.args.check_timeout)
        last = 'домен пока возвращает другую ревизию'
        while time.monotonic() < deadline:
            try:
                body = self.endpoint(DOMAIN + HEALTH_PATH + '?revision=' + urllib.parse.quote(revision),
                                     timeout=min(12, max(0.1, deadline - time.monotonic())))
                if self.health_matches(body, revision):
                    return
            except EndpointError as exc:
                last = str(exc)
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                break
            print('Проверяю обновление через основной домен…', flush=True)
            time.sleep(min(5, remaining))
        raise UpdateError('Cloud Run уже переключён на ' + revision + ', но проверка домена не завершена: '
                          + last + '. Новый пароль ещё не установлен.')

    def check_and_promote(self, tag, revision):
        print('Проверяю готовую ревизию ' + revision + '…', flush=True)
        self.probe_revision(tag, revision)
        print('Проверки ревизии пройдены. Переключаю 100% трафика…', flush=True)
        self.gc('run', 'services', 'update-traffic', self.service, '--region=' + self.region, '--to-revisions=' + revision + '=100')
        if self.hosting_run and self.hosting_run.get('tag'):
            self.gc('run', 'services', 'update-traffic', self.service, '--region=' + self.region,
                    '--update-tags=' + self.hosting_run['tag'] + '=' + revision)
        final = self.describe()
        percent = sum(r.get('percent', 0) for r in final['status']['traffic'] if r.get('revisionName') == revision)
        if percent != 100:
            raise UpdateError('Cloud Run не подтвердил 100% трафика на новой ревизии.')
        self.verify_public_revision(revision)
        self.gc('run', 'services', 'update-traffic', self.service, '--region=' + self.region, '--remove-tags=' + tag)
        print('Обновлена ревизия: ' + revision, flush=True)

    def resume(self):
        deployed = self.describe()
        state = deployed.get('status', {})
        revision = state.get('latestCreatedRevisionName')
        routes = [r for r in state.get('traffic', []) if r.get('revisionName') == revision
                  and re.fullmatch(r'check-\d{8}-\d{6}', r.get('tag', ''))]
        if not revision or not routes:
            raise UpdateError('У последней созданной ревизии нет проверочного тега этого обновления. '
                              'Для обычного обновления: ' + update_command('--set-password'))
        route = max(routes, key=lambda r: r['tag'])
        print('Использую уже созданную ревизию. Повторная сборка в Cloud Build не требуется.', flush=True)
        self.check_and_promote(route['tag'], revision)


def parse_args(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--project', default='liikkuva-korjaamo')
    p.add_argument('--region', default='europe-north1')
    p.add_argument('--site', default=SITE)
    p.add_argument('--service', help='Normally detected from live Firebase Hosting configuration')
    p.add_argument('--resume', action='store_true', help='Check and promote the latest existing check-tagged revision without rebuilding')
    p.add_argument('--check-timeout', type=int, default=150, help='Maximum seconds to wait for revision routing (30–600)')
    p.add_argument('--set-password', action='store_true', help='Set an owner password after deployment')
    p.add_argument('--password-only', action='store_true', help='Set a password without deploying')
    p.add_argument('--configure-email', action='store_true', help='Configure SMTP using hidden credential prompts')
    p.add_argument('--configure-media', action='store_true', help='Connect private durable storage for website image uploads')
    p.add_argument('--configure-telegram', action='store_true', help='Configure an existing Telegram bot using hidden credential prompts')
    p.add_argument('--check-integrations', action='store_true', help='Read configuration status without changing cloud resources')
    p.add_argument('--enable-leads', action='store_true', help='Leave setup mode after integrations are configured')
    return p.parse_args(argv)


def main(argv=None):
    args = parse_args(argv)
    if not 30 <= args.check_timeout <= 600:
        raise UpdateError('--check-timeout должен быть от 30 до 600 секунд.')
    if args.check_integrations and (args.resume or args.password_only or args.set_password or args.configure_email or args.configure_telegram or args.configure_media or args.enable_leads):
        raise UpdateError('--check-integrations используется отдельно от обновления и изменения конфигурации.')
    if args.resume and (args.password_only or args.configure_email or args.configure_telegram or args.configure_media or args.enable_leads):
        raise UpdateError('--resume используется отдельно от смены конфигурации и --password-only.')
    if args.password_only and (args.configure_email or args.configure_telegram or args.configure_media or args.enable_leads):
        raise UpdateError('--password-only используется отдельно от настройки почты и приёма заявок.')
    if not shutil.which('gcloud'):
        raise UpdateError('Откройте Google Cloud Shell: https://shell.cloud.google.com/')
    if not (ROOT / 'source' / 'admin_access.go').is_file():
        raise UpdateError('Не найдены исходники этого обновления. Выполните git pull в папке репозитория.')
    needs_password = args.set_password or args.password_only
    updater = Updater(args)
    updater.discover()
    if args.check_integrations:
        updater.integration_report()
        return
    updater.prepare_storage()
    if args.configure_media:
        updater.configure_media()
    if args.configure_email:
        updater.configure_email()
    if args.configure_telegram:
        updater.configure_telegram()
    if args.enable_leads:
        updater.check_intake()
    with tempfile.TemporaryDirectory(prefix='liikkuva-build-') as temp:
        binary = updater.build_cli(temp) if needs_password or not args.resume else None
        if args.resume:
            updater.resume()
        elif not args.password_only:
            updater.deploy()
        if needs_password:
            # Ask only after successful verification, so a failed health check
            # does not make the user enter a password which was never applied.
            updater.set_password(binary, prompt_password())
    print('Кабинет: ' + DOMAIN + '/admin')
    print('Почта восстановления по умолчанию: medynikv@gmail.com. Сохранённый в кабинете адрес имеет приоритет.')
    if not (args.configure_email or all(k in updater.entries for k in ('SMTP_HOST', 'SMTP_USERNAME', 'SMTP_PASSWORD', 'MAIL_FROM'))):
        print('Для восстановления по почте подключите отправку: ' + update_command('--configure-email'))


if __name__ == '__main__':
    try:
        main()
    except (UpdateError, KeyboardInterrupt) as exc:
        print('\nОбновление остановлено: ' + str(exc), file=sys.stderr)
        sys.exit(1)
