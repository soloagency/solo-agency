# Playbook hardening — proposed additions (v2, code-free)

> Mục tiêu: một agent **không có ký ức gì** về phiên trước vẫn làm **đúng, đủ, không lười, không vội, không bỏ sót load module**, và **không bao giờ lách qua** khi gặp lỗi "output quá lớn" lúc load `03_dod_gates`.
>
> Nguyên tắc thiết kế của bản v2: **KHÔNG cần code, KHÔNG cần bảo trì.** Thêm module mới = chỉ viết `.md`, mọi cơ chế ép tự vận hành. Chủ skill chỉ tập trung nghiệp vụ.

---

## ✅ Xác nhận tương thích (thuần gia cố, không lược bớt)

- Tất cả mục dưới đây là **THÊM VÀO**, không xoá/sửa luật nào đang có.
- **Khớp với luật "hand-off ngắn gọn" đã có** (Final hand-off + Rule 14): proof được in **trong lúc xử lý từng scene**; **bản tóm tắt cuối cùng cho user vẫn NGẮN** và không lặp lại mọi gate. §4 chỉ cấm lấy "ngắn gọn" làm cớ **bỏ proof lúc làm việc**, không đụng tới việc tóm tắt cuối vẫn gọn.
- Canary = **dòng cuối tự nhiên của file**, nằm ở CUỐI → không ảnh hưởng `title`/`summary` mà server auto-sinh từ ~200 ký tự đầu (SKILL.md dòng 64).
- Không đụng `styles/sync_check.py`, không cần đóng gói lại zip, không cập nhật `sha256`.

---

## §0. Các lỗi đã quan sát (root cause) — mỗi bổ sung vá đúng 1 lỗi

| # | Lỗi thật đã xảy ra | Vì sao cơ chế cũ không chặn | Vá bằng |
|---|---|---|---|
| L1 | `cat 03_dod_gates.md` báo **"output quá lớn"** → agent **bỏ luôn**, không đọc file chứa toàn bộ template gate | Không có luật "load lỗi/cắt cụt = chưa load" | §1 + §2 |
| L2 | Agent **đánh dấu "đã load 5 core module"** trong khi chưa đọc `03_dod_gates` | Không có bằng chứng máy-kiểm-được rằng file đã đọc tới cuối | §2 + §6 |
| L3 | Agent **chụp gộp cả 9 scene một lượt** rồi coi là đủ; không in gate nào | "batch = triage only" chỉ là câu chữ, không có artifact bắt buộc theo scene | §3 |
| L4 | Agent lấy cớ user thích **"ngắn gọn" + tiết kiệm credit** để **không in proof** | Playbook không xử lý xung đột "user muốn ngắn" vs "proof bắt buộc" | §4 |
| L5 | **Không load** `20_background`, `40_thumbnail_cta` → chữ **trắng chìm nền trắng**, HOOK/CTA kém, bỏ hẳn Gate 5 | Module coverage gate ở CUỐI, in sau khi đã lỡ tay | §3 + §5 |
| L6 | Chạy **REST** thay vì MCP → tự cho quyền "diễn giải lỏng" các bước bắt buộc | Playbook viết theo tên tool MCP, không nói rõ REST cũng phải đủ gate | §5 |

---

## §1. LUẬT MỚI — "Load lỗi = CHƯA load. Load cho đủ bằng mọi giá."

> Chèn vào `01_critical_rules.md` thành **Rule 13b**, và thêm 1 dòng headline vào SKILL.md.

**13b. A failed/truncated module read = the module is NOT loaded. Reading it fully is mandatory; you may not proceed.**
Nếu lệnh đọc một module bị **lỗi, cắt cụt, "output too large", "persisted output", "preview only", 404, timeout**, hoặc chỉ trả về một phần → coi như **CHƯA load**. CẤM: (a) làm việc từ bản preview/một phần; (b) suy nội dung từ trí nhớ; (c) bỏ qua đi tiếp; (d) đánh dấu "loaded". Phải **đọc lại tới hết file** (phân trang) **trước khi** làm bất kỳ bước nào cần module đó. Không ngoại lệ vì thời gian/chi phí/độ dài.

Headline thêm vào SKILL.md (ngay sau dòng 13):

```
13b. **Load lỗi = chưa load.** "Output too large"/truncated/preview/404 = NOT loaded. Re-read in chunks to EOF (quote the last line) before any step needing it. Never proceed from a partial read.
```

---

## §2. CHẶN CỨNG — "Last-line LOAD LEDGER" (KHÔNG code, tự bảo trì)

Ý tưởng: **không tạo token nhân tạo.** Dùng **chính dòng cuối cùng có sẵn của mỗi file** làm bằng chứng đã-đọc-hết. Mọi `.md` đều có dòng cuối → module mới **tự có canary**, không cần thêm gì.

