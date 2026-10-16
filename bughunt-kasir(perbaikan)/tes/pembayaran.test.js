/*
 * Test pembayaran dan kembalian (aturan 7).
 *
 * Aturan:
 * - uang < total -> transaksi ditolak dengan pesan persis "Uang pembayaran kurang".
 * - uang >= total -> kembalian = uang - total.
 */

const test = require("node:test");
const assert = require("node:assert/strict");
const kasir = require("../kasir.js");

const TOTAL = 10500;

test("pembayaran kurang Rp1 ditolak dengan pesan persis", () => {
  assert.throws(
    () => kasir.hitungKembalian(TOTAL, TOTAL - 1),
    (err) => err instanceof Error && err.message === "Uang pembayaran kurang"
  );
});

test("pembayaran jauh di bawah total ditolak", () => {
  assert.throws(
    () => kasir.hitungKembalian(TOTAL, 5000),
    (err) => err.message === "Uang pembayaran kurang"
  );
});

test("pembayaran nol ditolak (uang dibayar kosong -> 0)", () => {
  assert.throws(
    () => kasir.hitungKembalian(TOTAL, Number("")),
    (err) => err.message === "Uang pembayaran kurang"
  );
});

test("pembayaran tepat sama menghasilkan kembalian 0", () => {
  assert.equal(kasir.hitungKembalian(TOTAL, TOTAL), 0);
});

test("pembayaran lebih menghasilkan kembalian yang benar", () => {
  assert.equal(kasir.hitungKembalian(TOTAL, 20000), 9500);
});

test("transaksi yang ditolak tidak menghasilkan kembalian menyesatkan", () => {
  let kembalian;
  let ditolak = false;
  try {
    kembalian = kasir.hitungKembalian(TOTAL, TOTAL - 1);
  } catch (e) {
    ditolak = true;
  }
  assert.equal(ditolak, true);
  assert.equal(kembalian, undefined);
});
