import json
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer

import pytest

from noetivela import (
    ChatStreamError,
    Client,
    Contract,
    Policy,
    UnsatisfiableError,
)


class MockGateway(BaseHTTPRequestHandler):
    def log_message(self, *args):  # silence
        pass

    def _send(self, code, obj):
        body = json.dumps(obj).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_POST(self):
        if self.path == "/v1/chat/completions":
            length = int(self.headers.get("Content-Length", 0))
            body = json.loads(self.rfile.read(length) or b"{}")
            assert self.headers.get("X-Noetivela-Contract"), "contract header required"
            if body.get("stream"):
                # SSE response
                payload = b"".join([
                    b"data: {\"choices\":[{\"delta\":{\"content\":\"Hel\"},\"finish_reason\":\"\"}]}\n\n",
                    b"data: {\"choices\":[{\"delta\":{\"content\":\"lo\"},\"finish_reason\":\"stop\"}],\"usage\":{\"output_tokens\":2}}\n\n",
                    b"data: {\"done\":true}\n\n",
                ])
                self.send_response(200)
                self.send_header("Content-Type", "text/event-stream")
                self.send_header("Content-Length", str(len(payload)))
                self.end_headers()
                self.wfile.write(payload)
                return
            self._send(200, {
                "choices": [{"message": {"content": "ok"}, "finish_reason": "stop"}],
                "usage": {"input_tokens": 100, "output_tokens": 20},
                "noetivela": {"decision_id": "abc123", "model_ref": "m1",
                              "endpoint_ref": "e1", "tcos": {"total": 0.0002}},
            })
        elif self.path == "/v1/contracts/validate":
            self._send(200, {"valid": True, "satisfiable": True, "policy": "legal",
                             "candidates": [{"model_ref": "m1", "endpoint_ref": "e1", "eligible": True, "score": 0.8}]})
        elif self.path == "/v1/eligible":
            self._send(200, {"ranked": [{"model_ref": "m1", "endpoint_ref": "e1", "eligible": True, "score": 0.8}]})
        elif self.path == "/v1/embeddings":
            self._send(200, {"data": [[0.1, 0.2], [0.3, 0.4]], "dimensions": 2,
                             "usage": {"input_tokens": 4}, "noetivela": {"endpoint_ref": "e1"}})
        elif self.path == "/v1/replay":
            self._send(200, {"records": 10, "savings_percent": 50.0, "verdict": "PASS",
                             "baseline": {"strategy": "fixed:exp", "quality_adjusted_tcos": 0.02, "served_count": 10, "denied_count": 0},
                             "routed": {"strategy": "policy_routed", "quality_adjusted_tcos": 0.01, "served_count": 10, "denied_count": 0}})
        elif self.path == "/v1/train":
            length = int(self.headers.get("Content-Length", 0))
            body = json.loads(self.rfile.read(length) or b"{}")
            assert body.get("version_id") == "v2", "version_id required"
            assert body.get("tcos_weight") == 0.5 and body.get("min_samples") == 10
            samples = body.get("samples")
            assert samples and len(samples) == 2, "dataset samples required"
            assert {"features", "candidate_ref", "quality", "tcos"} <= set(samples[0])
            assert samples[0]["features"]["task"] == "contract_clause_extraction"
            self._send(200, {"version_id": body["version_id"], "trained_samples": 2,
                             "validate_mae": 0.12, "min_confidence": 0.6,
                             "artifact_digest": "sha256:0123456789abcdef0123456789abcdef"})
        else:
            # unsatisfiable contract
            self._send(422, {"error": {"type": "unsatisfiable_contract",
                                       "message": "no eligible candidate",
                                       "failed_gates": [{"gate": "policy", "reason": "region_not_allowed"}]}})


@pytest.fixture(scope="module")
def srv():
    server = HTTPServer(("127.0.0.1", 0), MockGateway)
    t = threading.Thread(target=server.serve_forever, daemon=True)
    t.start()
    yield server.server_address[1]
    server.shutdown()


def make_contract():
    return Contract(
        task="contract_clause_extraction",
        modality=["text"],
        policy=Policy(allowed_regions=["sg"], provider_training="prohibited"),
    )


def test_chat(srv):
    c = Client(f"http://127.0.0.1:{srv}", contract=make_contract())
    resp = c.chat("auto", [{"role": "user", "content": "hi"}])
    assert resp.content == "ok"
    assert resp.decision_id == "abc123"
    assert resp.endpoint_ref == "e1"
    assert resp.usage["input_tokens"] == 100


def test_chat_stream(srv):
    c = Client(f"http://127.0.0.1:{srv}", contract=make_contract())
    out = "".join(c.chat_stream("auto", [{"role": "user", "content": "hi"}]))
    assert out == "Hello"


def test_validate(srv):
    c = Client(f"http://127.0.0.1:{srv}")
    res = c.validate(make_contract(), policy="legal")
    assert res.satisfiable is True
    assert res.candidates[0].endpoint_ref == "e1"


def test_eligible(srv):
    c = Client(f"http://127.0.0.1:{srv}")
    cands = c.eligible(make_contract())
    assert len(cands) == 1 and cands[0].eligible is True


def test_embeddings(srv):
    c = Client(f"http://127.0.0.1:{srv}")
    resp = c.embeddings("auto", ["a", "b"], dimensions=2)
    assert len(resp.vectors) == 2
    assert resp.dimensions == 2
    assert resp.endpoint_ref == "e1"


def test_replay(srv):
    c = Client(f"http://127.0.0.1:{srv}")
    res = c.replay("exp-ep", policy="legal")
    assert res.records == 10
    assert res.verdict == "PASS"
    assert res.savings_percent == 50.0


def test_train_dataset_upload(srv):
    c = Client(f"http://127.0.0.1:{srv}")
    samples = [
        {
            "features": {
                "task": "contract_clause_extraction",
                "domain": "legal",
                "input_tokens_est": 1200,
                "has_session": True,
                "tool_use_required": False,
                "structured_output": False,
                "has_deadline": True,
                "has_latency_p95": True,
            },
            "candidate_ref": "ep-openai-sg/v1",
            "quality": 0.9,
            "tcos": 0.002,
        },
        {
            "features": {
                "task": "code_review",
                "domain": "software",
                "input_tokens_est": 3000,
                "has_session": False,
                "tool_use_required": True,
                "structured_output": False,
                "has_deadline": False,
                "has_latency_p95": False,
            },
            "candidate_ref": "ep-groq-sg/v1",
            "quality": 0.4,
            "tcos": 0.0001,
        },
    ]
    res = c.train("v2", samples=samples)
    assert res["version_id"] == "v2"
    assert res["trained_samples"] == 2
    assert res["min_confidence"] == 0.6
    assert res["artifact_digest"].startswith("sha256:")


def test_unsatisfiable(srv):
    c = Client(f"http://127.0.0.1:{srv}")
    with pytest.raises(UnsatisfiableError) as exc:
        c._request("POST", "/v1/nonexistent")
    assert "region_not_allowed" in exc.value.failed_gates


def test_contract_to_dict_drops_none():
    c = Contract(task="t", modality=["text"])
    d = c.to_dict()
    assert d == {"task": "t", "modality": ["text"]}
    assert "policy" not in d
