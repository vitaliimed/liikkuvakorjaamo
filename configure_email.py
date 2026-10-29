#!/usr/bin/env python3
"""Configure SMTP on the actual domain-backed Cloud Run service."""
import sys
from update import main, UpdateError

if __name__ == '__main__':
    try:
        main(['--configure-email', *sys.argv[1:]])
    except (UpdateError, KeyboardInterrupt) as exc:
        print('Настройка остановлена: ' + str(exc), file=sys.stderr)
        sys.exit(1)
