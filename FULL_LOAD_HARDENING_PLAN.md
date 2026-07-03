# Plan — ép agent LOAD ĐỦ FILE & ĐỦ DÒNG (Solo Agency)

> Trạng thái: **ĐỀ XUẤT — chưa sửa file nào.** Chờ duyệt rồi mới triển khai.
> Mục tiêu: một agent không có ký ức về phiên này **không thể** hành động trên một stage/module đọc thiếu, kể cả khi file rất lớn báo "output quá lớn" hoặc bản tải từ GitHub bị cắt cụt.

---

## 1. Vì sao repo này rủi ro hơn repo video

| # | Đặc điểm repo | Rủi ro đọc-thiếu |
|---|---|---|
| R1 | Load theo kiểu **"chỉ load stage cần cho hành động"** + file rất lớn: `09`=1893, `07`=1873, `00`=1805, `08`=1586, `06`=1503, `03`=1061 dòng | Load 1 stage lớn → bị cắt cụt → agent làm việc trên phần đầu, y hệt lỗi `03_dod_gates` ở repo video, nhưng file to gấp 4× nên xác suất cao hơn |
| R2 | **Đa tầng phụ thuộc**: Stage 3 → 3A adapter → 3B `skills/video-editing/SKILL.md` → `ai_video_editor/*` → `styles/*` | Load cha, **quên load con**; không có gì buộc từng dependency phải được load ĐỦ |
| R3 | Có **fallback tải từ GitHub raw** khi thiếu file | Bản tải về có thể **cụt/stale**; hiện KHÔNG kiểm tra toàn vẹn nội dung `.md` (chỉ binary collector mới có `SHA256SUMS`) |
| R4 | Completion Gates ghi **"Stage X was loaded"** (nhị phân) | Agent đọc 200/1893 dòng vẫn "thành thật" nói *đã loaded* |
| R5 | Không công bố **kích thước kỳ vọng** của file | Agent không có cách nào **tự biết** mình vừa nhận file cụt |

Cơ chế ràng buộc hiện có (Stage Map, Jump-Prevention, Completion Gates, Self-Audit) đều là **"đã load hay chưa"** — không có mức **"đã load ĐỦ chưa"**. Đó là khe hở.

---

## 2. Nguyên tắc thiết kế (giữ đúng ưu tiên của bạn: không bảo trì tay)

- **Tier A — code-free, zero-maintenance:** dùng *dòng-cuối + số-dòng tự báo* làm bằng chứng đọc tới EOF. Áp cho MỌI file, kể cả file mới thêm sau này, **không cần thao tác gì**.
- **Tier B — mạnh hơn, chạy trên hạ tầng ĐÃ CÓ:** tự sinh `playbooks/LOAD_MANIFEST.md` (đường dẫn + số dòng kỳ vọng + dòng cuối + sha256 cho mọi `.md`) **ngay trong `deploy-soloagency.sh`** (tái dùng `sha256_for_file`). Agent so khớp file vừa đọc với manifest → phát hiện **cắt cụt/stale một cách xác định**. Manifest được tái sinh mỗi lần deploy → **thêm playbook mới không cần làm gì tay**.

Tier A luôn hoạt động kể cả khi chưa có Tier B. Tier B biến "chống lười" thành "chống cả cụt file/tải lỗi".

---

## 3. Cơ chế cốt lõi — PER-LOAD LOAD LEDGER (in tại thời điểm load, TRƯỚC khi hành động)

Khác repo video (có "kickoff set" cố định), ở đây load theo yêu cầu → ledger phải in **mỗi lần load một stage/dependency**, ngay trước khi dùng nó.

```text
LOAD LEDGER (in ngay sau khi load, trước khi hành động trên stage này):
File: playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md
lines_read=<N>            last="<dòng cuối nguyên văn>"
manifest_lines=<M>       manifest_last="<...>"     (Tier B; nếu chưa có manifest ghi 'manifest=absent')
Full-read check: <PASS lines_read==manifest_lines & last==manifest_last | FAIL — re-read/re-fetch>
Dependencies named by this stage: <list, mỗi cái phải có LOAD LEDGER riêng ở trên>
Verdict: <PASS loaded-in-full | BLOCKED — chưa được hành động trên stage này>
```

- **Tier A** (chưa có manifest): chỉ cần `lines_read` + `last="…"`. Trích đúng dòng cuối = đã đọc tới EOF.
- **Tier B** (có manifest): bắt buộc `lines_read==manifest_lines` **và** `last==manifest_last`. Lệch = cụt/stale → cấm hành động, phải đọc lại đủ hoặc tải lại từ GitHub và so lại.
- **Dependency-complete:** stage nào "Load When" trỏ tới dependency (vd 3→3A→3B→skill) thì mỗi dependency phải có dòng ledger riêng; cha **chưa tính loaded** tới khi mọi con đã ledger.

---

## 4. Các điểm sửa cụ thể (thuần THÊM, không lược)

