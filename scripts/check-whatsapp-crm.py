#!/usr/bin/env python3
"""Read-only audit of Liikkuva CRM WhatsApp ingestion (no client message bodies)."""

import json
import subprocess
import urllib.error
import urllib.request

PROJECT = "liikkuva-korjaamo"
DATABASE = "liikkuvakorjaamo"
BASE = f"https://firestore.googleapis.com/v1/projects/{PROJECT}/databases/{DATABASE}/documents"

def token():
    return subprocess.check_output(["gcloud", "auth", "print-access-token"], text=True).strip()

def request_json(url, token_value, data=None):
    req = urllib.request.Request(
        url,
        data=None if data is None else json.dumps(data).encode(),
        method="GET" if data is None else "POST",
        headers={"Authorization": "Bearer " + token_value, "Content-Type": "application/json"},
    )
    with urllib.request.urlopen(req, timeout=35) as response:
        return json.load(response)

def value(fields, key):
    item = fields.get(key, {})
    if "stringValue" in item:
        return item["stringValue"]
    if "timestampValue" in item:
        return item["timestampValue"]
    return ""

def masked(number):
    n = "".join(x for x in str(number) if x.isdigit())
    return ("…" + n[-4:]) if len(n) >= 4 else "—"

def main():
    access_token = token()
    print("===== FIRESTORE CRM WHATSAPP AUDIT =====")
    for collection in ["crm_whatsapp_inbox", "crm_whatsapp_events", "whatsapp_inbox"]:
        query = {
            "structuredAggregationQuery": {
                "aggregations": [{"alias": "total", "count": {}}],
                "structuredQuery": {"from": [{"collectionId": collection}]},
            }
        }
        try:
            result = request_json(BASE + ":runAggregationQuery", access_token, query)
            total = (result[0].get("result", {})
                     .get("aggregateFields", {})
                     .get("total", {})
                     .get("integerValue", "0"))
            print(f"{collection}: {total}")
        except (urllib.error.HTTPError, IndexError, ValueError) as error:
            print(f"{collection}: query error ({error})")

    print("===== SAMPLE CRM RECORD METADATA (MAX 30) =====")
    try:
        result = request_json(BASE + "/crm_whatsapp_inbox?pageSize=30", access_token)
        docs = result.get("documents", [])
        for doc in docs:
            f = doc.get("fields", {})
            m = f.get("Message", {}).get("mapValue", {}).get("fields", {})
            print(
                "peer=" + masked(value(f, "Peer")),
                "direction=" + value(f, "Direction"),
                "status=" + value(f, "Status"),
                "has_message=" + str(bool(value(m, "ID"))),
                "time=" + value(f, "At"),
            )
        if not docs:
            print("No records returned")
    except urllib.error.HTTPError as error:
        print("Failed to list records, HTTP", error.code)

    print("===== CLOUD RUN WEBHOOK POST (LAST 30) =====")
    log_filter = (
        'resource.type="cloud_run_revision" '
        'resource.labels.service_name="liikkuva-preview" '
        'httpRequest.requestMethod="POST" '
        'httpRequest.requestUrl:"/crm/webhooks/whatsapp" '
        'timestamp>="2026-11-11T13:30:00Z"'
    )
    command = [
        "gcloud", "logging", "read", log_filter,
        "--project=" + PROJECT, "--limit=30", "--format=json",
    ]
    try:
        logs = json.loads(subprocess.check_output(command, text=True, stderr=subprocess.PIPE))
        for item in logs:
            h = item.get("httpRequest", {})
            labels = item.get("resource", {}).get("labels", {})
            print(
                item.get("timestamp", ""),
                "revision=" + labels.get("revision_name", ""),
                "HTTP=" + str(h.get("status", "n/a")),
            )
        if not logs:
            print("No POST calls recorded during the checked period")
    except (subprocess.CalledProcessError, ValueError) as error:
        print("Logs unavailable:", str(error)[:180])

if __name__ == "__main__":
    main()
