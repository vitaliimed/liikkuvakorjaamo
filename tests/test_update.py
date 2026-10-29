import importlib.util
import io
from pathlib import Path
import subprocess
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('liikkuva_update', ROOT / 'update.py')
update = importlib.util.module_from_spec(spec)
spec.loader.exec_module(update)


def service(name):
    return {'metadata': {'name': name}}


class UpdateTests(unittest.TestCase):
    def test_live_hosting_selects_actual_service_and_rejects_wrong_override(self):
        services = [service('liikkuva-preview'), service('liikkuva-landing'), service('autohelp-landing')]
        self.assertEqual(update.select_service(services, {'serviceId': 'liikkuva-preview', 'region': 'europe-north1'}), 'liikkuva-preview')
        with self.assertRaises(update.UpdateError):
            update.select_service(services, {'serviceId': 'liikkuva-preview'}, 'liikkuva-landing')
        with self.assertRaises(update.UpdateError):
            update.select_service(services)
        with self.assertRaises(update.UpdateError):
            update.select_service(services, {'serviceId': 'liikkuva-preview', 'region': 'us-central1'})

    def test_password_limits_match_server_and_preserve_spaces(self):
        update.validate_password('  long passphrase here  ')
        for value in ('short', 'я' * 40, 'new\npassword\n1234'):
            with self.assertRaises(update.UpdateError):
                update.validate_password(value)

    def make_updater(self, health_ok=True):
        updater = update.Updater(update.parse_args([]))
        updater.service = 'liikkuva-preview'
        updater.updates = {'ADMIN_RECOVERY_EMAIL': 'medynikv@gmail.com'}
        events = []
        revision = 'liikkuva-preview-00042-test'
        def gc(*args, **kwargs):
            events.append(('gcloud', args, kwargs))
            return subprocess.CompletedProcess(args, 0, '{}', '')
        updater.gc = gc
        def describe():
            if any(e[0] == 'gcloud' and '--to-revisions=' + revision + '=100' in e[1] for e in events):
                return {'status': {'traffic': [{'revisionName': revision, 'percent': 100}]}}
            tag = next(a.split('=', 1)[1] for e in events if e[0] == 'gcloud' for a in e[1] if a.startswith('--tag='))
            return {'status': {'traffic': [{'tag': tag, 'url': 'https://checked.example.test', 'revisionName': revision}]}}
        updater.describe = describe
        def api(url, **kwargs):
            events.append(('check', url))
            return {'version': update.VERSION if health_ok else 'old-version', 'revision': revision}
        updater.api = api
        return updater, events

    def test_failed_health_check_never_promotes(self):
        updater, events = self.make_updater(False)
        with self.assertRaises(update.UpdateError):
            updater.deploy()
        self.assertFalse(any(e[0] == 'gcloud' and any('--to-revisions=' in a for a in e[1]) for e in events))

    def test_checks_before_promoting_and_preserves_unrelated_configuration(self):
        updater, events = self.make_updater()
        def open_url(url, **kwargs):
            return io.BytesIO(('Войти в кабинет' if '/admin' in url else '.hero .hero-layout').encode())
        with patch.object(update.urllib.request, 'urlopen', side_effect=open_url):
            updater.deploy()
        deploy = next(e for e in events if e[0] == 'gcloud' and e[1][:2] == ('run', 'deploy'))
        self.assertIn('--no-traffic', deploy[1])
        self.assertTrue(any(a.startswith('--update-env-vars=^~^') for a in deploy[1]))
        self.assertFalse(any(a.startswith(('--set-env-vars=', '--env-vars-file=', '--set-secrets=')) for a in deploy[1]))
        checked = next(i for i, e in enumerate(events) if e[0] == 'check')
        promoted = next(i for i, e in enumerate(events) if e[0] == 'gcloud' and any(a.startswith('--to-revisions=') for a in e[1]))
        self.assertLess(checked, promoted)

    def test_smtp_migrates_only_named_plain_values_to_secret_refs(self):
        updater, events = self.make_updater()
        updater.entries = {
            'SMTP_USERNAME': {'value': 'old-login'},
            'SMTP_PASSWORD': {'value': 'old-key'},
            'TELEGRAM_BOT_TOKEN': {'valueFrom': {'secretKeyRef': {'name': 'keep-telegram'}}},
        }
        updater.secret_updates = {'SMTP_USERNAME': 'smtp-login:2', 'SMTP_PASSWORD': 'smtp-password:3'}
        with patch.object(update.urllib.request, 'urlopen', side_effect=lambda *a, **k: io.BytesIO('Войти в кабинет .hero .hero-layout'.encode())):
            updater.deploy()
        deploy = next(e for e in events if e[0] == 'gcloud' and e[1][:2] == ('run', 'deploy'))
        self.assertIn('--remove-env-vars=SMTP_USERNAME,SMTP_PASSWORD', deploy[1])
        self.assertFalse(any('TELEGRAM_BOT_TOKEN' in a for a in deploy[1]))

    def test_operator_password_and_token_stay_out_of_arguments(self):
        updater, _ = self.make_updater()
        updater.token = 'synthetic-operator-token'
        updater.gc = lambda *args, **kwargs: subprocess.CompletedProcess(args, 0, 'synthetic-operator-token\n', '')
        updater.entries = {'SITE_SETTINGS_COLLECTION': {'value': 'liikkuva_settings'}}
        with patch.object(update, 'run') as call:
            updater.set_password('/tmp/server', 'new-console-password-123')
        args, kwargs = call.call_args
        self.assertEqual(args, ('/tmp/server', 'admin-password'))
        self.assertEqual(kwargs['input'], 'new-console-password-123\n')
        self.assertEqual(kwargs['env']['SITE_SETTINGS_COLLECTION'], 'liikkuva_settings')
        self.assertEqual(kwargs['env']['LIIKKUVA_OPERATOR_TOKEN'], 'synthetic-operator-token')
        self.assertEqual(kwargs['env']['ADMIN_PASSWORD'], '')

    def test_secure_rules_target_only_named_database(self):
        updater, _ = self.make_updater()
        updater.project_number = '123456789'
        updater.entries = {}
        calls = []
        def api(url, method='GET', data=None, authenticated=True):
            calls.append((url, method, data, authenticated))
            if not authenticated:
                raise update.APIError(403, url)
            if method == 'GET':
                raise update.APIError(404, url)
            if url.endswith('/rulesets'):
                return {'name': 'projects/liikkuva-korjaamo/rulesets/test-rule'}
            return {}
        updater.api = api
        updater.secure_database()
        release = next(c for c in calls if c[0].endswith('/releases'))
        self.assertTrue(release[2]['name'].endswith('/cloud.firestore/liikkuvakorjaamo'))
        self.assertNotIn('(default)', release[2]['name'])
        ruleset = next(c for c in calls if c[0].endswith('/rulesets'))
        self.assertIn('allow read, write: if false', ruleset[2]['source']['files'][0]['content'])


if __name__ == '__main__':
    unittest.main()