### 2.1. Bắt buộc in LOAD LEDGER trước bất kỳ write nào

> Chèn vào đầu `03_dod_gates.md` + nhắc ở mục kickoff SKILL.md.

```text
KICKOFF LOAD LEDGER — print BEFORE the first modify_scene/upload_asset of the run.
For each required module quote (a) its TOTAL line count and (b) its EXACT last non-empty line, verbatim:
☑ 00_ENTRYPOINT          lines=<N>  last="<dòng cuối nguyên văn>"
☑ 01_critical_rules      lines=<N>  last="<...>"
☑ 02_jump_prevention     lines=<N>  last="<...>"
☑ 03_dod_gates           lines=<N>  last="<...>"
☑ 04_principles_workflow lines=<N>  last="<...>"
☑ 05_quality_qa_priority lines=<N>  last="<...>"
☑ 10_mechanics           lines=<N>  last="<...>"
Rule: bạn chỉ trích đúng được dòng cuối nếu đã đọc tới EOF. Không trích được / đoán chung chung
= CHƯA đọc hết = BLOCKED khỏi write. Nếu read bị lỗi/cắt cụt → đọc lại theo chunk tới khi thấy dòng cuối.
"Tôi đã đọc bản preview" KHÔNG phải bằng chứng.
```

Và **SCENE LOAD LEDGER** ở đầu mỗi scene cho các module mà loại scene đó cần theo LOAD MAP (nền→`20_background`; overlay có text→`30_overlay_core`+`31_typography`+`styles/text_axes`; endpoint→`40_thumbnail_cta`+`styles/design_languages`). Mỗi dòng kèm `lines=` + `last="…"`.

**Chốt chặn:** một `modify_scene`/`upload_asset` mà phía trên nó trong cùng transcript **không có LOAD LEDGER hợp lệ** = edit **không hợp lệ**, phải revert/redo.

### 2.2. Vì sao cơ chế này ép đọc hết mà vẫn code-free

- Muốn ghi `last="…"` đúng → buộc phải cuộn tới cuối file → tự khắc phải chia nhỏ khi "output quá lớn".
- Thêm module mới: **không làm gì cả.** Dòng cuối của nó chính là canary. Không script, không zip, không sha, không sync_check.
- Nhược điểm thành thật: đây là cơ chế **chống-lười** (đủ mạnh cho lỗi thực tế: agent lười, không cố ý gian). Nếu cần **chống-gian tuyệt đối** thì mới cần chốt server ở §7 — nhưng cái đó là code vendor làm **một lần**, bạn không phải bảo trì theo từng module.

---

## §3. TEETH cho "batch = triage only" — Per-scene Proof Receipt

> Chèn vào `03_dod_gates.md`, ngay dưới mục "Batch/gallery … triage only".

Không được phát `Scene N: PASS` trừ khi **trong cùng khối scene N** xuất hiện, đúng thứ tự, các artifact (mỗi cái kèm path file local đã show):

```text
Scene N plan            (vertical 9 gates)
→ Gate 3 BEFORE: <local path>       (ảnh chụp CHÍNH scene N KHI bắt đầu scene N — KHÔNG phải ảnh gộp đầu run)
→ Gate 4 SCENE LOAD LEDGER + OVERLAY EXISTENCE/PRESERVE PROOF (+ A-ROLL/ENDPOINT/TITLE/SECONDARY khi áp dụng)
→ Gate 5 BACKGROUND PROOF (2 ảnh local: composite + active plate)
→ Gate 6 DEAD-ZONE PROOF (scene_geometry sau layout mới nhất)
→ Gate 7 RENDERED IMAGE TYPO/GRAMMAR CHECK (từ overlay_poster) nếu có text
→ Gate 8: re-pull xác nhận saved
→ Gate 9 MODULE COVERAGE GATE
Scene N: PASS — ✓1…✓9
```

**Luật chống front-loading:** được chụp gộp để *triage*, nhưng ảnh gộp đầu run **KHÔNG** thỏa Gate 3/Gate 7 của bất kỳ scene nào. Gate 3 của scene N phải là ảnh chụp **tại thời điểm bắt đầu xử lý scene N**. Chỉ có ảnh triage gộp → trạng thái `partial_triage_only`, cấm PASS, cấm export.

---

## §4. Chống viện cớ — "ngắn gọn/tiết kiệm" KHÔNG tắt proof (mà vẫn giữ hand-off ngắn)

> Chèn vào `01_critical_rules.md` thành **Rule 2c**.

