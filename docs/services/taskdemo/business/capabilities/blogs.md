---
id: blogs
service: taskdemo
status: planned
rules: BR-TASKDEMO-050..054, BR-TASKDEMO-060..067, BR-TASKDEMO-070..075, BR-TASKDEMO-080..084
updated: 2026-10-29
implementation:
---

# Blogs

## Purpose

Memungkinkan akun role user (non-admin) yang login menulis dan menerbitkan blog untuk pembaca umum, sambil mempertahankan kendali penulis atas karyanya. Admin dapat memoderasi tulisan melalui unpublish dan soft delete.

Dokumen ini merekam perilaku yang diminta, bukan hasil verifikasi implementasi. Alasan **Why** merupakan rasional dari konteks yang diberikan, bukan keputusan tambahan tentang perilaku.

## Ubiquitous language

| Term | Meaning |
|---|---|
| **Blog** | Tulisan milik seorang penulis, dengan title, content, dan status publikasi. |
| **Penulis / owner blog** | Akun yang membuat blog tersebut. |
| **Draft** | Tulisan yang belum ditampilkan di kanal publik. |
| **Published** | Tulisan yang telah dipublish; keterlihatan publik tetap bergantung pada kondisi blog dan akun penulis. |
| **Soft-deleted blog** | Blog yang ditandai dihapus tanpa penghapusan fisik; hanya admin dikecualikan dari larangan melihat. |
| **Moderasi blog** | Wewenang admin untuk meng-unpublish atau melakukan soft delete tulisan siapa pun. |
| **Slug** | Pengenal publik berbasis title dengan suffix ID pendek yang dibuat server; tetap sepanjang umur blog dan tidak dipakai ulang. |

Definisi akun, role admin/user, active, non-active, dan soft-deleted account mengacu pada [capability akun](./user-management-authentication-task-ownership.md). Soft delete blog berbeda dari soft delete akun.

## Actors and triggers

| Trigger | Actor | Effect |
|---|---|---|
| Membuat tulisan | Akun role user login yang active dan tidak soft-deleted | Blog baru berstatus draft dan dimiliki pembuat. |
| Mengubah tulisan | Owner login | Title/content miliknya berubah jika valid. |
| Publish / unpublish | Owner login | Tulisan diterbitkan / dikembalikan ke draft. |
| Moderasi | Admin login | Blog siapa pun di-unpublish atau di-soft delete. |
| Soft delete tulisan sendiri | Owner login | Tulisan ditandai dihapus. |
| Membaca list/detail publik | Siapa saja | Hanya tulisan yang memenuhi seluruh syarat visibilitas publik ditampilkan. |
| Membaca draft | Owner atau admin login | Akses baca privat terhadap draft. |
| Membaca tulisan terhapus | Admin login | Akses admin terhadap catatan tulisan terhapus. |

## Rules

Blok 050 (ownership), 060 (status/moderasi), 070 (visibilitas), dan 080 (validasi/pengenal publik) disetujui pada permintaan ini. Aturan penolakan dan pembatasan visibilitas tetap membatasi aturan izin; izin publish tidak melewati validasi atau penghapusan.

### BR-TASKDEMO-050 — Pembuatan blog hanya untuk role user yang login

**Given** seseorang belum login atau pemohon login memiliki role admin  
**When** ia mencoba membuat blog  
**Then** pembuatan ditolak; hanya role user (non-admin) yang boleh menjadi penulis.

**Why:** Penulis harus dapat dipertanggungjawabkan agar kepemilikan tulisan dapat dilindungi.

### BR-TASKDEMO-051 — Blog baru dimiliki pembuatnya

**Given** akun role user login yang active dan tidak soft-deleted  
**When** ia berhasil membuat blog  
**Then** user tersebut menjadi owner blog.

**Why:** Atribusi penulis menjadi dasar hak pengelolaan tulisan.

### BR-TASKDEMO-052 — Perubahan isi blog hanya oleh owner

**Given** blog belum soft-deleted dan pemohon bukan owner  
**When** pemohon mencoba mengubah title atau content  
**Then** perubahan ditolak, termasuk jika pemohon adalah admin.

**Why:** Moderasi tidak memberi pihak lain hak menulis ulang karya penulis.

### BR-TASKDEMO-053 — Operasi pengelolaan memerlukan login

