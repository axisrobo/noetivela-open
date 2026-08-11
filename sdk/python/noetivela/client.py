"""NOETIVELA gateway client (Apache-2.0, zero dependencies).

Talks to a governed NOETIVELA gateway. Applications submit Inference
Contracts; routing, policy and credential binding are the gateway's job.
"""

from __future__ import annotations

import json
import urllib.error
import urllib.request
from dataclasses import dataclass, field
from typing import Any, Dict, Iterator, List, Optional

from .contract import Contract


class NoetivelaError(Exception):
    """Base class for SDK errors."""


class UnsatisfiableError(NoetivelaError):
    """Contract cannot be satisfied after hard gates (HTTP 422)."""

    def __init__(self, message: str, failed_gates: List[str]):
        super().__init__(message)
        self.failed_gates = failed_gates


class ChatStreamError(NoetivelaError):
    pass


@dataclass
class ChatResponse:
    content: str
    finish_reason: str
    usage: Dict[str, int] = field(default_factory=dict)
    decision_id: str = ""
    model_ref: str = ""
    endpoint_ref: str = ""
    tcos: Dict[str, float] = field(default_factory=dict)


@dataclass
class EligibleCandidate:
    model_ref: str
    endpoint_ref: str
    eligible: bool
    score: float = 0.0
    filter_reasons: List[str] = field(default_factory=list)


@dataclass
class ValidateResult:
    valid: bool
    satisfiable: bool
    candidates: List[EligibleCandidate] = field(default_factory=list)
    policy: str = ""


@dataclass
class EmbeddingResponse:
    vectors: List[List[float]]
    dimensions: int
    usage: Dict[str, int] = field(default_factory=dict)
    endpoint_ref: str = ""


@dataclass
class DecisionSummary:
    decision_id: str
    policy: str = ""
    selection: Optional[Any] = None


@dataclass
class StrategyResult:
    strategy: str
    total_tcos: float = 0.0
    total_quality: float = 0.0
    quality_adjusted_tcos: float = 0.0
    served_count: int = 0
    denied_count: int = 0


@dataclass
class ReplayResult:
    records: int
    savings_percent: float
    verdict: str
    baseline: StrategyResult = field(default_factory=StrategyResult)
    routed: StrategyResult = field(default_factory=StrategyResult)


