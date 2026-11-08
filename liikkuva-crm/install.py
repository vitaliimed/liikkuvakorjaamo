#!/usr/bin/env python3
"""Install only the CRM package and two integration points. No deployment."""
from pathlib import Path
import argparse, shutil, datetime, subprocess
p=argparse.ArgumentParser();p.add_argument('project',type=Path);a=p.parse_args()
root=a.project.resolve();bundle=Path(__file__).resolve().parent
router=root/'router.go';store=root/'services_firestore.go';mod=root/'go.mod'
for f in (router,store,mod):
    if not f.is_file():raise SystemExit(f'Не найден файл {f}; укажите папку исходников Go')
if 'module liikkuvakorjaamo.fi/landing' not in mod.read_text():raise SystemExit('Другой Go module: требуется адаптация импорта перед установкой')
r=router.read_text();s=store.read_text()
old='''ref, _, err := client.Collection(cfg.FirestoreLeadsCollection).Add(ctx, record)
	if err != nil {
		return "", fmt.Errorf("save lead to firestore: %w", err)
	}'''
new='''ref := client.Collection(cfg.FirestoreLeadsCollection).NewDoc()
	batch := client.Batch()
	batch.Create(ref, record)
	if err := crm.AddLeadToBatch(client, batch, ref.ID, lead.Car, lead.Name, lead.Phone, lead.Email, lead.Address, lead.Message, "liikkuva", lead.Language, lead.PhotoURLs); err != nil {
		return "", fmt.Errorf("prepare CRM case: %w", err)
	}
	if _, err := batch.Commit(ctx); err != nil {
		return "", fmt.Errorf("save lead and CRM case: %w", err)
	}'''
installed='registerCRM(mux, cfg)' in r and 'crm.AddLeadToBatch' in s
if not installed:
    if 'return mux' not in r or old not in s:raise SystemExit('Структура проекта изменилась. Файлы не изменены: интеграцию нужно адаптировать к актуальному коду.')
    r=r.replace('return mux','registerCRM(mux, cfg)\n\treturn mux',1)
    s=s.replace('import (','import (\n "liikkuvakorjaamo.fi/landing/crm"',1).replace(old,new,1)
backup=root/('.crm-backup-'+datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%S%fZ'));backup.mkdir()
for f in (router,store,root/'crm_bridge.go'):
    if f.exists():shutil.copy2(f,backup/f.name)
if (root/'crm').exists():shutil.copytree(root/'crm',backup/'crm')
shutil.copytree(bundle/'crm',root/'crm',dirs_exist_ok=True)
shutil.copy2(bundle/'bridge.go.txt',root/'crm_bridge.go')
router.write_text(r);store.write_text(s)
print('CRM установлена. Резервная копия:',backup)
print('Следующий шаг: go test ./... ; перед развёртыванием настройте приватный bucket и секреты по README-RU.md')
