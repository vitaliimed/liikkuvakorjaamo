"""Regression guard for the Brevo sender typo that caused undelivered email."""
import unittest

from prod_preflight import authorized_mail_from


class TestLiikkuvaBrevoSender(unittest.TestCase):
    def test_authenticated_sender(self):
        for value in (
            "service@liikkuvakorjaamo.fi",
            "Liikkuva Auto Korjaamo <service@liikkuvakorjaamo.fi>",
            "Liikkuva Auto Korjaamo <SERVICE@LIIKKUVAKORJAAMO.FI>",
        ):
            with self.subTest(value=value):
                self.assertTrue(authorized_mail_from(value))

    def test_typo_rejected(self):
        for value in (
            "service@liikkuvakorjamo.fi",
            "Liikkuva Auto Korjaamo <service@liikkuvakorjamo.fi>",
            "service@liikkuvako rjaamo.fi",
            "service@liikkuvakorjaamo.fi\\r\\nBcc: hostile@example.com",
            "service@liikkuvakorjaamo.fi (unexpected comment)",
            "service@autohelp.fi",
            "service@liikkuvakorjaamo.fi.evil.invalid",
            "",
            None,
        ):
            with self.subTest(value=value):
                self.assertFalse(authorized_mail_from(value))


if __name__ == "__main__":
    unittest.main()