**Given** pemohon tidak memiliki identitas login  
**When** ia mencoba mengubah, mempublish, meng-unpublish, atau menghapus blog  
**Then** operasi ditolak.

**Why:** Identitas pelaku dibutuhkan untuk melindungi tulisan dari perubahan anonim.

### BR-TASKDEMO-054 — Akun non-active atau soft-deleted tidak boleh mengelola atau membaca privat

**Given** akun pemohon non-active atau soft-deleted  
**When** ia mencoba membuat, mengubah, publish, unpublish, soft delete, moderasi, atau membaca blog melalui akses privat/admin  
**Then** operasi ditolak sebagai unauthenticated; pembacaan publik tetap dapat dilakukan sebagai pengunjung anonim.

**Why:** Pencabutan akses akun harus berlaku konsisten terhadap aktivitas menulis, moderasi, dan akses non-publik.

### BR-TASKDEMO-060 — Blog baru selalu draft

**Given** user yang memenuhi syarat mengirim title dan content yang valid untuk pembuatan  
**When** blog berhasil dibuat  
**Then** status awalnya draft.

**Why:** Tulisan perlu ruang persiapan sebelum ditampilkan kepada pembaca umum; pembuatan bukan persetujuan publikasi.

### BR-TASKDEMO-061 — Status blog hanya draft atau published

**Given** sebuah blog disimpan  
**When** status publikasinya ditetapkan  
**Then** status tersebut hanya draft atau published.

**Why:** Makna keterlihatan tulisan harus konsisten. Soft delete adalah kondisi penghapusan terpisah, bukan status publikasi ketiga.

### BR-TASKDEMO-062 — Hanya owner boleh mempublish

**Given** blog draft belum soft-deleted dan pemohon login adalah owner yang berhak bertindak  
**When** owner mempublish blog dengan title dan content yang memenuhi validasi  
**Then** blog menjadi published.

**Why:** Penulis menentukan kapan tulisannya siap dibaca publik.

Owner boleh mempublish ulang setelah admin melakukan unpublish, tanpa proses persetujuan tambahan. Unpublish merupakan penarikan sementara; soft delete menghilangkan kendali owner terhadap tulisan terhapus.

### BR-TASKDEMO-063 — Non-owner tidak boleh mempublish

**Given** pemohon bukan owner blog  
**When** ia mencoba mempublish blog  
**Then** publikasi ditolak, termasuk jika pemohon adalah admin.

**Why:** Wewenang moderasi untuk menarik tulisan tidak mencakup persetujuan menerbitkan atas nama penulis.

### BR-TASKDEMO-064 — Owner boleh meng-unpublish blog miliknya

**Given** blog published belum soft-deleted dan pemohon login adalah owner yang berhak bertindak  
**When** owner meng-unpublish blog  
**Then** status blog menjadi draft.

**Why:** Penulis perlu dapat menarik tulisan dari pembaca umum tanpa kehilangan tulisan itu.

### BR-TASKDEMO-065 — Admin boleh meng-unpublish blog siapa pun

**Given** admin login dan blog published belum soft-deleted  
**When** admin meng-unpublish blog  
**Then** status blog menjadi draft.

**Why:** Moderator perlu dapat menghentikan keterlihatan tulisan tanpa bergantung pada tindakan penulis.

### BR-TASKDEMO-066 — Owner dan admin boleh melakukan soft delete

**Given** blog belum soft-deleted dan pemohon login adalah owner yang berhak bertindak atau admin  
**When** pemohon menghapus blog  
**Then** blog ditandai soft-deleted.

**Why:** Penulis dan moderator perlu dapat menarik tulisan sambil mempertahankan catatannya.

Publish terhadap published dan unpublish terhadap draft bersifat idempotent selama pemohon tetap berhak dan blog belum terhapus. Delete berulang oleh admin sukses tanpa perubahan; delete berulang oleh owner non-admin menghasilkan 404 karena blog terhapus tidak boleh terlihat.

### BR-TASKDEMO-067 — Pengguna lain tidak boleh menarik atau menghapus blog

**Given** pemohon bukan owner maupun admin  
**When** ia mencoba meng-unpublish atau menghapus blog  
**Then** operasi ditolak.

**Why:** Pembaca tidak boleh mengambil alih kendali penulis maupun moderator.

### BR-TASKDEMO-070 — List dan detail publik dapat dibaca tanpa login

