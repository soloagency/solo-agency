// Solo Agency canonical record schema (platform-module refactor, step 2).
//
// This is the SHARED shape every platform module (Facebook today, Zillow, and whatever
// comes after) normalizes its typed capability output INTO, before it reaches the CRM.
// One profile/post/comment/group/message contract, independent of which site produced it.
//
// FIELD-NAMING RULE — read this before adding a field. bridge-go/main.go's isSensitiveKey
// (mirrored below as SENSITIVE_KEY_NEEDLES / isSensitiveKey) redacts any field whose NAME
// contains one of a fixed list of needles ("auth", "token", "secret", "session", ...) as a
// case-insensitive SUBSTRING match — it does not know or care what the field actually holds.
// That is why:
//   - the posting/commenting entity is named `actor`, never `author` — "author" contains
//     "auth" and would come back "[redacted]" on every single record.
//   - Facebook's per-story comment-ordering token is named `comment_intent`, never
//     `intent_token` (or anything ending in "_token") — it has to survive redaction because
//     it is the value fb.post.comments needs to fetch the WHOLE thread instead of a
//     filtered subset, and a redacted token is indistinguishable from a missing one.
// The same trap applies to anything you add under `ext` or `refs`: a field called
// `session_id`, `csrf`, `auth_state`, or similar gets silently wiped before a human ever
// sees it. validateRecord()/findSensitiveKeys() catch this at any depth, arrays included,
// so a bad field name fails fast in a unit test instead of silently in production.
//
// Loaded by background.js via importScripts (classic MV3 service worker, no ES modules) and
// by Node tests via vm.runInContext against a fake self/window — see tests/test_schema.js.
(function (root) {
  "use strict";

  const SCHEMA_VERSION = 1;

  // ---------------------------------------------------------------- sensitive keys
  // Verbatim copy of bridge-go/main.go's isSensitiveKey needle list (main.go ~line 2952).
  // Case-insensitive SUBSTRING match, same as the Go implementation — keep this list and
  // the Go one in lockstep; tests/test_platform_registry.js cross-checks them by reading
  // bridge-go/main.go's source text.
  const SENSITIVE_KEY_NEEDLES = [
    "cookie", "token", "secret", "password", "passwd", "pwd", "otp",
    "authorization", "auth", "session", "bearer", "csrf", "xsrf"
  ];

  function isSensitiveKey(name) {
    const k = String(name === null || name === undefined ? "" : name).toLowerCase();
    for (let i = 0; i < SENSITIVE_KEY_NEEDLES.length; i++) {
      if (k.indexOf(SENSITIVE_KEY_NEEDLES[i]) !== -1) return true;
    }
    return false;
  }

  // findSensitiveKeys: walks an object/array at ANY depth and returns the dotted (and
  // bracketed, for array indexes) paths of every KEY that matches isSensitiveKey. The value
  // is never inspected — only the key name, exactly like the bridge.
  function findSensitiveKeys(obj) {
    const hits = [];
    function walk(node, path) {
      if (node === null || typeof node !== "object") return;
      if (Array.isArray(node)) {
        for (let i = 0; i < node.length; i++) walk(node[i], path + "[" + i + "]");
        return;
      }
      const keys = Object.keys(node);
      for (let i = 0; i < keys.length; i++) {
        const key = keys[i];
        const nextPath = path ? path + "." + key : key;
        if (isSensitiveKey(key)) hits.push(nextPath);
        walk(node[key], nextPath);
      }
    }
    walk(obj, "");
    return hits;
  }

  // ---------------------------------------------------------------- kinds / platforms
  const KINDS = ["profile", "post", "comment", "group", "message"];
  const KNOWN_PLATFORMS = ["facebook", "zillow", "instagram", "linkedin", "x", "youtube", "tiktok", "web"];
  // A platform id must look like a token (lowercase, starts with a letter). An id outside
  // KNOWN_PLATFORMS is only a WARNING — a new platform module can ship before this list is
  // updated — but a malformed one (spaces, punctuation, empty) is an ERROR: nothing
  // downstream can key off it safely.
  const PLATFORM_PATTERN = /^[a-z][a-z0-9_]*$/;

  function isPlainObject(v) {
    return !!v && typeof v === "object" && !Array.isArray(v);
  }

  // ---------------------------------------------------------------- entity contract (v1)
  // required: present, non-null, correct primitive type — enforced by validateRecord.
  // optional: type-checked ONLY when present (null is treated as "absent").
  // identity: the fields a later dedupe/merge step would key on for this kind. Descriptive
  // only; validateRecord does not enforce it (there is nothing to enforce about a record
  // being self-consistent with its own identity fields).
  const ENTITIES = {
    profile: {
      required: ["kind", "platform", "name", "captured_at"],
      optional: [
        "url", "platform_id", "handle", "bio", "category", "industry", "location",
        "work", "education", "emails", "phones", "websites", "socials", "photo_url",
        "verified", "follower_count", "about", "extraction_audit", "source_capability",
        "refs", "ext"
      ],
      identity: ["platform", "platform_id", "url"]
    },
    post: {
      required: ["kind", "platform", "platform_id", "captured_at"],
      optional: [
        "url", "text", "actor", "created_at", "platform_time", "engagement",
        "attachments", "group_ref", "media", "source_capability", "refs", "ext"
      ],
      identity: ["platform", "platform_id"]
    },
    comment: {
      required: ["kind", "platform", "platform_id", "post_ref", "text", "captured_at"],
      optional: [
        "url", "actor", "created_at", "platform_time", "depth", "reply_count",
        "replies", "own_ref", "pagination", "source_capability", "refs", "ext"
      ],
      identity: ["platform", "platform_id"]
    },
    group: {
      required: ["kind", "platform", "platform_id", "name", "url", "captured_at"],
      optional: ["type", "member_count", "source_capability", "refs", "ext"],
      identity: ["platform", "platform_id"]
    },
    message: {
      required: ["kind", "platform", "direction", "channel", "body_text", "status", "captured_at"],
      optional: [
        "thread_ref", "post_ref", "group_ref", "verified", "contact_ref",
        "source_capability", "refs", "ext"
      ],
      identity: ["platform", "channel", "thread_ref", "captured_at"]
    }
  };

  // The "at least one of X, Y must be a non-empty string" rule — a constraint ACROSS two
  // fields, so it cannot live in `required`/`optional` (either alone would be too strict).
  const AT_LEAST_ONE = {
    profile: ["platform_id", "url"],
    post: ["text", "url"]
  };

  // Per-field primitive type, shared across every kind (field names do not collide in
  // meaning between kinds). "any" = accepted but not type-checked (raw/opaque values like
  // platform_time). "enum:a,b,c" = must be a string equal to one of the listed values.
  const FIELD_TYPES = {
    kind: "string",
    platform: "string",
    captured_at: "string",
    source_capability: "string",
    refs: "object",
    ext: "object",

    name: "string",
    url: "string",
    platform_id: "string",
    handle: "string",
    bio: "string",
    category: "string",
    industry: "string",
    location: "array",
    work: "array",
    education: "array",
    emails: "array",
    phones: "array",
    websites: "array",
    socials: "object",
    photo_url: "string",
    verified: "boolean",
    follower_count: "number",
    about: "object",
    extraction_audit: "object",

    text: "string",
    actor: "object",
    created_at: "string",
    platform_time: "any",
    engagement: "object",
    attachments: "array",
    group_ref: "object",
    media: "object",

    // Opaque platform handles are STRINGS: for a Facebook comment post_ref is the parent
    // post's feedback id (the only key fb.post.comments accepts), for a write action it is the
    // post url or feedback id targeted; own_ref is the comment's own reply-addressing handle;
    // contact_ref is the CRM contact id once resolved.
    post_ref: "string",
    depth: "number",
    reply_count: "number",
    replies: "array",
    own_ref: "string",
    pagination: "object",

    type: "enum:group,page,channel,community",
    member_count: "number",

    direction: "enum:in,out",
    channel: "enum:dm,comment,post,email,reaction",
    body_text: "string",
    status: "string",
    thread_ref: "object",
    contact_ref: "string"
  };

  function typeOk(type, value) {
    if (!type || type === "any") return true;
    if (type.indexOf("enum:") === 0) {
      const allowed = type.slice(5).split(",");
      return typeof value === "string" && allowed.indexOf(value) !== -1;
    }
    switch (type) {
      case "string": return typeof value === "string";
      case "boolean": return typeof value === "boolean";
      case "number": return typeof value === "number" && isFinite(value);
      case "array": return Array.isArray(value);
      case "object": return isPlainObject(value);
      default: return true;
    }
  }

  // ---------------------------------------------------------------- time helpers
  // toIso: accepts a unix-seconds number, a unix-ms number, or an already-parseable string
  // (ISO or otherwise Date-parseable). >= 1e11 is treated as milliseconds — a seconds value
  // that large would be the year 5138, so the threshold never misreads a real epoch-seconds
  // timestamp. Junk (NaN, unparseable, null/undefined/"") returns null rather than throwing
  // or emitting "Invalid Date".
  function toIso(value) {
    if (value === null || value === undefined || value === "") return null;
    if (typeof value === "number") {
      if (!isFinite(value)) return null;
      const ms = Math.abs(value) >= 1e11 ? value : value * 1000;
      const d = new Date(ms);
      return isFinite(d.getTime()) ? d.toISOString() : null;
    }
    if (typeof value === "string") {
      const trimmed = value.trim();
      if (!trimmed) return null;
      if (/^-?\d+(\.\d+)?$/.test(trimmed)) return toIso(Number(trimmed));
      const d = new Date(trimmed);
      return isFinite(d.getTime()) ? d.toISOString() : null;
    }
    return null;
  }

  function nowIso() {
    return new Date().toISOString();
  }

  const ISO_DATETIME_RE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$/;
  function isIsoDateTime(s) {
    if (typeof s !== "string" || !ISO_DATETIME_RE.test(s)) return false;
    const t = Date.parse(s);
    return isFinite(t);
  }

  // ---------------------------------------------------------------- validation
  // validateRecord: kind must be valid; required fields present with the right primitive
  // type; the kind's at-least-one rule (if any) satisfied; platform matches PLATFORM_PATTERN
  // (error) and is a KNOWN_PLATFORMS member (warning otherwise); captured_at parses as
  // ISO8601; enum fields enforced; ext/refs must be plain objects when present; no sensitive
  // key anywhere in the record (error, one per hit); unrecognised top-level keys (warning).
  function validateRecord(rec) {
    const errors = [];
    const warnings = [];

    if (!isPlainObject(rec)) {
      return { ok: false, errors: ["record must be a plain object"], warnings: [] };
    }

    const kind = rec.kind;
    if (typeof kind !== "string" || KINDS.indexOf(kind) === -1) {
      errors.push("kind must be one of " + KINDS.join(", ") + " (got " + JSON.stringify(kind) + ")");
      findSensitiveKeys(rec).forEach(function (p) {
        errors.push("sensitive key found at '" + p + "'");
      });
      return { ok: false, errors: errors, warnings: warnings };
    }

    const spec = ENTITIES[kind];
    const known = {};
    spec.required.concat(spec.optional).forEach(function (f) { known[f] = true; });

    spec.required.forEach(function (field) {
      const v = rec[field];
      if (v === undefined || v === null) {
        errors.push("missing required field '" + field + "'");
        return;
      }
      const type = FIELD_TYPES[field] || "any";
      if (!typeOk(type, v)) errors.push("field '" + field + "' has the wrong type (expected " + type + ")");
    });

    spec.optional.forEach(function (field) {
      if (!(field in rec)) return;
      const v = rec[field];
      if (v === undefined || v === null) return; // null/undefined treated as absent
      const type = FIELD_TYPES[field] || "any";
      if (!typeOk(type, v)) errors.push("field '" + field + "' has the wrong type (expected " + type + ")");
    });

    const pair = AT_LEAST_ONE[kind];
    if (pair) {
      const ok = pair.some(function (f) { return typeof rec[f] === "string" && rec[f].trim() !== ""; });
      if (!ok) errors.push("at least one of " + pair.join(", ") + " must be a non-empty string");
    }

    if (typeof rec.platform === "string") {
      if (!PLATFORM_PATTERN.test(rec.platform)) {
        errors.push("platform " + JSON.stringify(rec.platform) + " does not match " + PLATFORM_PATTERN);
      } else if (KNOWN_PLATFORMS.indexOf(rec.platform) === -1) {
        warnings.push("unknown platform '" + rec.platform + "'");
      }
    }

    if (typeof rec.captured_at === "string" && !isIsoDateTime(rec.captured_at)) {
      errors.push("captured_at is not a parseable ISO8601 timestamp: " + JSON.stringify(rec.captured_at));
    }

    findSensitiveKeys(rec).forEach(function (p) {
      errors.push("sensitive key found at '" + p + "'");
    });

    Object.keys(rec).forEach(function (k) {
      if (!known[k]) warnings.push("unknown field '" + k + "'");
    });

    return { ok: errors.length === 0, errors: errors, warnings: warnings };
  }

  // validateCanonical: the additive canonical block {schema_version, kind, items[]} a
  // platform module hands back alongside its existing (untouched) output. Every item is
  // validated with validateRecord; item-level errors/warnings are prefixed with their index.
  function validateCanonical(canonical) {
    const errors = [];
    const warnings = [];

    if (!isPlainObject(canonical)) {
      return { ok: false, errors: ["canonical must be a plain object"], warnings: [], item_count: 0 };
    }
    if (canonical.schema_version !== SCHEMA_VERSION) {
      warnings.push(
        "schema_version " + JSON.stringify(canonical.schema_version) +
        " does not match the current SCHEMA_VERSION " + SCHEMA_VERSION
      );
    }
    if (typeof canonical.kind !== "string" || KINDS.indexOf(canonical.kind) === -1) {
      errors.push("canonical.kind must be one of " + KINDS.join(", ") + " (got " + JSON.stringify(canonical.kind) + ")");
    }
    if (!Array.isArray(canonical.items)) {
      errors.push("canonical.items must be an array");
      return { ok: false, errors: errors, warnings: warnings, item_count: 0 };
    }

    canonical.items.forEach(function (item, i) {
      const r = validateRecord(item);
      r.errors.forEach(function (e) { errors.push("items[" + i + "]: " + e); });
      r.warnings.forEach(function (w) { warnings.push("items[" + i + "]: " + w); });
    });

    return { ok: errors.length === 0, errors: errors, warnings: warnings, item_count: canonical.items.length };
  }

  root.SoloSchema = {
    SCHEMA_VERSION: SCHEMA_VERSION,
    SENSITIVE_KEY_NEEDLES: SENSITIVE_KEY_NEEDLES,
    isSensitiveKey: isSensitiveKey,
    findSensitiveKeys: findSensitiveKeys,
    KINDS: KINDS,
    KNOWN_PLATFORMS: KNOWN_PLATFORMS,
    PLATFORM_PATTERN: PLATFORM_PATTERN,
    ENTITIES: ENTITIES,
    AT_LEAST_ONE: AT_LEAST_ONE,
    FIELD_TYPES: FIELD_TYPES,
    validateRecord: validateRecord,
    validateCanonical: validateCanonical,
    toIso: toIso,
    nowIso: nowIso
  };
})(typeof self !== "undefined" ? self : globalThis);
