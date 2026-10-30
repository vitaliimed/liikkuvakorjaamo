import importlib.util
import io
import json
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path
import subprocess
import unittest
from unittest.mock import patch, Mock

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
        updater = update.Updater(update.parse_args(['--check-timeout', '30']))
        updater.service = 'liikkuva-preview'
        updater.updates = {'ADMIN_RECOVERY_EMAIL': 'medynikv@gmail.com'}
        events = []
        revision = 'liikkuva-preview-00042-test'
        def gc(*args, **kwargs):
            events.append(('gcloud', args, kwargs))
            if args[:2] == ('auth', 'print-identity-token'):
                return subprocess.CompletedProcess(args, 1, '', '')
            return subprocess.CompletedProcess(args, 0, json.dumps({'status': {'latestCreatedRevisionName': revision}}), '')
        updater.gc = gc
        def describe():
            promoted = any(e[0] == 'gcloud' and '--to-revisions=' + revision + '=100' in e[1] for e in events)
            tag = next((a.split('=', 1)[1] for e in events if e[0] == 'gcloud' for a in e[1] if a.startswith('--tag=')), 'check-20261030-100000')
            status = {'latestCreatedRevisionName': revision, 'latestReadyRevisionName': revision,
                      'conditions': [{'type': 'Ready', 'status': 'True'}, {'type': 'RoutesReady', 'status': 'True'}],
                      'url': 'https://liikkuva-preview-test.europe-north1.run.app', 'observedGeneration': 2,
                      'traffic': [{'tag': tag, 'url': 'https://' + tag + '---liikkuva-preview-test.europe-north1.run.app', 'revisionName': revision, 'percent': 100 if promoted else 0}]}
            return {'metadata': {'generation': 2}, 'status': status}
        updater.describe = describe
        def endpoint(url, identity_token=None, **kwargs):
            events.append(('check', url, identity_token))
            if '/api/health' in url:
                return json.dumps({'ok': True, 'site': 'liikkuvakorjaamo.fi', 'version': update.VERSION if health_ok else 'old-version', 'revision': revision})
            return 'Войти в кабинет' if '/admin' in url else '.hero .hero-layout'
        updater.endpoint = endpoint
        return updater, events

    def test_failed_health_check_never_promotes(self):
        updater, events = self.make_updater(False)
        with self.assertRaises(update.UpdateError):
            updater.deploy()
        self.assertFalse(any(e[0] == 'gcloud' and any('--to-revisions=' in a for a in e[1]) for e in events))

    def test_checks_before_promoting_and_preserves_unrelated_configuration(self):
        updater, events = self.make_updater()
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

    def test_tag_404_then_ready_is_retried_before_promotion(self):
        updater, events = self.make_updater()
        original = updater.endpoint
        attempts = []
        def endpoint(url, **kwargs):
            if '.run.app/api/health' in url:
                attempts.append(url)
                if len(attempts) <= 2:
                    raise update.EndpointError(404, url)
            return original(url, **kwargs)
        updater.endpoint = endpoint
        clock = FakeClock()
        with patch.object(update.time, 'monotonic', clock.monotonic), patch.object(update.time, 'sleep', clock.sleep):
            updater.deploy()
        self.assertEqual(len(attempts), 3)
        self.assertTrue(any(e[0] == 'gcloud' and any(a.startswith('--to-revisions=') for a in e[1]) for e in events))
        self.assertGreater(clock.now, 0)

    def test_private_probe_authenticates_health_admin_and_css(self):
        updater, events = self.make_updater()
        original_gc, original_endpoint = updater.gc, updater.endpoint
        def gc(*args, **kwargs):
            if args[:2] == ('auth', 'print-identity-token'):
                return subprocess.CompletedProcess(args, 0, 'synthetic.jwt.token', '')
            return original_gc(*args, **kwargs)
        updater.gc = gc
        def endpoint(url, identity_token=None, **kwargs):
            if '.run.app/' in url and identity_token is None:
                raise update.EndpointError(403, url)
            return original_endpoint(url, identity_token=identity_token, **kwargs)
        updater.endpoint = endpoint
        updater.deploy()
        marker_calls = [e for e in events if e[0] == 'check' and ('/admin' in e[1] or '/static/' in e[1])]
        self.assertEqual(len(marker_calls), 2)
        self.assertTrue(all(e[2] == 'synthetic.jwt.token' for e in marker_calls))
        self.assertFalse(any('--allow-unauthenticated' in e[1] for e in events if e[0] == 'gcloud'))

    def test_permanent_404_never_promotes_and_diagnostics_hide_secrets(self):
        updater, events = self.make_updater()
        describe = updater.describe
        def state():
            data = describe()
            data['spec'] = {'template': {'spec': {'containers': [{'env': [{'name': 'SMTP_PASSWORD', 'value': 'MUST-NOT-APPEAR'}]}]}}}
            return data
        updater.describe = state
        updater.endpoint = lambda url, **kwargs: (_ for _ in ()).throw(update.EndpointError(404, url))
        clock = FakeClock()
        output = io.StringIO()
        with patch.object(update.time, 'monotonic', clock.monotonic), patch.object(update.time, 'sleep', clock.sleep), patch('sys.stdout', output):
            with self.assertRaisesRegex(update.UpdateError, '--resume'):
                updater.deploy()
        self.assertNotIn('MUST-NOT-APPEAR', output.getvalue())
        self.assertIn('revision', output.getvalue())
        self.assertFalse(any(e[0] == 'gcloud' and any(a.startswith('--to-revisions=') for a in e[1]) for e in events))

    def test_resume_checks_existing_revision_without_new_cloud_build(self):
        updater, events = self.make_updater()
        updater.resume()
        self.assertFalse(any(e[0] == 'gcloud' and e[1][:2] == ('run', 'deploy') for e in events))
        self.assertTrue(any(e[0] == 'gcloud' and any(a.startswith('--to-revisions=') for a in e[1]) for e in events))

    def test_concurrent_deployment_stops_before_switch(self):
        updater, events = self.make_updater()
        describe = updater.describe
        def state():
            data = describe()
            data['status']['latestCreatedRevisionName'] = 'different-deployment'
            return data
        updater.describe = state
        with self.assertRaisesRegex(update.UpdateError, 'другая ревизия'):
            updater.deploy()
        self.assertFalse(any(e[0] == 'gcloud' and any(a.startswith('--to-revisions=') for a in e[1]) for e in events))

    def test_routes_ready_and_observed_generation_must_catch_up(self):
        updater, _ = self.make_updater()
        describe = updater.describe
        reads = []
        def state():
            data = describe()
            reads.append(1)
            if len(reads) == 1:
                data['status']['observedGeneration'] = 1
                data['status']['conditions'][1]['status'] = 'Unknown'
            return data
        updater.describe = state
        clock = FakeClock()
        with patch.object(update.time, 'monotonic', clock.monotonic), patch.object(update.time, 'sleep', clock.sleep):
            updater.deploy()
        self.assertGreaterEqual(len(reads), 3)
        self.assertGreater(clock.now, 0)

    def test_documented_alternative_run_url_can_be_used(self):
        updater, _ = self.make_updater()
        describe = updater.describe
        original = updater.endpoint
        calls = []
        def state():
            data = describe()
            route = data['status']['traffic'][0]
            route['url'] = 'https://' + route['tag'] + '---random-hash-lz.a.run.app'
            return data
        updater.describe = state
        def endpoint(url, **kwargs):
            calls.append(url)
            if 'random-hash-lz.a.run.app' in url:
                raise update.EndpointError(404, url)
            return original(url, **kwargs)
        updater.endpoint = endpoint
        updater.deploy()
        self.assertTrue(any('---liikkuva-preview-test.europe-north1.run.app/api/health' in u for u in calls))
        self.assertFalse(any(u.startswith('https://liikkuva-preview-test.europe-north1.run.app/') for u in calls))

    def test_endpoint_identity_is_only_sent_to_run_app(self):
        updater, _ = self.make_updater()
        opener = Mock()
        opener.open.return_value = io.BytesIO(b'OK')
        transport = update.Updater.endpoint.__get__(updater)
        with patch.object(update.urllib.request, 'build_opener', return_value=opener):
            self.assertEqual(transport('https://check---svc.run.app/api/health', identity_token='synthetic.jwt.token'), 'OK')
            request = opener.open.call_args.args[0]
            self.assertEqual(request.get_header('X-serverless-authorization'), 'Bearer synthetic.jwt.token')
            with self.assertRaises(update.UpdateError):
                transport('https://untrusted.example/api/health', identity_token='synthetic.jwt.token')
        self.assertEqual(opener.open.call_count, 1)

    def test_endpoint_does_not_follow_redirects(self):
        hits = []
        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                hits.append(self.path)
                if self.path == '/api/health':
                    self.send_response(302)
                    self.send_header('Location', '/other')
                    self.end_headers()
                else:
                    self.send_response(200)
                    self.end_headers()
                    self.wfile.write(b'not expected')
            def log_message(self, *args):
                pass
        server = HTTPServer(('127.0.0.1', 0), Handler)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        updater, _ = self.make_updater()
        transport = update.Updater.endpoint.__get__(updater)
        try:
            with self.assertRaises(update.EndpointError) as caught:
                transport(f'http://127.0.0.1:{server.server_port}/api/health')
            self.assertEqual(caught.exception.code, 302)
            self.assertEqual(hits, ['/api/health'])
        finally:
            server.shutdown()
            server.server_close()
            thread.join()

    def test_real_http_404_responses_recover_before_traffic_switch(self):
        updater, events = self.make_updater()
        health_reads = []
        health = json.dumps({'ok': True, 'site': 'liikkuvakorjaamo.fi', 'version': update.VERSION,
                             'revision': 'liikkuva-preview-00042-test'})
        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                if self.path == '/api/health':
                    health_reads.append(1)
                    code, body = (404, 'not ready') if len(health_reads) <= 2 else (200, health)
                elif self.path == '/admin':
                    code, body = 200, 'Войти в кабинет'
                else:
                    code, body = 200, '.hero .hero-layout'
                self.send_response(code)
                self.end_headers()
                self.wfile.write(body.encode())
            def log_message(self, *args):
                pass
        server = HTTPServer(('127.0.0.1', 0), Handler)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        base = f'http://127.0.0.1:{server.server_port}'
        updater.candidate_urls = lambda route, deployed: [base]
        transport = update.Updater.endpoint.__get__(updater)
        def endpoint(url, **kwargs):
            events.append(('check', url))
            return health if url.startswith(update.DOMAIN) else transport(url, **kwargs)
        updater.endpoint = endpoint
        try:
            with patch.object(update.time, 'sleep', return_value=None):
                updater.deploy()
            self.assertEqual(len(health_reads), 3)
            self.assertTrue(any(e[0] == 'gcloud' and any(a.startswith('--to-revisions=') for a in e[1]) for e in events))
        finally:
            server.shutdown()
            server.server_close()
            thread.join()

    def test_wrong_revision_is_not_accepted_during_route_propagation(self):
        updater, events = self.make_updater()
        original = updater.endpoint
        calls = []
        def endpoint(url, **kwargs):
            if '.run.app/api/health' in url:
                calls.append(1)
                if len(calls) == 1:
                    return json.dumps({'ok': True, 'site': 'liikkuvakorjaamo.fi', 'version': update.VERSION,
                                       'revision': 'previous-revision'})
            return original(url, **kwargs)
        updater.endpoint = endpoint
        clock = FakeClock()
        with patch.object(update.time, 'monotonic', clock.monotonic), patch.object(update.time, 'sleep', clock.sleep):
            updater.deploy()
        self.assertEqual(len(calls), 2)

    def test_disabled_run_url_is_reported_without_changing_access(self):
        updater, events = self.make_updater()
        describe = updater.describe
        def state():
            data = describe()
            data['metadata']['annotations'] = {'run.googleapis.com/default-url-disabled': 'true'}
            return data
        updater.describe = state
        with patch('sys.stdout', io.StringIO()):
            with self.assertRaisesRegex(update.UpdateError, 'выключены'):
                updater.deploy()
        self.assertFalse(any(e[0] == 'gcloud' and any(a.startswith('--to-revisions=') or a.startswith('--ingress=') for a in e[1]) for e in events))

    def test_failed_resume_does_not_ask_for_unsaved_password(self):
        updater = Mock()
        updater.resume.side_effect = update.UpdateError('unreachable revision')
        with patch.object(update, 'Updater', return_value=updater), patch.object(update.shutil, 'which', return_value='/bin/gcloud'), patch.object(update, 'prompt_password') as prompt:
            with self.assertRaises(update.UpdateError):
                update.main(['--resume', '--set-password'])
            prompt.assert_not_called()
            updater.set_password.assert_not_called()

    def test_password_prompt_follows_successful_verification(self):
        updater = Mock(entries={})
        events = []
        updater.resume.side_effect = lambda: events.append('verified')
        updater.set_password.side_effect = lambda *args: events.append('password saved')
        def prompt():
            events.append('password prompt')
            return 'synthetic-new-password'
        with patch.object(update, 'Updater', return_value=updater), patch.object(update.shutil, 'which', return_value='/bin/gcloud'), patch.object(update, 'prompt_password', side_effect=prompt), patch('sys.stdout', io.StringIO()):
            update.main(['--resume', '--set-password'])
        self.assertEqual(events, ['verified', 'password prompt', 'password saved'])

    def test_error_commands_use_this_checkout_even_from_old_folder(self):
        command = update.update_command('--resume', '--set-password')
        self.assertEqual(update.shlex.split(command), ['python3', str(ROOT / 'update.py'), '--resume', '--set-password'])

    def test_blocked_legacy_health_path_does_not_block_new_deployment(self):
        updater, events = self.make_updater()
        hits = []
        health = json.dumps({'ok': True, 'site': 'liikkuvakorjaamo.fi', 'version': update.VERSION,
                             'revision': 'liikkuva-preview-00042-test'})
        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                hits.append(self.path)
                if self.path == '/healthz':
                    code, body = 404, '<html>Google frontend 404</html>'
                elif self.path == '/api/health':
                    code, body = 200, health
                elif self.path == '/admin':
                    code, body = 200, 'Войти в кабинет'
                else:
                    code, body = 200, '.hero .hero-layout'
                self.send_response(code)
                self.end_headers()
                self.wfile.write(body.encode())
            def log_message(self, *args):
                pass
        server = HTTPServer(('127.0.0.1', 0), Handler)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        base = f'http://127.0.0.1:{server.server_port}'
        updater.candidate_urls = lambda route, deployed: [base]
        transport = update.Updater.endpoint.__get__(updater)
        public_checks = []
        def endpoint(url, **kwargs):
            if url.startswith(update.DOMAIN):
                public_checks.append(url)
                return health
            return transport(url, **kwargs)
        updater.endpoint = endpoint
        try:
            with self.assertRaises(update.EndpointError):
                transport(base + '/healthz')
            updater.deploy()
            self.assertEqual(hits, ['/healthz', '/api/health', '/admin', '/static/css/responsive.css?v=20261029'])
            self.assertTrue(public_checks and all('/api/health?' in u for u in public_checks))
            self.assertTrue(any(e[0] == 'gcloud' and any(a.startswith('--to-revisions=') for a in e[1]) for e in events))
        finally:
            server.shutdown()
            server.server_close()
            thread.join()


class FakeClock:
    def __init__(self):
        self.now = 0
    def monotonic(self):
        return self.now
    def sleep(self, seconds):
        self.now += seconds


if __name__ == '__main__':
    unittest.main()