**Given** pengunjung dengan atau tanpa login  
**When** ia mengakses list atau detail publik untuk blog yang memenuhi syarat publik  
**Then** akses baca tidak mensyaratkan login.

**Why:** Tulisan yang diterbitkan ditujukan untuk menjangkau siapa saja.

### BR-TASKDEMO-071 — Hanya published yang dapat muncul di kanal publik

**Given** blog berstatus draft  
**When** pengunjung meminta list atau detail publik  
**Then** blog tersebut tidak ditampilkan.

**Why:** Tulisan yang belum diterbitkan tidak boleh terbuka melalui akses publik, termasuk ketika pemohon kebetulan adalah owner atau admin.

### BR-TASKDEMO-072 — Draft hanya dapat dilihat owner dan admin

**Given** blog draft belum soft-deleted  
**When** seseorang meminta akses baca privat  
**Then** draft dapat dilihat hanya jika pemohon login adalah owner atau admin.

**Why:** Penulis memerlukan privasi saat menyiapkan tulisan; moderator tetap membutuhkan akses pengawasan.

### BR-TASKDEMO-073 — Blog soft-deleted disembunyikan dari non-admin

**Given** blog sudah soft-deleted  
**When** pemohon non-admin mengakses endpoint apa pun yang dapat menampilkan blog  
**Then** blog tidak ditampilkan, termasuk kepada owner.

**Why:** Penghapusan harus menghilangkan keterlihatan tulisan bagi pengguna biasa, bukan sekadar mengeluarkannya dari list.

### BR-TASKDEMO-074 — Admin merupakan pengecualian akses blog soft-deleted

**Given** admin login  
**When** ia menggunakan akses admin untuk melihat blog soft-deleted  
**Then** blog dapat dilihat melalui akses admin.

**Why:** Moderator memerlukan catatan tulisan yang sudah ditarik.

Akses tersedia melalui list/detail khusus admin dengan filter penghapusan. Blog terhapus hanya dapat dibaca; restore, edit, publish ulang, hard delete, dan kebijakan retensi tambahan tidak termasuk scope awal.

### BR-TASKDEMO-075 — Published milik akun non-active atau soft-deleted tidak tampil publik

**Given** blog published dan akun owner non-active atau soft-deleted  
**When** pengunjung mengakses list atau detail publik  
**Then** blog tidak ditampilkan.

**Why:** Akun yang aksesnya dicabut tidak boleh tetap menjadi penerbit aktif di kanal publik.

Perubahan status/penghapusan akun tidak mengubah status publikasi blog. Kelayakan akun owner diperiksa saat list/detail publik dibaca; published yang belum terhapus muncul kembali ketika owner kembali active dan tidak soft-deleted, tanpa publish ulang.

### BR-TASKDEMO-080 — Title wajib berpanjang 10–200 karakter

**Given** permintaan membuat atau mengubah title blog  
**When** title kosong, lebih pendek dari 10 karakter, atau lebih panjang dari 200 karakter  
**Then** permintaan ditolak.

**Why:** Judul harus cukup informatif untuk mengenali tulisan sekaligus tetap ringkas.

Title di-trim sebelum dihitung dengan Unicode code points. Teks content dihitung dengan Unicode code points setelah sanitasi HTML, decoding entities, normalisasi whitespace menjadi spasi tunggal, dan trim. Batas elemen blok dan `br` menjadi pemisah teks; markup tidak dihitung. Whitespace-only dianggap kosong.

### BR-TASKDEMO-081 — Content wajib saat publish

**Given** owner mengajukan publikasi blog  
**When** content tidak diisi  
**Then** publikasi ditolak.

**Why:** Halaman publik harus memberikan isi yang bisa dibaca, bukan hanya judul.

### BR-TASKDEMO-082 — Content publikasi harus sepanjang 100–1000 karakter

**Given** owner mengajukan publikasi dengan content terisi  
**When** panjang content di luar batas inklusif 100–1000 karakter  
**Then** publikasi ditolak.

**Why:** Pembaca membutuhkan isi yang cukup bermakna dengan panjang yang terkendali.

Draft boleh tanpa content atau kurang dari 100 karakter, tetapi tetap maksimal 1000 karakter. Owner boleh mengedit published langsung dan hasilnya langsung terlihat jika masih memenuhi validasi publikasi; edit invalid ditolak tanpa mengubah data atau status.

### BR-TASKDEMO-083 — Content mendukung HTML yang disebutkan

