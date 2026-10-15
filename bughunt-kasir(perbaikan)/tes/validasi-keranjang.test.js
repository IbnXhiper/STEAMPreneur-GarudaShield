/*
 * Test validasi jumlah dan penggabungan produk di keranjang.
 * Mengacu pada aturan bisnis 1 dan 2 (README).
 *
 * Perilaku yang diharapkan:
 * - Jumlah harus bilangan bulat 1..99.
 * - Jumlah kosong/0/negatif/pecahan/NaN/tak hingga ditolak (throw).
 * - Produk sama yang ditambah lagi: jumlahnya dijumlahkan.
 * - Total per produk tidak boleh lebih dari 99.
 * - Operasi yang ditolak tidak boleh mengubah keranjang.
 */

const test = require("node:test");
const assert = require("node:assert/strict");
const kasir = require("../kasir.js");

// --- Bug: validasi jumlah (aturan 1) ---

test("jumlah 0 ditolak", () => {
  assert.throws(() => kasir.tambahItem([], "K04", 0));
});

test("jumlah negatif ditolak", () => {
  assert.throws(() => kasir.tambahItem([], "K04", -1));
});

test("jumlah pecahan ditolak", () => {
  assert.throws(() => kasir.tambahItem([], "K04", 1.5));
});

test("jumlah NaN ditolak", () => {
  assert.throws(() => kasir.tambahItem([], "K04", NaN));
});

test("jumlah Infinity ditolak", () => {
  assert.throws(() => kasir.tambahItem([], "K04", Infinity));
});

test("jumlah string kosong ditolak (tidak diam-diam jadi 0)", () => {
  assert.throws(() => kasir.tambahItem([], "K04", ""));
});

test("jumlah string spasi ditolak (tidak diam-diam jadi 0)", () => {
  assert.throws(() => kasir.tambahItem([], "K04", "   "));
});

test("jumlah string 'abc' ditolak", () => {
  assert.throws(() => kasir.tambahItem([], "K04", "abc"));
});

test("jumlah 100 ditolak (di atas batas 99)", () => {
  assert.throws(() => kasir.tambahItem([], "K04", 100));
});

test("jumlah 1 diterima", () => {
  const keranjang = [];
  kasir.tambahItem(keranjang, "K04", 1);
  assert.equal(keranjang.length, 1);
  assert.equal(keranjang[0].qty, 1);
});

test("jumlah 99 diterima dan tersimpan sebagai angka", () => {
  const keranjang = [];
  kasir.tambahItem(keranjang, "K04", 99);
  assert.equal(keranjang[0].qty, 99);
  assert.equal(typeof keranjang[0].qty, "number");
});

test("jumlah berupa string angka '2' diterima dan dikonversi ke angka", () => {
  const keranjang = [];
  kasir.tambahItem(keranjang, "K04", "2");
  assert.equal(keranjang[0].qty, 2);
  assert.equal(typeof keranjang[0].qty, "number");
});

// --- Bug: penggabungan produk sama (aturan 2) ---

test("tambah produk sama menjumlahkan qty: Es Teh 2 + 3 = 5", () => {
  const keranjang = [];
  kasir.tambahItem(keranjang, "K04", 2);
  kasir.tambahItem(keranjang, "K04", 3);
  assert.equal(keranjang.length, 1);
  assert.equal(keranjang[0].qty, 5);
});

test("tambah produk sama dari input string menjumlahkan qty sebagai angka (2 + '3' = 5)", () => {
  const keranjang = [];
  kasir.tambahItem(keranjang, "K04", 2);
  kasir.tambahItem(keranjang, "K04", "3");
  assert.equal(keranjang[0].qty, 5);
});

// --- Bug: batas maksimum 99 per produk (aturan 2) ---

test("98 + 1 = 99 diterima", () => {
  const keranjang = [];
  kasir.tambahItem(keranjang, "K04", 98);
  kasir.tambahItem(keranjang, "K04", 1);
  assert.equal(keranjang[0].qty, 99);
});

test("98 + 2 melebihi batas dan ditolak", () => {
  const keranjang = [];
  kasir.tambahItem(keranjang, "K04", 98);
  assert.throws(() => kasir.tambahItem(keranjang, "K04", 2));
});

test("99 + 1 melebihi batas dan ditolak", () => {
  const keranjang = [];
  kasir.tambahItem(keranjang, "K04", 99);
  assert.throws(() => kasir.tambahItem(keranjang, "K04", 1));
});

test("operasi merge yang ditolak tidak mengubah keranjang (tanpa perubahan sebagian)", () => {
  const keranjang = [];
  kasir.tambahItem(keranjang, "K04", 99);
  try {
    kasir.tambahItem(keranjang, "K04", 1);
  } catch (e) {
    // diharapkan
  }
  assert.equal(keranjang.length, 1);
  assert.equal(keranjang[0].qty, 99);
  assert.equal(kasir.hitungSubtotal(keranjang), 99 * 4000);
});

test("tambah 3 x 99 tetap terkunci di 99 dan gagal pada penambahan setelah penuh", () => {
  const keranjang = [];
  kasir.tambahItem(keranjang, "K04", 40);
  kasir.tambahItem(keranjang, "K04", 59);
  assert.equal(keranjang[0].qty, 99);
  assert.throws(() => kasir.tambahItem(keranjang, "K04", 1));
  assert.equal(keranjang[0].qty, 99);
});

// --- Regresi: pengurangan item ---

test("hapusItem membuang produk dari keranjang", () => {
  const keranjang = [];
  kasir.tambahItem(keranjang, "K04", 2);
  kasir.tambahItem(keranjang, "K07", 1);
  const baru = kasir.hapusItem(keranjang, "K04");
  assert.equal(baru.length, 1);
  assert.equal(baru[0].kode, "K07");
});
