#!/usr/bin/env python3
"""Promote an explicitly identified existing build; never rebuild either site."""
import argparse
import importlib.util
from pathlib import Path
import sys
import time

spec = importlib.util.spec_from_file_location('setup', Path(__file__).with_name('connect-crm-telegram.py'))
setup = importlib.util.module_from_spec(spec)
spec.loader.exec_module(setup)
PROJECT, REGION, SERVICE = 'seventh-chassis-247718', 'europe-north1', 'autohelp-landing'


def describe():
    return setup.cloud('run','services','describe',SERVICE,'--project='+PROJECT,'--region='+REGION,'--format=json')


def promote(revision):
    setup.cloud('run','services','update-traffic',SERVICE,'--project='+PROJECT,'--region='+REGION,
                '--to-revisions='+revision+'=100','--format=json')


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--revision', required=True)
    p.add_argument('--previous', required=True)
    p.add_argument('--image-digest', required=True)
    args = p.parse_args()
    state = describe()
    if state['status'].get('latestCreatedRevisionName') != args.revision or setup.serving_revision(state) not in (args.previous,args.revision):
        raise setup.SetupError('Состояние AutoHelp изменилось. Трафик не переключён.')
    candidate = setup.cloud('run','revisions','describe',args.revision,'--project='+PROJECT,'--region='+REGION,'--format=json')
    if candidate.get('status',{}).get('imageDigest') != args.image_digest:
        raise setup.SetupError('Образ ревизии не совпал с ожидаемой сборкой.')
    setup.ready_created_revision(state, PROJECT, REGION)
    print('Переключаю AutoHelp на '+args.revision+' без повторной сборки…',flush=True)
    promote(args.revision)
    for _ in range(12):
        try:
            health = setup.request('https://autohelp.fi/api/tariff-health?revision='+args.revision)
            if health.get('adapter') == 'autohelp-crm-tariffs-v1' and health.get('revision') == args.revision:
                current = describe()
                if setup.serving_revision(current) != args.revision:
                    raise setup.SetupError('Обнаружено параллельное переключение трафика.')
                print('AutoHelp опубликован и проверен через основной домен: '+args.revision)
                return
        except setup.SetupError:
            pass
        time.sleep(5)
    current = describe()
    if setup.serving_revision(current) == args.revision and current['status'].get('latestCreatedRevisionName') == args.revision:
        promote(args.previous)
        raise setup.SetupError('Домен не подтвердил адаптер. Трафик возвращён на '+args.previous+'. Подключения не запускались.')
    raise setup.SetupError('Домен не подтвердил адаптер; обнаружено параллельное изменение. Автоматический откат не выполнен.')


if __name__ == '__main__':
    try:
        main()
    except setup.SetupError as exc:
        print(str(exc),file=sys.stderr)
        sys.exit(1)