### 4.1. `SOLO_AGENCY_PLAYBOOK.md` (master)
1. **Mục mới "Full-Load Discipline"** (đặt ngay sau *First Instruction To The Agent*): định nghĩa "load lỗi/cụt = CHƯA load", cấm hành động trên bản một phần, và yêu cầu in LOAD LEDGER cho mỗi lần load. Kèm mệnh đề chống-viện-cớ (mirror Rule 13b/2c đã dùng cho repo video): cấm lấy "file quá lớn / tiết kiệm / nhớ rồi / đang chạy schedule" làm cớ bỏ đọc.
2. **First Instruction To The Agent** (dòng 9–16): đổi "load 00 và 01" → "load 00 và 01 **IN FULL, in LOAD LEDGER**; chỉ hỏi câu đầu sau khi 2 ledger PASS".
3. **Stage Map** (dòng 430): thêm một câu quy tắc "mỗi lần load ⇒ một LOAD LEDGER"; (Tier B) thêm cột/nhắc *expected lines từ LOAD_MANIFEST*.
4. **Fresh GitHub Source And Missing Playbook Download Rule** (dòng 374): thêm bước — sau khi tải file playbook từ raw GitHub, **so số dòng + dòng cuối (+sha) với LOAD_MANIFEST**; lệch = bản tải cụt/stale, tải lại; **cấm hành động trên bản tải một phần**.
5. **Completion Gates** (dòng 713): đổi mọi "Stage X was loaded" → **"Stage X was loaded IN FULL (LOAD LEDGER printed; khớp manifest)"**. Áp cho cả 5 nhóm gate (Setup, Update, Private, Production, Measurement, Daily).
6. **Jump-Prevention Rules** (dòng 785): thêm 2 interrupt:
   - "Sắp hành động trên một stage vừa load bị lỗi/cụt/đọc-thiếu → STOP, đọc lại tới EOF / tải lại rồi so manifest."
   - "Sắp chạy tác vụ có side-effect (report/scan/render/publish/notify) mà stage tương ứng **chưa có LOAD LEDGER PASS** ở trên → STOP, load đủ trước."
7. **Self-Audit Summary** (dòng 800): thêm câu hỏi:
   - "Có file nào phiên này đọc bị lỗi/cụt không? Đã đọc lại đủ (khớp dòng cuối / manifest) chưa?"
   - "Mọi stage tôi đang dùng đã có LOAD LEDGER PASS + dependency đã ledger đủ chưa?"

### 4.2. Module dùng chung mới — `playbooks/LOAD_LEDGER_PROTOCOL.md`
Chứa template LOAD LEDGER, luật full-read, luật đọc file lớn theo chunk (Read `offset/limit`, `sed -n 'A,Bp'`, phân trang), và cách so manifest. Được tham chiếu bởi master + **cả hai entrypoint** (`SETUP_FLOW_ENTRYPOINT.md`, `SCHEDULED_RUN_ENTRYPOINT.md`) để cả Setup Flow lẫn Scheduled Flow đều thừa hưởng — vì scheduled run là nơi "làm theo trí nhớ" dễ xảy ra nhất.

### 4.3. Hai entrypoint
Thêm ở đầu mỗi entrypoint một dòng: "Trước mọi hành động, tuân thủ `LOAD_LEDGER_PROTOCOL.md`; mỗi stage/dependency phải có LOAD LEDGER PASS."

### 4.4. `deploy-soloagency.sh` (Tier B — một-lần, sau đó tự chạy)
Thêm hàm `generate_load_manifest()` (tái dùng `sha256_for_file` đã có): duyệt mọi `playbooks/**/*.md`, ghi `playbooks/LOAD_MANIFEST.md` gồm `path | lines | last_line | sha256`, và publish lên GitHub raw như các artifact khác. Chạy tự động mỗi lần deploy ⇒ thêm/sửa playbook **không phát sinh việc tay**.

### 4.5. Skill video-editing lồng trong repo này
`playbooks/skills/video-editing/` ở đây là **bản sao riêng** — cần port đúng các bổ sung mình vừa làm ở repo `transcoder` (Rule 2c/13b, LOAD LEDGER, per-scene receipt, write-preflight). Ghi chú để đồng bộ, tránh mỗi nơi một phách.

---

## 5. Thứ tự triển khai đề xuất

1. Tạo `playbooks/LOAD_LEDGER_PROTOCOL.md` (nguồn chân lý của cơ chế).
2. Chèn 7 sửa đổi vào `SOLO_AGENCY_PLAYBOOK.md` (thuần thêm).
3. Chèn dòng tham chiếu vào 2 entrypoint.
4. Thêm `generate_load_manifest()` vào `deploy-soloagency.sh` + sinh `LOAD_MANIFEST.md` lần đầu.
5. Port hardening vào `playbooks/skills/video-editing/`.
6. Verify: đọc lại diff, kiểm anchor gốc còn nguyên, đếm dòng tăng đúng, fence cân bằng; chạy thử manifest-generate và so vài file lớn.

**Không đụng** nội dung nghiệp vụ; chỉ thêm lớp ép-đọc-đủ.

---

## 6. Câu chuyện bảo trì (chốt lại đúng mối quan tâm của bạn)
- **Thêm playbook mới:** Tier A tự có (dòng cuối luôn tồn tại); Tier B tự cập nhật khi deploy. **Bạn không làm gì tay.**
- **Không có script phải nuôi riêng:** manifest ăn theo `deploy-soloagency.sh` sẵn có.
- **Mạnh nhất (tùy chọn về sau):** để chống cả agent cố tình gian, cho tác vụ ghi/side-effect từ chối chạy nếu thiếu LOAD LEDGER — nhưng đó là việc runtime làm một lần, không phải bảo trì theo từng file.

---

## Tóm tắt 1 dòng
Nâng mọi ràng buộc từ **"đã load?"** lên **"đã load ĐỦ DÒNG?"** bằng: LOAD LEDGER in mỗi lần load (dòng-cuối + số-dòng), dependency-complete, đối chiếu `LOAD_MANIFEST.md` tự sinh trong deploy, và preflight chặn trước mọi hành động — agent hết đường hành động trên file đọc thiếu, mà bạn không phải bảo trì tay.
