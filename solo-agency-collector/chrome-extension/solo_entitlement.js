// Solo Agency Free/Pro entitlement — verified by the EXTENSION itself.
//
// The bridge hands every job a signed token (collector_bridge.entitlement_token) it fetched from
// the owner's server with the operator's WideCast key. This file checks that token against the
// server's public key, in this extension, before a Pro capability runs — so a bridge that was
// rewritten or patched to say "pro" gets nothing the stock extension would not give a Free plan.
// Everything here is ordinary WebCrypto (Chrome 137+ ships Ed25519 natively); no library, no
// network, nothing hidden. Loaded by background.js via importScripts and by the Node test.
//
// SOLO_ENTITLEMENT_ENFORCE=true since 2026-09-07 (launch): a paid capability without a verified
// token carrying its feature is skipped and reported as solo_entitlement_required. Set it to false
// to fall back to verify-and-report only.
(function (root) {
  "use strict";

  const SOLO_ENTITLEMENT_PUBLIC_KEY_HEX = "bf2a1f8444e573799c887481c0374adfe54f22ca0ee7fff2d76bac861c91d3a5";
  const SOLO_ENTITLEMENT_ISSUER = "widecast.ai";
  const SOLO_ENTITLEMENT_AUDIENCE = "solo-agency";
  const SOLO_ENTITLEMENT_GRACE_MS = 14 * 24 * 60 * 60 * 1000; // same offline window as the bridge
  const SOLO_UPGRADE_URL = "https://widecast.ai/#setup";
  const SOLO_ENTITLEMENT_ENFORCE = true;
  // The ladder (2026-09-07): free · starter $49 · pro $99 · business $199 · enterprise. Names are
  // informational — grants come from the token's `features`, limits from its `limits`.
  const SOLO_KNOWN_TIERS = new Set(["free", "starter", "pro", "business", "enterprise"]);

  // The extension's OWN map of paid capabilities → the feature the plan must carry. It is never
  // taken from the bridge or the catalog it serves — a homebrew bridge would simply call
  // everything free. Keep in step with `tier: pro` / `feature` in collector_capabilities.json.
  // Plans (2026-09-07): free none · starter enrich+write_actions · pro/business/enterprise all.
  const SOLO_CAPABILITY_FEATURES = {
    "fb.profile.friends": "harvest", "fb.people.search": "harvest",
    "fb.profile.header": "enrich", "fb.profile.hovercard": "enrich", "fb.profile.videos": "enrich",
    "fb.profile.enrich": "enrich", "fb.profile.dossier": "enrich", "fb.profile.contacts": "enrich",
    "fb.post.react": "write_actions", "fb.post.comment": "write_actions",
    "fb.message.send": "write_actions", "fb.group.post": "write_actions",
    "zillow.agents.list": "zillow", "zillow.profile.enrich": "zillow"
  };

  function free(source, reason, extra) {
    return Object.assign({ ok: false, tier: "free", features: [], limits: {}, source, reason, expiresAt: null, companyId: "" }, extra || {});
  }

  function b64urlToBytes(s) {
    const pad = "=".repeat((4 - (s.length % 4)) % 4);
    const b64 = (s + pad).replace(/-/g, "+").replace(/_/g, "/");
    const bin = atob(b64);
    const out = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
    return out;
  }

  function hexToBytes(hex) {
    const out = new Uint8Array(hex.length / 2);
    for (let i = 0; i < out.length; i++) out[i] = parseInt(hex.substr(i * 2, 2), 16);
    return out;
  }

  function utf8(s) {
    return new TextEncoder().encode(s);
  }

  // verify(token, nowMs, opts) → { ok, tier, features, limits, source, reason, expiresAt, companyId }
  //   source: none | unverifiable | invalid | token | grace | expired
  //   ok is true only when the signature verified; tier is what the caller may act on.
  async function verify(token, nowMs, opts) {
    const now = typeof nowMs === "number" ? nowMs : Date.now();
    const publicKeyHex = (opts && opts.publicKeyHex) || SOLO_ENTITLEMENT_PUBLIC_KEY_HEX;
    const subtle = (opts && opts.subtle) || (root.crypto && root.crypto.subtle) || (globalThis.crypto && globalThis.crypto.subtle);
    const raw = String(token || "").trim();
    if (!raw) return free("none", "no_token");
    const parts = raw.split(".");
    if (parts.length !== 3 || !parts[0] || !parts[1] || !parts[2]) return free("invalid", "malformed_token");
    let header, claims;
    try {
      header = JSON.parse(new TextDecoder().decode(b64urlToBytes(parts[0])));
      claims = JSON.parse(new TextDecoder().decode(b64urlToBytes(parts[1])));
    } catch (e) {
      return free("invalid", "undecodable_token");
    }
    if (!header || header.alg !== "EdDSA") return free("invalid", "unsupported_alg");
    if (!subtle) return free("unverifiable", "webcrypto_unavailable");
    let verified = false;
    try {
      const key = await subtle.importKey("raw", hexToBytes(publicKeyHex), { name: "Ed25519" }, false, ["verify"]);
      verified = await subtle.verify({ name: "Ed25519" }, key, b64urlToBytes(parts[2]), utf8(parts[0] + "." + parts[1]));
    } catch (e) {
      return free("unverifiable", "ed25519_unsupported: " + String(e && e.message ? e.message : e));
    }
    if (!verified) return free("invalid", "bad_signature");
    if (claims.aud !== SOLO_ENTITLEMENT_AUDIENCE || claims.iss !== SOLO_ENTITLEMENT_ISSUER) return free("invalid", "wrong_audience_or_issuer");
    const expMs = Number(claims.exp || 0) * 1000;
    if (!expMs) return free("invalid", "no_expiry");
    const tierName = String(claims.tier || "free").toLowerCase();
    const tier = SOLO_KNOWN_TIERS.has(tierName) ? tierName : "free"; // an unknown tier is never more than free
    const base = {
      ok: true, tier, features: Array.isArray(claims.features) ? claims.features.slice() : [],
      limits: claims.limits && typeof claims.limits === "object" ? claims.limits : {},
      expiresAt: new Date(expMs).toISOString(), companyId: String(claims.sub || ""), reason: ""
    };
    if (now <= expMs) return Object.assign(base, { source: "token" });
    if (now < expMs + SOLO_ENTITLEMENT_GRACE_MS) return Object.assign(base, { source: "grace", reason: "expired_within_grace" });
    return free("expired", "expired_past_grace", { expiresAt: base.expiresAt, companyId: base.companyId });
  }

  // featureFor: "" for capabilities every plan may run, else the feature name the token must list.
  function featureFor(capabilityId) {
    return SOLO_CAPABILITY_FEATURES[String(capabilityId || "")] || "";
  }

  // granted: a free capability is always granted; a paid one needs a VERIFIED token whose
  // `features` carries the capability's feature (the tier name is never consulted).
  function granted(ent, capabilityId) {
    const feature = featureFor(capabilityId);
    if (!feature) return true;
    return !!(ent && ent.ok && Array.isArray(ent.features) && ent.features.includes(feature));
  }

  // A small view for state/popup/source_status rows — never the token itself.
  function view(ent) {
    if (!ent) return { tier: "free", source: "none", verified: false };
    return { tier: ent.tier, source: ent.source, verified: !!ent.ok, reason: ent.reason || "", expires_at: ent.expiresAt || null, enforce: SOLO_ENTITLEMENT_ENFORCE };
  }

  root.SoloEntitlement = {
    verify, featureFor, granted, view,
    ENFORCE: SOLO_ENTITLEMENT_ENFORCE,
    UPGRADE_URL: SOLO_UPGRADE_URL,
    PUBLIC_KEY_HEX: SOLO_ENTITLEMENT_PUBLIC_KEY_HEX,
    CAPABILITY_FEATURES: SOLO_CAPABILITY_FEATURES
  };
})(typeof self !== "undefined" ? self : globalThis);