class Client:
    """Thin, dependency-free client for the NOETIVELA data plane."""

    def __init__(
        self,
        base_url: str,
        contract: Optional[Contract] = None,
        policy: str = "",
        tenant: str = "",
        principal: str = "",
        timeout: float = 120.0,
    ):
        self.base_url = base_url.rstrip("/")
        self.contract = contract
        self.policy = policy
        self.tenant = tenant
        self.principal = principal
        self.timeout = timeout

    def _headers(self, extra: Optional[Dict[str, str]] = None) -> Dict[str, str]:
        h = {"Content-Type": "application/json"}
        if self.contract is not None:
            h["X-Noetivela-Contract"] = json.dumps(self.contract.to_dict())
        if self.policy:
            h["X-Noetivela-Policy"] = self.policy
        if self.tenant:
            h["X-Noetivela-Tenant"] = self.tenant
        if self.principal:
            h["X-Noetivela-Principal"] = self.principal
        if extra:
            h.update(extra)
        return h

    def _request(self, method: str, path: str, payload: Any = None) -> dict:
        req = urllib.request.Request(self.base_url + path, method=method)
        for k, v in self._headers().items():
            req.add_header(k, v)
        data = None
        if payload is not None:
            data = json.dumps(payload).encode("utf-8")
        try:
            with urllib.request.urlopen(req, data=data, timeout=self.timeout) as resp:
                body = resp.read()
                return json.loads(body) if body else {}
        except urllib.error.HTTPError as e:
            body = e.read().decode("utf-8", errors="replace")
            if e.code == 422:
                try:
                    wrapped = json.loads(body)
                    gates = []
                    err = wrapped.get("error", {})
                    for g in err.get("failed_gates", []):
                        gates.append(g.get("reason", str(g)))
                    raise UnsatisfiableError(err.get("message", "unsatisfiable contract"), gates)
                except json.JSONDecodeError:
                    pass
            raise NoetivelaError(f"HTTP {e.code}: {body}") from e

    # ---- governed inference ----

    def chat(self, model: str, messages: List[Dict[str, str]]) -> ChatResponse:
        data = self._request(
            "POST",
            "/v1/chat/completions",
            {"model": model, "messages": messages, "stream": False},
        )
        choices = data.get("choices", [])
        if not choices:
            raise NoetivelaError("empty response")
        noetivela = data.get("noetivela", {})
        return ChatResponse(
            content=choices[0]["message"]["content"],
            finish_reason=choices[0].get("finish_reason", ""),
            usage=data.get("usage", {}),
            decision_id=noetivela.get("decision_id", ""),
            model_ref=noetivela.get("model_ref", ""),
            endpoint_ref=noetivela.get("endpoint_ref", ""),
            tcos=noetivela.get("tcos", {}),
        )

    def chat_stream(self, model: str, messages: List[Dict[str, str]]) -> Iterator[str]:
        """Yield delta content chunks from an SSE stream."""
        req = urllib.request.Request(
            self.base_url + "/v1/chat/completions", method="POST"
        )
        for k, v in self._headers().items():
            req.add_header(k, v)
        body = json.dumps(
            {"model": model, "messages": messages, "stream": True}
        ).encode("utf-8")
        try:
            resp = urllib.request.urlopen(req, data=body, timeout=self.timeout)
        except urllib.error.HTTPError as e:
            raise ChatStreamError(f"HTTP {e.code}") from e
        with resp:
            buffer = ""
            while True:
                chunk = resp.read(1)
                if not chunk:
                    break
                buffer += chunk.decode("utf-8", errors="replace")
                while "\n\n" in buffer:
                    event, _, buffer = buffer.partition("\n\n")
                    for line in event.splitlines():
                        if line.startswith("data:"):
                            payload = line[5:].strip()
                            if payload == "[DONE]":
                                return
                            try:
                                obj = json.loads(payload)
                            except json.JSONDecodeError:
                                continue
                            if obj.get("done"):
                                return
                            for choice in obj.get("choices", []):
                                delta = choice.get("delta", {})
                                content = delta.get("content", "")
                                if content:
                                    yield content

    # ---- explain / governance ----

    def validate(self, contract: Contract, policy: str = "") -> ValidateResult:
        payload = {"contract": contract.to_dict(), "policy": policy or self.policy}
        data = self._request("POST", "/v1/contracts/validate", payload)
        candidates = [
            EligibleCandidate(
                model_ref=c.get("model_ref", ""),
                endpoint_ref=c.get("endpoint_ref", ""),
                eligible=c.get("eligible", False),
                score=c.get("score", 0.0),
                filter_reasons=c.get("filter_reasons", []),
            )
            for c in data.get("candidates", [])
        ]
        return ValidateResult(
            valid=data.get("valid", False),
            satisfiable=data.get("satisfiable", False),
            candidates=candidates,
            policy=data.get("policy", ""),
        )

    def eligible(self, contract: Contract, policy: str = "") -> List[EligibleCandidate]:
        payload = {"contract": contract.to_dict(), "policy": policy or self.policy}
        data = self._request("POST", "/v1/eligible", payload)
        return [
            EligibleCandidate(
                model_ref=c.get("model_ref", ""),
                endpoint_ref=c.get("endpoint_ref", ""),
                eligible=c.get("eligible", False),
                score=c.get("score", 0.0),
                filter_reasons=c.get("filter_reasons", []),
            )
            for c in data.get("ranked", [])
        ]

    def embeddings(self, model: str, input: List[str], dimensions: int = 0) -> EmbeddingResponse:
        data = self._request(
            "POST",
            "/v1/embeddings",
            {"model": model, "input": input, "dimensions": dimensions},
        )
        noetivela = data.get("noetivela", {})
        return EmbeddingResponse(
            vectors=data.get("data", []),
            dimensions=data.get("dimensions", 0),
            usage=data.get("usage", {}),
            endpoint_ref=noetivela.get("endpoint_ref", ""),
        )

    def decisions(self, limit: int = 50, offset: int = 0, task: str = "", model: str = "") -> List[DecisionSummary]:
        q = f"limit={limit}&offset={offset}"
        if task:
            q += f"&task={task}"
        if model:
            q += f"&model={model}"
        data = self._request("GET", f"/v1/decisions?{q}")
        return [
            DecisionSummary(
                decision_id=d.get("decision_id", ""),
                policy=d.get("policy_version", ""),
                selection=d.get("selection"),
            )
            for d in data.get("decisions", [])
        ]

    def get_decision(self, decision_id: str) -> dict:
        return self._request("GET", f"/v1/decisions/{decision_id}")

    def usage(self) -> List[dict]:
        return self._request("GET", "/v1/usage")

    def replay(self, baseline_endpoint: str, policy: str = "", limit: int = 0) -> ReplayResult:
        data = self._request(
            "POST",
            "/v1/replay",
            {"baseline_endpoint": baseline_endpoint, "policy": policy or self.policy, "limit": limit},
        )
        return ReplayResult(
            records=data.get("records", 0),
            savings_percent=data.get("savings_percent", 0.0),
            verdict=data.get("verdict", ""),
            baseline=StrategyResult(**data.get("baseline", {})),
            routed=StrategyResult(**data.get("routed", {})),
        )

    def train(self, version_id: str, min_samples: int = 10, tcos_weight: float = 0.5, hold_out_frac: float = 0.0, samples: Optional[List[dict]] = None) -> dict:
        """Train the learned router. samples = optional offline evidence
        (features + candidate_ref + quality + tcos); when None the gateway's
        online collector is used."""
        payload = {
            "version_id": version_id,
            "min_samples": min_samples,
            "tcos_weight": tcos_weight,
            "hold_out_frac": hold_out_frac,
        }
        if samples:
            payload["samples"] = samples
        return self._request("POST", "/v1/train", payload)
