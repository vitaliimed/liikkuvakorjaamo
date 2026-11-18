import importlib.util
from pathlib import Path
import unittest
from unittest.mock import Mock, patch

spec=importlib.util.spec_from_file_location('resume',Path(__file__).resolve().parents[1]/'scripts/resume-autohelp-release.py')
r=importlib.util.module_from_spec(spec)
spec.loader.exec_module(r)

class ResumeTests(unittest.TestCase):
    def state(self, revision):
        return {'status':{'latestCreatedRevisionName':'new','traffic':[{'revisionName':revision,'percent':100}]}}

    def run_case(self, health, image='digest'):
        with patch.object(r.sys,'argv',['resume','--revision=new','--previous=old','--image-digest=digest']), patch.object(r,'describe',side_effect=[self.state('old'),self.state('new')]), patch.object(r.setup,'cloud',return_value={'status':{'imageDigest':image}}), patch.object(r.setup,'ready_created_revision',return_value='new'), patch.object(r.setup,'request',return_value=health), patch.object(r.time,'sleep'), patch.object(r,'promote') as promote:
            try:
                r.main()
                return promote.call_args_list, None
            except r.setup.SetupError as error:
                return promote.call_args_list, str(error)

    def test_success(self):
        calls,error=self.run_case({'adapter':'autohelp-crm-tariffs-v1','revision':'new'})
        self.assertIsNone(error)
        self.assertEqual([x.args for x in calls],[('new',)])

    def test_mismatched_image_does_not_touch_traffic(self):
        calls,error=self.run_case({},'different')
        self.assertIsNotNone(error)
        self.assertEqual(calls,[])

    def test_failed_domain_check_rolls_back(self):
        calls,error=self.run_case({'adapter':'old','revision':'old'})
        self.assertIsNotNone(error)
        self.assertEqual([x.args for x in calls],[('new',),('old',)])
