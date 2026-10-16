/*
 * Test PPN 11% dan total (aturan 5 dan 6).
 *
 * Aturan:
 * - PPN = 11% x (subtotal - diskon).
 * - Dibulatkan ke rupiah terdekat, pecahan tepat 0,5 dibulatkan naik.
 * - Total = subtotal - diskon + PPN.
 */

const test = require("node:test");
const assert = require("node:assert/strict");
const kasir = require("../kasir.js");

// --- Bug: pembulatan PPN ---

test("PPN pecahan di bawah 0,5 dibulatkan turun (940 -> 103,4 -> 103)", () => {
  assert.equal(kasir.hitungPPN(940), 103);
});

test("PPN pecahan tepat 0,5 dibulatkan naik (950 -> 104,5 -> 105)", () => {
  assert.equal(kasir.hitungPPN(950), 105);
});

test("PPN pecahan di atas 0,5 dibulatkan naik (960 -> 105,6 -> 106)", () => {
  assert.equal(kasir.hitungPPN(960), 106);
});

test("PPN bilangan bulat tetap sama (1000 -> 110)", () => {
  assert.equal(kasir.hitungPPN(1000), 110);
});

// --- Total ---

test("ringkasan non-member tepat 100000 (diskon 5000, PPN 10450, total 105450)", () => {
  const keranjang = [];
  kasir.tambahItem(keranjang, "K04", 25); // 25 x 4000 = 100000
  const r = kasir.hitungRingkasan(keranjang, false);
  assert.equal(r.subtotal, 100000);
  assert.equal(r.diskon, 5000);
  assert.equal(r.ppn, 10450);
  assert.equal(r.total, 105450);
});

test("ringkasan member 100000 (diskon 10000, PPN 9900, total 99900)", () => {
  const keranjang = [];
  kasir.tambahItem(keranjang, "K04", 25); // 100000
  const r = kasir.hitungRingkasan(keranjang, true);
  assert.equal(r.subtotal, 100000);
  assert.equal(r.diskon, 10000);
  assert.equal(r.ppn, 9900);
  assert.equal(r.total, 99900);
});

test("ringkasan total = subtotal - diskon + PPN", () => {
  const keranjang = [];
  kasir.tambahItem(keranjang, "K01", 2); // 30000
  kasir.tambahItem(keranjang, "K04", 3); // 12000
  const r = kasir.hitungRingkasan(keranjang, false);
  assert.equal(r.subtotal, 42000);
  assert.equal(r.diskon, 0);
  assert.equal(r.total, r.subtotal - r.diskon + r.ppn);
});
