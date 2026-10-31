import contextlib
import io
import subprocess
import unittest
import urllib.error
from unittest.mock import Mock, patch
import update


class IntegrationTests(unittest.TestCase):
    def updater(self):
        value = update.Updater(update.parse_args([]))
        value.runtime_account = 'liikkuva-runtime@example.test'
        return value

    def test_shared_telegram_setup_only_checks_metadata_and_saves_secrets_on_stdin(self):
        updater = self.updater()
        token = '123456:' + 'a' * 30
        calls = []
        methods = []
        def gc(*args, **kwargs):
            calls.append((args, kwargs))
            return subprocess.CompletedProcess(args, 0, 'projects/test/secrets/example/versions/5' if args[:3] == ('secrets', 'versions', 'add') else '', '')
        def metadata(value, method, data=None):
            self.assertEqual(value, token)
            methods.append(method)
            return {'is_bot': True, 'username': 'same_autohelp_bot'} if method == 'getMe' else {'id': -100123}
        updater.gc = gc
        updater.telegram_metadata = metadata
        with patch.object(update.getpass, 'getpass', return_value=token), patch('builtins.input', return_value='-100123'), contextlib.redirect_stdout(io.StringIO()):
            updater.configure_telegram()
        self.assertEqual(methods, ['getMe', 'getChat'])
        self.assertEqual(updater.secret_updates, {'TELEGRAM_BOT_TOKEN': 'liikkuva-telegram-bot-token:5', 'TELEGRAM_CHAT_ID': 'liikkuva-telegram-chat-id:5'})
        self.assertFalse(any(token in str(args) for args, kwargs in calls))
        self.assertTrue(any(kwargs.get('input') == token for args, kwargs in calls))

    def test_telegram_failure_never_exposes_token_or_modifies_cloud(self):
        token = '123456:' + 's' * 30
        error = urllib.error.HTTPError('https://api.telegram.org/bot'+token+'/getMe', 401, 'Unauthorized', {}, io.BytesIO())
        with patch.object(update.urllib.request, 'urlopen', side_effect=error):
            with self.assertRaises(update.UpdateError) as caught:
                update.Updater.telegram_metadata(token, 'getMe')
        self.assertNotIn(token, str(caught.exception))
        with self.assertRaises(update.UpdateError):
            update.Updater.telegram_metadata(token, 'setWebhook')
        updater = self.updater()
        updater.gc = Mock()
        updater.telegram_metadata = Mock(side_effect=update.UpdateError('invalid bot'))
        with patch.object(update.getpass, 'getpass', return_value=token), patch('builtins.input', return_value='-100123'), contextlib.redirect_stdout(io.StringIO()):
            with self.assertRaises(update.UpdateError):
                updater.configure_telegram()
        updater.gc.assert_not_called()

    def test_status_report_is_read_only_and_never_prints_secrets(self):
        updater = Mock(entries={})
        with patch.object(update, 'Updater', return_value=updater), patch.object(update.shutil, 'which', return_value='/bin/gcloud'):
            update.main(['--check-integrations'])
        updater.discover.assert_called_once()
        updater.integration_report.assert_called_once()
        updater.prepare_storage.assert_not_called()
        updater.deploy.assert_not_called()
        updater.build_cli.assert_not_called()
        value = self.updater()
        value.entries = {'TELEGRAM_BOT_TOKEN': {'value': 'DO-NOT-PRINT'}, 'TELEGRAM_CHAT_ID': {'valueFrom': {'secretKeyRef': {'name': 'hidden-chat'}}}}
        value.updates = {'SMTP_HOST': 'pending.not.deployed'}
        output = io.StringIO()
        with contextlib.redirect_stdout(output): value.integration_report()
        self.assertIn('Telegram: задано', output.getvalue())
        self.assertNotIn('DO-NOT-PRINT', output.getvalue())
        self.assertNotIn('hidden-chat', output.getvalue())
        self.assertIn('SMTP_HOST', output.getvalue())
        self.assertNotIn('pending.not.deployed', output.getvalue())

    def test_enabling_intake_uses_new_credentials_and_rejects_empty_values(self):
        updater = self.updater()
        keys = ['INTERNAL_TASK_TOKEN', 'REVIEW_TOKEN_SECRET', 'GOOGLE_APPS_SCRIPT_URL', 'GOOGLE_APPS_SCRIPT_TOKEN', 'CLOUD_TASKS_PROJECT_ID', 'CLOUD_TASKS_LOCATION', 'CLOUD_TASKS_QUEUE', 'GCS_BUCKET_NAME', 'INTERNAL_PROCESS_LEAD_URL']
        updater.entries = {key: {'value': 'synthetic-configured'} for key in keys}
        updater.secret_updates = {'TELEGRAM_BOT_TOKEN': 'token:1', 'TELEGRAM_CHAT_ID': 'chat:1'}
        updater.check_intake()
        self.assertEqual(updater.updates['LIIKKUVA_BOOTSTRAP'], '0')
        updater.entries['GCS_BUCKET_NAME'] = {'value': ''}
        with self.assertRaisesRegex(update.UpdateError, 'GCS_BUCKET_NAME'): updater.check_intake()

    def test_setup_flags_cannot_be_combined_with_resume(self):
        with self.assertRaises(update.UpdateError): update.main(['--resume', '--configure-telegram'])
        with self.assertRaises(update.UpdateError): update.main(['--check-integrations', '--configure-telegram'])