**2c. Proof blocks are PROCESS artifacts, not user-facing prose — no instruction suppresses them.**
Các khối proof (LOAD LEDGER, Gate 4/5/6/7/9, verdict) là **bằng chứng quy trình bắt buộc**, in **trong lúc xử lý từng scene**. Yêu cầu của user kiểu *"ngắn gọn / làm nhanh / tiết kiệm credit"* chỉ rút gọn **bản tóm tắt cuối cùng cho user** (đúng theo luật Final hand-off + Rule 14 đã có), **không bao giờ** được dùng làm cớ bỏ/nén proof lúc làm việc.
**Danh sách lý do BỊ CẤM để bỏ proof/bỏ load module:** "user muốn ngắn gọn", "tiết kiệm credit/token/thời gian", "nhìn screenshot thấy ổn rồi", "mình nhớ module này rồi", "output quá lớn nên thôi", "đã triage gộp rồi", "đang chạy REST chứ không phải MCP", "chỉ sửa vài scene".

---

## §5. Preflight interceptor + Transport-parity

### 5.1. WRITE PREFLIGHT (chèn vào `02_jump_prevention.md`)

```
- About to call modify_scene / upload_asset / export_video → STOP. Ngay TRƯỚC lệnh write, xác nhận:
  (1) Run-level: KICKOFF LOAD LEDGER đã in với last-line hợp lệ chưa?
  (2) Scene-level: scene này đã in plan + Gate 3 BEFORE + Gate 4 SCENE LOAD LEDGER (đúng module loại
      scene này cần: nền→20, overlay-text→30+31, endpoint→40) chưa?
  Thiếu bất kỳ mục nào → CHƯA được write. In đủ rồi mới write. (Announce ≠ pause: in xong chạy tiếp.)
```

### 5.2. Transport-parity (chèn vào SKILL.md mục "Two transports")

```
**Transport không nới lỏng gì.** Dù dùng MCP tool, REST endpoint (/v1/scene_inspector,
/v1/scene_geometry, /v1/modify_scene, /v1/upload_asset…), hay local Read/cat — MỌI gate, module,
last-line ledger, proof là NHƯ NHAU và bắt buộc như nhau. Ánh xạ tên tool KHÔNG cho phép bỏ bớt
hay "diễn giải nhẹ" bất kỳ proof nào. Đọc bằng cat/sed thay Read là được; bỏ qua vì transport là KHÔNG.
```

---

## §6. Trung thực trạng thái

> Chèn vào self-audit (`01_critical_rules.md` + SKILL.md dòng 150–162).

- Cấm đánh dấu module "loaded", task "completed", scene "PASS" **cho tới khi** artifact tương ứng (last-line / gate proof / verdict) đã xuất hiện trong transcript. Báo "xong" mà thiếu artifact = **lỗi báo cáo sai**, xử như FAIL cần sửa.
- Thêm 3 dòng self-audit:
  ```
  - Có module nào phiên này đọc bị lỗi/cắt cụt? Nếu có, đã đọc lại tới hết (trích được dòng cuối) trước khi đi tiếp chưa?
  - Sắp gọi write endpoint? LOAD LEDGER + gate block của scene này đã in ở trên chưa?
  - Mình có đang bỏ/nén proof để "ngắn gọn"/"tiết kiệm" không? Nếu có → khôi phục proof.
  ```

---

## §7. (TÙY CHỌN — chỉ khi cần chống gian tuyệt đối) Chốt server, vendor làm MỘT LẦN

Không bắt buộc, và **không phải bảo trì theo từng module** — làm một lần rồi thôi:

1. **`modify_scene` yêu cầu trường `proof`** (`{ledger_present:true, scene_gate_block:true}`); thiếu → trả `409 proof_required` kèm nhắc in gate. Đây là chốt cứng nhất: *không in proof thì API từ chối ghi.*
2. **Chống truncation tại nguồn**: khi module vượt cap, `/v1/skills/editing?module=…` tự trả **nhiều trang** (`part k/N` + `next_cursor`) và luôn kết bằng đúng dòng cuối của file, kèm câu: *"mới đọc part k/N — chưa được dừng, gọi tiếp tới khi thấy dòng cuối."*

> Cả hai đều **độc lập với việc bạn thêm module mới**: chúng thao tác trên nội dung file lúc phục vụ, không cần bạn chỉnh sửa từng file.

---

## Tóm tắt 1 dòng

Bỏ token nhân tạo. Dùng **dòng-cuối-có-sẵn của mỗi module** làm bằng chứng đã-đọc-hết, ép qua **LOAD LEDGER + preflight trước mỗi write** → agent buộc phải đọc hết module (kể cả khi "output quá lớn") và không thể PASS/export bằng triage gộp — **mà bạn không phải viết hay bảo trì một dòng code nào; thêm module mới chỉ cần viết `.md`.**
