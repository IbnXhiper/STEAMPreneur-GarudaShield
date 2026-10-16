/*
 * Test aturan diskon (aturan 4).
 *
 * Aturan:
 * - Member: diskon 10% dari subtotal.
 * - Non-member: diskon 5% hanya jika subtotal >= 100000.
 * - Diskon tidak boleh digabung.
 * - Dibulatkan ke rupiah terdekat.
 */

const test = require("node:test");
const assert = require("node:assert/strict");
const kasir = require("../kasir.js");

test("non-member di bawah 100000 tidak dapat diskon", () => {
  assert.equal(kasir.hitungDiskon(99500, false), 0);
});

test("non-member tepat 100000 dapat diskon 5% (5000)", () => {
  assert.equal(kasir.hitungDiskon(100000, false), 5000);
});

test("non-member di atas 100000 dapat diskon 5% (105000 -> 5250)", () => {
  assert.equal(kasir.hitungDiskon(105000, false), 5250);
});

test("member dapat diskon 10% dari subtotal", () => {
  assert.equal(kasir.hitungDiskon(40000, true), 4000);
});

test("member tepat 100000 hanya dapat 10% (10000), tidak digabung 15%", () => {
  assert.equal(kasir.hitungDiskon(100000, true), 10000);
});

test("member di atas 100000 hanya dapat 10%, tidak digabung dengan 5%", () => {
  // Salah: 5% + 10% = 15% -> 22650. Benar: 10% -> 15100.
  assert.equal(kasir.hitungDiskon(151000, true), 15100);
});

test("disknon dibulatkan ke rupiah terdekat (151000 non-member -> 7550)", () => {
  assert.equal(kasir.hitungDiskon(151000, false), 7550);
});

test("pembulatan diskon ke atas pada pecahan 0,5 (100050 non-member -> 5002,5 -> 5003)", () => {
  assert.equal(kasir.hitungDiskon(100050, false), 5003);
});