**Given** content memenuhi aturan lain yang telah diputuskan  
**When** penulis menggunakan tag p, br, h1, h2, h3, h4, h5, atau a  
**Then** penggunaan tag tersebut diperbolehkan.

**Why:** Penulis membutuhkan struktur tulisan dan tautan agar isi mudah dibaca.

Tag tersebut merupakan allowlist tertutup. Hanya atribut `href` pada `a` yang diizinkan, dengan URL absolut HTTP/HTTPS. HTML disanitasi di server sebelum validasi panjang dan penyimpanan; hasil sanitasi yang disimpan dan dikembalikan. Tag/atribut di luar kebijakan tidak boleh bertahan dalam hasil sanitasi.

### BR-TASKDEMO-084 — Detail publik memakai pengenal berbasis title

**Given** blog memenuhi seluruh syarat visibilitas publik  
**When** pengunjung mengakses detail publik  
**Then** blog dapat ditemukan menggunakan pengenal berbasis title, bukan ID blog.

**Why:** Alamat tulisan perlu dapat dikenali pembaca.

Slug dibuat server saat create dari title ditambah suffix ID pendek. Title identik diperbolehkan, tetapi slug harus unik. Slug tetap ketika title diedit dan tetap dicadangkan setelah soft delete. Detail publik hanya melalui slug; UUID digunakan untuk pengelolaan/private/admin. Aturan normalisasi, fallback title tanpa hasil slug, panjang suffix, dan penanganan collision akan dirumuskan dalam tripack sesuai kebijakan ini.

## Edge cases

| Situation | Expected behaviour | Rule |
|---|---|---|
| Admin mencoba membuat blog | Ditolak; penulis hanya role user | `BR-TASKDEMO-050` |
| Owner meminta draft lewat kanal publik | Draft tidak ditampilkan; gunakan akses privat | `BR-TASKDEMO-071`, `072` |
| Admin mencoba mengedit atau mempublish blog orang lain | Ditolak | `BR-TASKDEMO-052`, `063` |
| Owner mencoba membaca blog sendiri setelah soft delete | Tidak ditampilkan kepada owner non-admin | `BR-TASKDEMO-073` |
| Blog published tetapi akun owner menjadi non-active | Hilang dari list dan detail publik | `BR-TASKDEMO-075` |
| Blog terhapus tetap memiliki status published | Tidak tampil publik; pengecualian baca hanya akses admin | `BR-TASKDEMO-073`, `074` |
| Content kosong ketika publish | Ditolak | `BR-TASKDEMO-081` |
| Title atau content berada tepat di batas | Batas inklusif dengan metode hitung yang telah disetujui | `BR-TASKDEMO-080`, `082` |

## Examples

Contoh panjang menggunakan ASCII tanpa spasi tepi atau HTML; metode hitung untuk HTML mengikuti aturan di atas.

| Input | Expected outcome |
|---|---|
| Budi (user active, login) membuat title `Catatan Go` (10 karakter), content `a` diulang 100 kali | Blog draft, owner Budi |
| Admin active mencoba create title `Catatan Go` dan content valid | Ditolak (403) |
| Budi membuat title `Catatan Go` tanpa content | Draft diterima |
| Title `Catatan` (7 karakter) saat create | Ditolak |
| Title `a` diulang 200 kali / 201 kali | Memenuhi batas title / ditolak |
| Owner publish dengan content `a` diulang 99 / 100 / 1000 / 1001 kali dan title valid | Ditolak / memenuhi batas / memenuhi batas / ditolak |
| Citra mencoba mengedit blog Budi | Ditolak |
| Admin meng-unpublish blog published Budi | Status menjadi draft; hilang dari kanal publik |
| Pengunjung tanpa login membaca blog published milik Budi active, belum dihapus | Diizinkan melalui pengenal berbasis title |
| Budi soft delete blognya lalu mencoba membacanya sebagai owner non-admin | Tidak ditampilkan; admin tetap dapat melihat melalui akses admin |

## Known gaps

Not yet implemented — verify against code after implementation.

## Open questions

None — OQ-01–14 telah diputuskan pengguna pada 2026-10-29. Detail teknis kontrak dan normalisasi slug disusun dalam tripack berdasarkan keputusan berikut; bukan blocker bisnis baru.

### Approved decisions — 2026-10-29

