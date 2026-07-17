/*
 * contact_extract.js
 * Structured business email + phone extraction for the Solo Agency Local Collector.
 *
 * ADDITIVE layer: this runs ALONGSIDE the existing capture pipeline and never
 * changes the captured/filtered text or any existing field. Given the already
 * cleaned visible page text (produced downstream of filtering.js) plus the live
 * DOM (for mailto:/tel: anchors), it returns:
 *
 *   { emails: [...unique, lowercased...], phones: [...unique, E.164-ish...] }
 *
 * attached as an optional `contacts` field on the collected data point.
 *
 * It is injected into the capture isolated world via CAPTURE_FILES (see
 * background.js) and exposes window.__soloExtractContacts(text, doc). The pure
 * text helpers are also exported under module.exports for Node self-tests.
 *
 * Scope note: it only structures what already appears in the public page content
 * the collector captured (visible text + link hrefs). It does not open, expand,
 * or navigate to any hidden "contact info" section, and it does not change the
 * existing getHumanReadableText() inline `Email:`/`Phone:` behavior.
 */

(function () {
  "use strict";

  // Cap output so a link-heavy page cannot produce a pathological payload.
  var MAX_CONTACTS = 50;

  // RFC-ish email pattern (deliberately permissive; false positives pruned below).
  var EMAIL_RE = /[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}/g;

  // Asset/file extensions that mean a match is really a filename, not an address.
  var IMAGE_EXT_RE = /\.(png|jpe?g|gif|svg|webp|bmp|ico|tiff?|mp4|webm|mp3|css|js|json|woff2?|ttf|eot)$/;

  // Obvious placeholder / non-deliverable domains.
  var PLACEHOLDER_EMAIL_DOMAIN_RE = /(^|\.)(example|test|domain|yourdomain|mydomain|email|sentry)\.(com|org|net|io|dev|local)$/;

  function isJunkEmail(email) {
    if (!email) return true;
    var at = email.lastIndexOf("@");
    if (at <= 0 || at === email.length - 1) return true;
    var local = email.slice(0, at);
    var domain = email.slice(at + 1);
    // Retina asset sprites, e.g. logo@2x.png / icon@3x.
    if (/@\d+x(\b|\.)/.test(email)) return true;
    if (domain.length < 3 || domain.indexOf(".") === -1) return true;
    if (local.charAt(0) === "." || local.charAt(local.length - 1) === ".") return true;
    if (domain.indexOf("..") !== -1) return true;
    if (IMAGE_EXT_RE.test(domain)) return true; // image/asset "addresses"
    if (domain.indexOf("sentry") !== -1) return true; // sentry DSNs / ingest keys
    if (PLACEHOLDER_EMAIL_DOMAIN_RE.test(domain)) return true; // example.com, etc.
    return false;
  }

  function extractEmailsFromText(text) {
    var out = [];
    if (!text) return out;
    var matches = String(text).match(EMAIL_RE);
    if (!matches) return out;
    for (var i = 0; i < matches.length; i++) {
      var email = matches[i].toLowerCase().replace(/[.,;:]+$/, "");
      if (!isJunkEmail(email)) out.push(email);
    }
    return out;
  }

  // Phone candidates: optional leading +, a digit, then 7+ phone-ish chars
  // (digits, spaces, common separators, unicode dashes), ending in a digit.
  var PHONE_CANDIDATE_RE = /\+?\d[\d\s().\-‐-―]{7,}\d/g;

  function normalizePhone(raw) {
    if (!raw) return "";
    var trimmed = String(raw).trim();
    var hasPlus = trimmed.charAt(0) === "+";
    var digits = trimmed.replace(/\D/g, "");
    if (digits.length < 10 || digits.length > 15) return ""; // spec: keep 10..15 digits
    if (/^(\d)\1+$/.test(digits)) return ""; // all-same-digit junk (e.g. 0000000000)
    if (hasPlus) return "+" + digits; // country code explicitly present
    // Bare NANP number carrying the country code 1 (11 digits starting with 1).
    if (digits.length === 11 && digits.charAt(0) === "1") return "+" + digits;
    return digits; // no explicit country code: normalized digit string
  }

  function extractPhonesFromText(text) {
    var out = [];
    if (!text) return out;
    var matches = String(text).match(PHONE_CANDIDATE_RE);
    if (!matches) return out;
    for (var i = 0; i < matches.length; i++) {
      var norm = normalizePhone(matches[i]);
      if (norm) out.push(norm);
    }
    return out;
  }

  function decodeMaybe(s) {
    try {
      return decodeURIComponent(s);
    } catch (e) {
      return String(s || "");
    }
  }

  // Harvest mailto:/tel: hrefs straight from the DOM: their address/number often
  // lives only in the href (not the visible text), so a text-only pass misses them.
  function harvestAnchorContacts(doc) {
    var emails = [];
    var phones = [];
    if (!doc || typeof doc.querySelectorAll !== "function") {
      return { emails: emails, phones: phones };
    }
    var anchors;
    try {
      anchors = doc.querySelectorAll("a[href]");
    } catch (e) {
      return { emails: emails, phones: phones };
    }
    if (!anchors) return { emails: emails, phones: phones };
    var list = Array.prototype.slice.call(anchors);
    for (var i = 0; i < list.length; i++) {
      var a = list[i];
      var href = a && typeof a.getAttribute === "function" ? String(a.getAttribute("href") || "") : "";
      if (!href) continue;
      var lower = href.toLowerCase();
      if (lower.indexOf("mailto:") === 0) {
        var addrPart = href.slice(7).split("?")[0];
        var parts = addrPart.split(",");
        for (var j = 0; j < parts.length; j++) {
          var addr = decodeMaybe(parts[j]).trim().toLowerCase();
          if (addr) emails.push(addr);
        }
      } else if (lower.indexOf("tel:") === 0) {
        var num = decodeMaybe(href.slice(4).split("?")[0]).trim();
        if (num) phones.push(num);
      }
    }
    return { emails: emails, phones: phones };
  }

  function uniqueSortedCapped(values) {
    var seen = Object.create(null);
    var out = [];
    for (var i = 0; i < values.length; i++) {
      var v = values[i];
      if (!v || seen[v]) continue;
      seen[v] = true;
      out.push(v);
    }
    out.sort();
    if (out.length > MAX_CONTACTS) out = out.slice(0, MAX_CONTACTS);
    return out;
  }

  // Main entry: cleaned visible page text + optional live DOM document.
  function extractContacts(text, doc) {
    var emails = extractEmailsFromText(text);
    var phones = extractPhonesFromText(text);
    if (doc) {
      var anchor = harvestAnchorContacts(doc);
      for (var i = 0; i < anchor.emails.length; i++) {
        if (!isJunkEmail(anchor.emails[i])) emails.push(anchor.emails[i]);
      }
      for (var k = 0; k < anchor.phones.length; k++) {
        var p = normalizePhone(anchor.phones[k]);
        if (p) phones.push(p);
      }
    }
    return { emails: uniqueSortedCapped(emails), phones: uniqueSortedCapped(phones) };
  }

  var api = {
    extractContacts: extractContacts,
    extractEmailsFromText: extractEmailsFromText,
    extractPhonesFromText: extractPhonesFromText,
    harvestAnchorContacts: harvestAnchorContacts,
    normalizePhone: normalizePhone,
    isJunkEmail: isJunkEmail
  };

  if (typeof window !== "undefined") {
    window.__soloExtractContacts = extractContacts;
    window.__soloContactExtract = api;
  }
  if (typeof module !== "undefined" && module.exports) {
    module.exports = api;
  }
})();
