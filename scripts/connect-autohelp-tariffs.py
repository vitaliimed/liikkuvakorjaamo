#!/usr/bin/env python3
"""Connect published AutoHelp tariff adapter to CRM without changing prices.

Run only after both repositories have been deployed. No changes without --apply.
"""
import argparse
import importlib.util
import json
import os
from pathlib import Path
import secrets
import sys
import time
import urllib.parse

spec = importlib.util.spec_from_file_location('crm_setup', Path(__file__).with_name('connect-crm-telegram.py'))
setup = importlib.util.module_from_spec(spec)
spec.loader.exec_module(setup)


def require_current(state):
    revision = setup.serving_revision(state)
    if any(state['status'].get(key) != revision for key in ['latestCreatedRevisionName', 'latestReadyRevisionName']):
        raise setup.SetupError('Сначала опубликуйте готовую сборку; последняя ревизия отличается от рабочей.')
    return revision


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--apply', action='store_true')
    args = p.parse_args(argv)
    c = setup.Connector(argparse.Namespace(project='liikkuva-korjaamo', region='europe-north1', service='liikkuva-preview'))
    state, revision = c.describe(c.project, c.service)
    before = require_current(state)
    ah_state, ah_revision = c.describe('seventh-chassis-247718', 'autohelp-landing')
    ah_before = require_current(ah_state)
    env = setup.env_map(revision)
    ah_env = setup.env_map(ah_revision)
    ah_url = c.value(ah_env, 'SITE_URL', 'seventh-chassis-247718').rstrip('/')
    parsed = urllib.parse.urlparse(ah_url)
    if parsed.scheme != 'https' or not parsed.hostname or parsed.username or parsed.path or parsed.query:
        raise setup.SetupError('Не удалось определить HTTPS-домен AutoHelp из его рабочей конфигурации.')
    health = setup.request(ah_url+'/api/tariff-health')
    if health.get('adapter') != 'autohelp-crm-tariffs-v1' or health.get('revision') != ah_before:
        raise setup.SetupError('Сначала опубликуйте ветку AutoHelp с адаптером тарифов. Текущий домен её не подтвердил.')
    token = c.value(env, 'CRM_AUTOHELP_TARIFF_TOKEN', c.project)
    other = c.value(ah_env, 'CRM_AUTOHELP_TARIFF_TOKEN', 'seventh-chassis-247718')
    if token and len(token) < 32:
        raise setup.SetupError('Существующий секрет тарифов слишком короткий.')
    if other and token and other != token:
        raise setup.SetupError('Секреты тарифов двух проектов различаются; автоматическая замена остановлена.')
    token = token or other or secrets.token_urlsafe(48)
    if len(token) < 32:
        raise setup.SetupError('Секрет тарифов слишком короткий.')
    c.runtime = revision['spec'].get('serviceAccountName')
    ah_runtime = ah_revision['spec'].get('serviceAccountName')
    if not c.runtime or not ah_runtime:
        raise setup.SetupError('Не определён runtime service account одного из сервисов.')
    sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
    import update
    updater = update.Updater(update.parse_args(['--project='+c.project,'--region='+c.region,'--service='+c.service]))
    updater.discover()
    if not args.apply:
        print('Оба опубликованных сервиса найдены. Для соединения добавьте --apply. Цены не меняются.')
        return
    directory = Path.home()/'.liikkuva-releases'
    directory.mkdir(mode=0o700, exist_ok=True)
    backup = directory/('tariffs-before-'+str(time.time_ns())+'.json')
    fd = os.open(backup, os.O_WRONLY|os.O_CREAT|os.O_EXCL, 0o600)
    with os.fdopen(fd,'w') as f:
        json.dump({'liikkuva':state,'autohelp':ah_state},f)
    print('Приватная резервная копия: '+str(backup))
    crm_secret = c.save_secret('crm-autohelp-tariff-token', token)
    ah = setup.Connector(argparse.Namespace(project='seventh-chassis-247718',region=c.region,service='autohelp-landing'))
    ah.runtime = ah_runtime
    ah_secret = ah.save_secret('crm-autohelp-tariff-token',token)
    for project, service, expected in [(c.project,c.service,before),(ah.project,ah.service,ah_before)]:
        current,_ = c.describe(project,service)
        if require_current(current) != expected:
            raise setup.SetupError('Параллельное развёртывание обнаружено. Повторите подключение.')
    def deploy(connector, secret, env_flag=None):
        flags = ['--update-secrets=CRM_AUTOHELP_TARIFF_TOKEN='+secret]
        if env_flag:
            flags.append('--update-env-vars='+env_flag)
        result = setup.cloud('run','services','update',connector.service,'--project='+connector.project,
                             '--region='+connector.region,*flags,'--format=json')
        new = result['status'].get('latestReadyRevisionName')
        if not new or new != result['status'].get('latestCreatedRevisionName'):
            raise setup.SetupError('Новая конфигурация не стала готовой.')
        setup.cloud('run','services','update-traffic',connector.service,'--project='+connector.project,
                    '--region='+connector.region,'--to-revisions='+new+'=100','--format=json')
        return new
    new_crm = deploy(c,crm_secret)
    if updater.hosting_run and updater.hosting_run.get('tag'):
        setup.cloud('run','services','update-traffic',c.service,'--project='+c.project,'--region='+c.region,
                    '--update-tags='+updater.hosting_run['tag']+'='+new_crm,'--format=json')
    updater.verify_public_revision(new_crm)
    endpoint = 'https://liikkuvakorjaamo.fi/crm/internal/autohelp-tariffs'
    book = setup.request(endpoint, headers={'Authorization':'Bearer '+token})
    if not book.get('items') or any(x.get('brand')!='autohelp' for x in book['items']):
        raise setup.SetupError('CRM не подтвердила отдельные тарифы AutoHelp. AutoHelp не переключён.')
    print('Подключаю существующие цены CRM (без перезаписи):')
    for item in book['items']:
        print(item['service']+': '+str(item['cents']/100)+' EUR; active='+str(item['active']))
    # Recheck AutoHelp immediately before its mutation as the CRM update can take time.
    current,_ = c.describe(ah.project,ah.service)
    if require_current(current) != ah_before:
        raise setup.SetupError('AutoHelp обновлён параллельно. Повторите запуск.')
    new_ah = deploy(ah,ah_secret,'CRM_AUTOHELP_TARIFF_URL='+endpoint)
    verified = False
    for _ in range(12):
        try:
            check = setup.request(ah_url+'/api/tariff-health?revision='+new_ah)
            if check.get('revision')==new_ah and check.get('connected') is True:
                verified = True
                break
        except setup.SetupError:
            pass
        time.sleep(5)
    if not verified:
        raise setup.SetupError('AutoHelp обновлён, но домен не подтвердил подключение. Проверьте маршрутизацию; резервная копия сохранена.')
    print('Подключено. CRM: '+new_crm+'; AutoHelp: '+new_ah)
    print('Тарифы AutoHelp: '+ah_url+'/admin/tariffs')


if __name__=='__main__':
    try:
        main()
    except (setup.SetupError, KeyboardInterrupt) as exc:
        print('Остановлено: '+str(exc),file=sys.stderr)
        sys.exit(1)
