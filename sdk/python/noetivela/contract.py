"""Inference Contract dataclass mirroring contracts/schema/inference-contract.json.

A Contract declares WHAT intelligence a task needs — never WHICH vendor
endpoint provides it. Policy fields are hard gates.
"""

from dataclasses import dataclass, field, asdict
from typing import List, Optional


@dataclass
class Quality:
    minimum_eval_grade: Optional[str] = None
    grounding_required: bool = False


@dataclass
class Latency:
    ttft_ms: Optional[int] = None
    p95_ms: Optional[int] = None
    deadline_ms: Optional[int] = None
    streaming: bool = False


@dataclass
class Cost:
    ceiling_per_request: Optional[str] = None
    budget_account: Optional[str] = None


@dataclass
class Context:
    session_id: Optional[str] = None
    cache_key: Optional[str] = None
    prefer_cache_locality: bool = True


@dataclass
class Policy:
    data_classification: Optional[str] = None
    allowed_regions: List[str] = field(default_factory=list)
    denied_providers: List[str] = field(default_factory=list)
    provider_training: Optional[str] = None
    retention: Optional[str] = None


@dataclass
class Reliability:
    retry_budget: Optional[int] = None
    fallback: Optional[str] = None


@dataclass
class Contract:
    task: str
    modality: List[str]
    domain: Optional[str] = None
    language: List[str] = field(default_factory=list)
    response_schema: Optional[str] = None
    quality: Optional[Quality] = None
    latency: Optional[Latency] = None
    cost: Optional[Cost] = None
    context: Optional[Context] = None
    policy: Optional[Policy] = None
    reliability: Optional[Reliability] = None

    def to_dict(self) -> dict:
        d = asdict(self)
        # Drop None values for a clean wire payload.
        return _drop_none(d)


def _drop_none(obj):
    if isinstance(obj, dict):
        out = {}
        for k, v in obj.items():
            dropped = _drop_none(v)
            if dropped is not None:
                out[k] = dropped
        return out
    if isinstance(obj, list):
        # Drop empty default lists so the wire payload only carries intent.
        if not obj:
            return None
        return [_drop_none(v) for v in obj]
    return obj
