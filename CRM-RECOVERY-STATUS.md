# Recovery checkpoint

Existing integrated Go CRM, base 3438a774350871a036b676e1f71ae679edc35b7e.

Restored server fixes: owner-only draft and messaging, role-filtered event responses without snapshots, technical history without prior customer contact or payments, technical/internal note actions. Added projection regression test.

This is an incomplete recovery checkpoint, NOT a deploy-ready release. The execution environment lost the earlier uncommitted working directory. Earlier test results do not validate this checkpoint. UI adaptations, full HTTP/emulator tests, vehicle IDs, schedules, prices, photographs, mail interfaces and deployment must be restored/completed before publication. No Cloud Run revision or traffic changed. No customer notifications sent.
