# TARA Package

This directory contains the P6 threat analysis and risk controls:

| File | Purpose |
|---|---|
| `assets.md` | Protected assets and trust boundaries |
| `threats.md` | STRIDE table across PHI, secrets, audit, evidence, MQTT, and OCR |
| `mitigations.md` | Control mapping from threat to implementation |
| `attack_trees/mqtt_mitm.md` | MQTT man-in-the-middle attack tree |
| `attack_trees/ocr_worker_rce.md` | Crafted-image OCR worker attack tree |
| `attack_trees/evidence_tampering.md` | Evidence-chain tampering attack tree |

The implementation controls are in `server/crypto`, `server/audit`, `server/evidence`, `server/reports`, `server/repository/bootstrap.go`, and the mTLS Docker/Mosquitto configuration.