Pengguna menyetujui usulan OQ-02–14 dan mengganti usulan OQ-01: admin tidak boleh membuat blog. Tabel ini merekam keputusan, bukan bukti implementasi.

| ID | Keputusan disetujui |
|---|---|
| OQ-01 | Hanya role user (non-admin) boleh membuat blog. Admin berperan sebagai moderator/pembaca berwenang, bukan penulis. |
| OQ-02 | Akun non-active/soft-deleted ditolak untuk seluruh pengelolaan dan akses privat/admin, termasuk moderasi. Kanal publik tetap dapat dibaca sebagai pengunjung anonim. |
| OQ-03 | Draft boleh kosong atau di bawah 100 karakter, maksimal 1000. Publikasi wajib 100–1000. Edit published invalid ditolak tanpa mengubah data/status. |
| OQ-04 | Title: trim lalu hitung Unicode code points. Content: hitung Unicode code points teks setelah sanitasi dan decoding entities, normalisasi whitespace menjadi satu spasi, lalu trim; elemen blok/br memisahkan teks. Markup tidak dihitung. |
| OQ-05 | Allowlist tertutup p/br/h1–h5/a; hanya href pada a, URL absolut HTTP/HTTPS. Sanitasi server sebelum validasi panjang dan penyimpanan; simpan/kembalikan hasil sanitasi. |
| OQ-06 | Slug berbasis title dengan suffix ID pendek buatan server; slug unik, title identik boleh. Detail normalisasi/fallback/collision disusun dalam tripack. |
| OQ-07 | Slug dibuat saat create, tetap saat rename, dan tidak dipakai ulang setelah soft delete. Publik melalui slug saja; pengelolaan/private/admin memakai UUID. |
| OQ-08 | Status blog tidak berubah akibat lifecycle akun. Public read memeriksa kelayakan akun saat dibaca; published muncul kembali setelah akun kembali active dan tidak terhapus, selama blog belum terhapus. |
| OQ-09 | List/detail khusus admin dengan filter penghapusan. Blog terhapus hanya dibaca. Restore/edit/publish ulang atas blog terhapus, hard delete, dan retensi tambahan di luar scope awal. |
| OQ-10 | Owner boleh publish ulang setelah admin unpublish, tanpa persetujuan tambahan. Alasan dan notifikasi moderasi di luar scope awal. |
| OQ-11 | Publish/unpublish idempotent jika aktor berhak dan blog belum terhapus. Delete berulang admin sukses tanpa perubahan; owner non-admin mendapat 404 setelah blog terhapus. |
| OQ-12 | Published boleh diedit langsung oleh owner; perubahan valid langsung terlihat publik. Perubahan invalid ditolak tanpa perubahan data/status. |
| OQ-13 | List publik: page/pageSize, default pageSize 20, maksimum 100; createdAt DESC dengan ID sebagai tie-breaker. Item berisi ID, slug, title, nama penulis, waktu pembuatan; content lengkap hanya pada detail. Search/filter publik lain dan list privat owner ditunda. List admin OQ-09 tetap termasuk scope. |
| OQ-14 | 401 untuk pengelolaan/akses protected tanpa identitas valid; 403 untuk operasi terlarang atas blog yang boleh dilihat; 404 untuk blog hilang/tidak boleh terlihat dan slug tak ditemukan; validasi 400; konflik state 409. |

### Approved contract constraints

Gunakan pagination dan field list pada OQ-13 serta pemetaan error pada OQ-14 saat menyusun kontrak HTTP. Pertahankan envelope existing. Path, method, request/response detail, success status, filter penghapusan admin, dan aturan slug yang belum dirinci di atas merupakan pekerjaan tripack. Keputusan ini tidak memberi izin implementasi kode atau mengubah kontrak task/account.

## Related

- Sibling capabilities: [user-management-authentication-task-ownership](./user-management-authentication-task-ownership.md) — definisi akun dan autentikasi; pembatasan khusus task tidak otomatis berlaku pada blog.
- Sibling capabilities: [task-tracking](./task-tracking.md) — konteks service; aturan panjang title task tidak berlaku pada blog.
- Source document: permintaan “capability-doc: Fitur: Blogs” beserta 11 invariant dan persetujuan blok BR (2026-10-29).
- Decision source: persetujuan pengguna atas usulan OQ-02–14 dan koreksi OQ-01 (2026-10-29): admin tidak boleh membuat blog.
