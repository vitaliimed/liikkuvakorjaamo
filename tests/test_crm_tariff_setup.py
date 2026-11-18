import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('tariff_setup', Path(__file__).resolve().parents[1]/'scripts/connect-autohelp-tariffs.py')
setup = importlib.util.module_from_spec(spec)
spec.loader.exec_module(setup)


class TariffSetupTests(unittest.TestCase):
    def test_requires_published_ready_revision(self):
        state = {'status': {'traffic': [{'percent': 100, 'revisionName': 'live'}],
                            'latestCreatedRevisionName': 'new', 'latestReadyRevisionName': 'live'}}
        with self.assertRaises(setup.setup.SetupError):
            setup.require_current(state)
        state['status']['latestCreatedRevisionName'] = 'live'
        self.assertEqual(setup.require_current(state), 'live')

    def test_cannot_overwrite_a_split_release(self):
        state = {'status': {'traffic': [{'percent': 50, 'revisionName': 'a'}, {'percent': 50, 'revisionName': 'b'}]}}
        with self.assertRaises(setup.setup.SetupError):
            setup.require_current(state)


if __name__ == '__main__':
    unittest.main()
