import argparse
import importlib.util
import io
from pathlib import Path
import subprocess
import unittest
import urllib.error
from unittest.mock import Mock, patch

spec = importlib.util.spec_from_file_location('crm_telegram_setup', Path(__file__).resolve().parents[1]/'scripts/connect-crm-telegram.py')
setup = importlib.util.module_from_spec(spec)
spec.loader.exec_module(setup)


class TelegramSetupTests(unittest.TestCase):
    def test_retired_ready_revision_is_publishable_despite_old_latest_ready(self):
        state = {'status': {'latestCreatedRevisionName': 'new', 'latestReadyRevisionName': 'old'}}
        revision = {'status': {'conditions': [{'type':'Ready','status':'True','reason':'Retired'}, {'type':'Active','status':'False','reason':'Retired'}]}}
        with patch.object(setup, 'cloud', return_value=revision):
            self.assertEqual(setup.ready_created_revision(state, 'project', 'region'), 'new')

    def test_failed_revision_is_never_promoted(self):
        state = {'status': {'latestCreatedRevisionName': 'new', 'latestReadyRevisionName': 'old'}}
        with patch.object(setup, 'cloud', return_value={'status':{'conditions':[{'type':'Ready','status':'False'}]}}):
            with self.assertRaises(setup.SetupError):
                setup.ready_created_revision(state, 'project', 'region')

    def connector(self):
        return setup.Connector(argparse.Namespace(project='test-project', region='europe-north1',
            service='liikkuva-preview', scheduler_region='europe-west1', owner_telegram_id='1234', apply=True))

    def test_webhook_conflict_and_group_id_stop(self):
        with self.assertRaises(setup.SetupError):
            setup.check_webhook({'url': 'https://another.example/secret-token'}, 'https://expected.example/hook')
        setup.check_webhook({'url': ''}, 'https://expected.example/hook')
        setup.check_webhook({'url': 'https://expected.example/hook'}, 'https://expected.example/hook')
        for value in ['-100123', '0', '+123', 'name', '12\n34']:
            with self.assertRaises(setup.SetupError):
                setup.valid_owner(value)

    def test_split_traffic_is_not_silently_replaced(self):
        with self.assertRaises(setup.SetupError):
            setup.serving_revision({'status': {'traffic': [{'percent': 60, 'revisionName': 'a'}, {'percent': 40, 'revisionName': 'b'}]}})
        self.assertEqual(setup.serving_revision({'status': {'traffic': [{'revisionName': 'old', 'tag': 'old'}, {'percent': 100, 'revisionName': 'new'}]}}), 'new')

    def test_errors_do_not_disclose_tokens(self):
        token = 'do-not-print-secret'
        failure = urllib.error.HTTPError('https://api.telegram.org/bot'+token+'/getMe', 401, token, {}, io.BytesIO())
        with patch.object(setup.urllib.request, 'urlopen', side_effect=failure):
            with self.assertRaises(setup.SetupError) as ctx:
                setup.telegram(token, 'getMe')
        self.assertNotIn(token, str(ctx.exception))
        with patch.object(setup.subprocess, 'run', return_value=subprocess.CompletedProcess([], 1, '', token)):
            with self.assertRaises(setup.SetupError) as ctx:
                setup.cloud('secrets', 'versions', 'add', 'name', stdin=token)
        self.assertNotIn(token, str(ctx.exception))

    def test_secret_uses_stdin_and_pinned_version(self):
        c = self.connector()
        c.runtime = 'runtime@example.test'
        c.google = Mock(return_value={})
        with patch.object(setup, 'cloud', side_effect=['projects/test/secrets/token/versions/7', {}]) as cloud:
            self.assertEqual(c.save_secret('token', 'sensitive-value'), 'token:7')
        self.assertEqual(cloud.call_args_list[0].kwargs['stdin'], 'sensitive-value')
        self.assertNotIn('sensitive-value', str([call.args for call in cloud.call_args_list]))

    def test_owner_binding_cannot_claim_another_users_id(self):
        c = self.connector()
        c.email, c.owner_id = 'owner@example.test', '1234'
        owner = {'name': 'users/owner', 'fields': {'Role': {'stringValue': 'owner'}, 'Active': {'booleanValue': True}}}
        c.users = Mock(side_effect=[[owner], [{'name': 'users/mechanic'}]])
        with self.assertRaises(setup.SetupError):
            c.check_owner()

    def test_same_secret_reuses_pinned_version(self):
        c = self.connector()
        c.runtime = 'runtime@example.test'
        c.google = Mock(side_effect=[{}, {'state': 'ENABLED', 'name': 'projects/test/secrets/token/versions/7'}])
        with patch.object(setup, 'cloud', side_effect=['same-secret', {}]) as cloud:
            self.assertEqual(c.save_secret('token', 'same-secret'), 'token:7')
        self.assertFalse(any('add' in call.args for call in cloud.call_args_list))

    def test_owner_binding_only_patches_id_in_transaction(self):
        c = self.connector()
        c.owner_id = '1234'
        c.check_owner = Mock(return_value={'name': 'projects/test/databases/db/documents/crm_users/owner', 'updateTime': 'version'})
        c.google = Mock(side_effect=[{'transaction': 'transaction-id'}, {}])
        c.bind_owner()
        c.check_owner.assert_called_once_with('transaction-id')
        body = c.google.call_args_list[1].args[2]
        self.assertEqual(body['transaction'], 'transaction-id')
        self.assertEqual(body['writes'][0]['updateMask']['fieldPaths'], ['TelegramID'])
        self.assertEqual(body['writes'][0]['currentDocument'], {'updateTime': 'version'})

    def test_failed_binding_rolls_back(self):
        c = self.connector()
        c.check_owner = Mock(side_effect=setup.SetupError('conflict'))
        c.google = Mock(side_effect=[{'transaction': 'transaction-id'}, {}])
        with self.assertRaises(setup.SetupError):
            c.bind_owner()
        self.assertTrue(c.google.call_args.args[0].endswith(':rollback'))

    def test_default_run_never_applies(self):
        c = Mock()
        with patch.object(setup, 'Connector', return_value=c):
            setup.main(['--owner-telegram-id=1234'])
        c.preflight.assert_called_once()
        c.apply.assert_not_called()


if __name__ == '__main__':
    unittest.main()
