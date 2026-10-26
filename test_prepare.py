import unittest
from pathlib import Path
from prepare import replace_function

class SafeTransformTests(unittest.TestCase):
 def test_nested_braces_comments_and_strings(self):
  source='''package main
func (h Handler) Submit() { s := "}"; _ = s; /* } */ if true { f := func(){println("{")}; f() } }
func next(){println("keep")}
'''
  updated=replace_function(source,'Submit','func (h Handler) Submit(){println("new")}')
  self.assertIn('func next(){println("keep")}',updated)
  self.assertNotIn('s :=',updated)
 def test_missing_function_is_not_guessed(self):
  with self.assertRaises(RuntimeError):replace_function('package main','Submit','bad')
 def test_actual_lead_methods_are_bounded(self):
  fixture=Path(__file__).resolve().parent/'source/handlers_lead.go'
  if not fixture.exists():self.skipTest('Fixture is available only during release validation')
  source=fixture.read_text()
  for name in ['Submit','handleAsyncSubmit']:
   updated=replace_function(source,name,'func replacement() {}')
   self.assertIn('func buildLeadFromRequest',updated)
   self.assertIn('func (h LeadHandler) ProcessLead',updated)
   self.assertEqual(updated.count('func replacement() {}'),1)

if __name__=='__main__':unittest.main()
