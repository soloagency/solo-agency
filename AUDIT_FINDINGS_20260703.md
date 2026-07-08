# Solo Agency — Audit toàn bộ skill/playbook (2026-07-03)

Phạm vi: master `SOLO_AGENCY_PLAYBOOK.md`, `AGENTS.md`, toàn bộ `playbooks/` (00–11, entrypoints, gate, adapter, LOAD protocol), `playbooks/skills/*`, `tools/solo_report_renderer.py`, đối chiếu `solo-agency-collector/` (bridge Go + extension + runbook). Renderer đã được chạy thử thực tế (render/package/scrub/PDF).

Ký hiệu: 🔴 HIGH — agent chắc chắn/thường xuyên làm sai; 🟠 MED — dễ miss ở đúng thời điểm hành động; 🟡 LOW — nhỏ/dọn dẹp.

---

## 1. Giao thức load skill (LOAD LEDGER / LOAD_MANIFEST)

- 🔴 **L1. Read tool đếm dư 1 dòng ảo → ledger check kẹt.** File master đúng 831 dòng (kết thúc `\n`, khớp manifest), nhưng Read tool của Claude Code báo "832 total" (dòng trống ảo cuối file). `LOAD_LEDGER_PROTOCOL.md:25-27` chỉ định nghĩa PASS khi `lines_read==manifest_lines` và FAIL khi `<`; trường hợp `>` không được định nghĩa → agent hoặc tự chặn nhầm hoặc tự bịa cách diễn giải. **Sửa:** định nghĩa cách đếm chuẩn (`wc -l`/`awk END{NR}`) và quy tắc "lines_read ≥ manifest + last line khớp = PASS (ghi chú artifact của reader)".
- 🔴 **L2. Line-count + last-line vẫn pass trên file hỏng — bằng chứng thực tế:** `05_MEASURE_LEARN_IMPROVE.md:281` kết thúc bằng heading `### Final Hard Gate` **rỗng hoàn toàn** (không có nội dung gate). Manifest chứng nhận 281 dòng, last line đúng → mọi ledger PASS, không ai phát hiện gate biến mất. **Sửa:** viết nội dung gate (hoặc xoá heading) + cân nhắc quy tắc "file không được kết thúc bằng heading rỗng".
- 🟠 **L3. Prompt automation per-client (kiến trúc mặc định) KHÔNG có kỷ luật ledger.** `SCHEDULED_RUN_ENTRYPOINT.md:55-72` (client-specific prompt, được tuyên bố là preferred ở dòng 47) không nhắc LOAD LEDGER/LOAD_MANIFEST/full-load; prompt generic có (dòng 15, bước 3F) nhưng không được cài mặc định. Task prompt là văn bản duy nhất agent scheduled chắc chắn thấy. **Sửa:** copy bước 2 + 3F vào template per-client.
- 🟡 **L4.** `LOAD_MANIFEST.md` không tự liệt kê chính nó, mâu thuẫn "lists every playbooks/**/*.md" (`LOAD_LEDGER_PROTOCOL.md:60`); thứ tự cột thực tế `path|lines|sha256|last_line` khác mô tả `path|lines|last_line|sha256`.
- 🟡 **L5.** `SCHEDULED_RUN_ENTRYPOINT.md:113-124` (Required Runtime Loads) thiếu chính `LOAD_LEDGER_PROTOCOL.md`/`LOAD_MANIFEST.md`. `04:64-77` (Loading Contract) thiếu Stage 3A/3B và không nhắc ledger — hai danh sách load "chính thức" lệch nhau và lệch Stage Map.
- 🟠 **L6. Skill con có LOAD_MANIFEST nhưng không gì kích hoạt việc kiểm.** 3 skill viết (blog/social/video-script) và `03`/adapter không hề nhắc kiểm line-count theo LOAD_MANIFEST của skill (grep 0 hit) → `method.md` bị cắt cụt là không phát hiện được đúng nơi master yêu cầu canary. **Sửa:** thêm bước verify manifest vào Final WideCast Script Skill Gate (`03:718-719`) + evidence list của adapter.
- 🟠 **L7. Stage 1 không được load ở first run.** Master Stage Map: Stage 1 load cho "Automation Flow first agency run/report", nhưng `SCHEDULED_RUN_ENTRYPOINT.md:128` và `04:69` nói chỉ load khi "setup repair" → first run (profile vừa setup xong, luôn "complete") bỏ qua đúng file chứa contract báo cáo đầu tiên.

## 2. Setup Flow

- 🔴 **S1. Exemplar trong 01 hỏi private data source SAI BƯỚC.** `01:137-153` ví dụ mẫu: sau inference (bước 2-3) hỏi ngay "Do you want to provide competitor pages, Facebook groups… private data sources" — vi phạm checkpoint bước-6-duy-nhất của master và chính `01:16,77`. Agent bắt chước exemplar hơn là luật trừu tượng → đây là con đường phá trật tự setup dễ nhất.
- 🟠 **S2.** Intake list First Client / Add Client (`01:284-285`, `01:304-305`) vẫn xin "Optional private data sources" ngay từ đầu; `00:485-486` và `00:1084-1088` cũng bảo hỏi private sources vô điều kiện trong phần "setup steps". **Sửa:** thay bằng "ghi nhận nguồn tự nguyện thành `pending_private_review`, xử lý tại bước 6".
- 🟠 **S3.** `Automation freshness check` — bắt buộc trong mọi progress block sau khi có automation — **không xuất hiện** trong 00/01 (0 hit), kể cả exemplar "good first automation report" (`01:426-453`). `00:532` còn dùng nhãn sai `Action needed` thay vì `**[ACTION REQUIRED]**`; 00/01 không có luật "tối đa 3 block" và `No action required right now.`.
- 🟡 **S4.** Tham chiếu ma "16-item/16-step setup list": `master:612`, `00:264`, `TODO.md:161` — roadmap chuẩn hiện là 9 mục; không tồn tại danh sách 16 mục nào. `TODO.md:201` còn yêu cầu ngược thứ tự hiện hành (private trước routine).
- 🟡 **S5.** `00:191-202` danh sách "Required meanings" thiếu 7 thuật ngữ master bắt buộc (idea matrix, hot/warm lead, competitor, HTML report, draft, analytics/statistics, schedule/routine) — bước 9 roadmap dùng "idea matrix" mà không có định nghĩa trong file đang load.

## 3. Tạo report (mảng nặng nhất)

- 🔴 **R1. MÂU THUẪN CHÍNH DIỆN về link handoff chuẩn.** Master `:323,329` bắt link chính là `{client-name}-daily-report.html`; nhưng `06:196,455`, `07:1029-1030`, `09:47,1420`, `SCHEDULED_RUN_ENTRYPOINT:36,166` bắt link chính là `{client-name}-client-report.html` và **cấm** đưa daily/lane link. `09:58` tự mâu thuẫn với `09:47` cách nhau 11 dòng. Master là bên cũ, nhưng master lại tuyên bố override → agent chọn bên nào cũng vi phạm một "must"; output hướng người dùng đổi qua đổi lại giữa các run. **Sửa 1 câu ở master §319-331** (chốt client-report.html là handoff, 3 file HTML là staging) + reword 09:58.
- 🔴 **R2. SKILL.md report-design tắt scrub gate một cách im lặng.** Ví dụ lệnh ở `skills/report-design/SKILL.md:86,93` **thiếu `--client-facing --fail-on-scrub`**; renderer không quét scrub nếu thiếu flag (đã verify thực tế). SKILL.md là file được load đúng lúc hành động (6A) và là nguồn copy-paste → agent giao HTML/PDF cho khách mà không hề quét client-blind. **Sửa:** thêm flag vào SKILL.md, hoặc đổi renderer thành scrub mặc định bật (`--internal` để tắt).
- 🔴 **R3. `package` tạo nút copy CHẾT trong file khách nhận.** `extract_body_fragment` strip mọi `<script>` nhưng giữ `<button data-copy-target>` (verify: 2 button, 0 script) — trong khi `06:1076` bắt buộc "real local copy buttons" và `06:23` cấm fake buttons. File duy nhất khách mở chứa toàn nút không bấm được.
- 🔴 **R4. Deadlock editable draft blocks.** `06:1328` bắt renderer tự sinh block editable từ heading `## Version N:` và cấm AI tự viết HTML; renderer **không có** tính năng này (pipeline markdown chỉ xuất section thường). Kèm `06:20,182-183` cấm script one-off → mọi run hoặc vi phạm 1328 hoặc giao draft thiếu UI bắt buộc. **Sửa:** implement trong renderer hoặc sửa 06 nói rõ đây là tính năng pending + fallback hợp lệ.
- 🟠 **R5.** PDF fallback `reportlab` không được tài liệu hoá (`06:175` chỉ ghi chrome/weasyprint/wkhtmltopdf): mất toàn bộ URL/bảng/CSS nhưng vẫn báo `pdf_status: generated` sạch sẽ → INTERNAL_REPORT ghi nhận PDF tốt trong khi thực tế degraded. **Sửa:** status `generated_degraded`.
- 🟠 **R6.** Scrub FAIL nhưng file HTML nhiễm vẫn nằm trên đĩa với đúng tên thật (ghi file trước, check sau — verify exit 3 vẫn còn file) → bước sau chỉ check tồn tại file là ship nhầm. **Sửa:** ghi `*.blocked.html` hoặc xoá khi fail.
- 🟠 **R7.** Danh sách bước thao tác `06:134-141` (thứ tự bắt buộc mỗi report) **không có**: tạo INTERNAL_REPORT, copy `outputs/latest/`, update `report_state.json` khi thành công, reconciliation counts. 4 danh sách section report khác nhau tồn tại song song (06 hierarchy 14 mục vs SKILL 9 mục — thiếu Decision Scorecard, Compliance…), đánh số hỏng (`06:1228→1233` 12 rồi 11).
- 🟠 **R8.** `01 §15` là bản sao đã drift của thuật toán 04 (thiếu report_state.json, renderer, skill 6A, Stage 6 load — grep 0 hit `solo_report_renderer` trong 00/01); `00:388-405` private-scan completion còn theo mô hình 1 báo cáo cũ, có nguy cơ clobber lane public; `INTERNAL_REPORT` = 0 hit trong toàn bộ 1805 dòng của 00.
- 🟡 **R9.** Enum `notification_status: not_sent|sent|skipped` (`06:437`) không có giá trị failed/blocked trong khi log delivery dùng `sent|failed|unavailable|skipped`; scrub list của renderer hẹp hơn master ("config files", "debug" generic lọt lưới); 2 path root khác nhau cho cùng artifact trong 06 (B2).

## 4. Content, idea matrix, production

- 🔴 **C1. Idea matrix & best idea KHÔNG có gate value-first.** `00 §E (1439-1487)` và `§F (1619-1656)` — spec chuẩn của idea matrix/best idea — 0 hit `promotional_not_value_first`; 12 tiêu chí chọn best idea còn có "Business relevance"/"Lead potential" mà không có tiêu chí giá trị khán giả → chủ động đẩy về hướng quảng cáo. Gate chỉ nằm ở 01 (file mà scheduled run được bảo đừng load — xem L7). 3 skill viết (blog/social/video-script checklist) và Hard Gates của 03 cũng không có. **Sửa:** thêm 1 bullet gate vào 00 §E + §F, Hard Gates 03, checklist 3 skill.
- 🔴 **C2. Đường no-research của skill vendored tự gọi `widecast_create_video(source="idea")` không cần approval, không qua script gate.** `video-script-writing/method.md:78` + `handoff.md:39`. Scheduled run trên runtime không có web tool sẽ theo đúng chữ và **tiêu credit không giám sát**. Adapter không map path này. 
- 🔴 **C3. `widecast_create_image` tốn credit chỉ cần "tell the user", không cần approval** (`method.md:141`, `SKILL.md:181` "disclosed") — ngược master (credit spend luôn cần approval + ACTION REQUIRED) và chính adapter dòng 97.
- 🔴 **C4. Blog skill repurpose-to-video bỏ qua mọi gate:** `blog-writing/SKILL.md:135-142` cho phép `output_type="video"` render thẳng MP4 (bỏ scene review + editing pass + fresh render approval) và `source="blog"` gửi thẳng bài viết không qua video-script-writing skill. Ngược cả `handoff.md:42` của skill anh em. Adapter chỉ cấm "report script pasted through", không phủ `source="blog"|"audio_url"|"idea"`.
- 🔴 **C5. Run scheduled không có nguồn cho `script_approved`/`production_mode`.** `handoff.md:33-34` bắt "Ask each time" (giả định chat live); master cấm gate xác nhận thứ hai khi đã có approval — nhưng không đâu định nghĩa saved approval record phải chứa `production_mode` → run tự treo hoặc tự bịa `script_approved=true`.
- 🟠 **C6.** "Implicit approval" (`method.md:170`) trái chữ với `03:952` "never assume approval"; câu hỏi production ở dạng văn xuôi, không dùng block `**[ACTION REQUIRED]**` như 03:23 bắt buộc.
- 🟠 **C7.** Hard gate "5 phiên bản" (`03:12,271`) vs luật fit-based của skill (`SKILL.md:43,95` — bỏ format khi phải bịa) → agent tuân 03 sẽ bịa "myth" để đủ 5, vi phạm luật trung thực của chính skill. **Sửa:** 03 defer về fit rules của skill.
- 🟠 **C8.** Stage-2 vetting ảnh là chat-only (SHOWN-LOCAL "display to the user", "Want to tweak anything?") — không có tương đương cho run không giám sát. **Sửa:** quy định scheduled run thoả SHOWN-LOCAL bằng evidence file + nhúng vào INTERNAL_REPORT.
- 🟡 **C9.** Client-blind không được nhắc nơi viết caption/blog (skill social gắn caption với "a WideCast video"); `handoff.md:42` "user renders MP4 from UI" vs adapter bảo agent gọi export sau fresh approval; `03:719` thiếu module `hooks`/`ctas` trong danh sách gate.

## 5. Scheduled run / đo lường / leads

- 🔴 **A1. Client-specific prompt (mặc định triển khai) thiếu gần hết gates:** không Stage 9 trước completion, không Stage 5/analytics, không progress block + freshness check, không ACTION REQUIRED contract, không value-first, không Stage 10 load tường minh, Collector Runtime Verification chỉ một nửa, không notification content contract. **Sửa 1 dòng delegation:** "Then follow every numbered rule of the Scheduler Prompt above and the Loading Contract in 04, restricted to this client."
- 🟠 **A2.** Thuật toán Daily Run trong 04 (`:250-360`) **không có bước đo lường/Stage 5 và không có bước report_state.json** — chỉ nằm ở prose; thuật toán trong 09 (`:650-658`) cũng thiếu INTERNAL_REPORT, report_state, scrub gate, latest copies, `lead_competitor_opportunities.jsonl` — đúng những thứ completion gate của chính nó đòi; đánh số hỏng (9→11, hai số 9).
- 🟠 **A3.** Generic prompt xử lý mọi client trong khi override `target_client_slug` nằm NGOÀI code block (`ENTRYPOINT:74`); `04:250` "For each active client" vô điều kiện vs `04:34-36` cấm task client-specific đụng client khác.
- 🟠 **A4. Không có run-lock/concurrency/dedup notification nào trong toàn corpus:** run hôm qua chưa xong chồng run hôm nay; master-task và client-task cùng build lại một report; retry gửi lại notification (log được ghi nhưng không bắt đọc trước khi gửi). Timezone `"local"` không định nghĩa thuộc đồng hồ nào (máy người dùng vs sandbox UTC) → "yesterday" và folder `YYYY-MM-DD` có thể lệch ngày.
- 🟡 **A5.** `05:281` Final Hard Gate rỗng (xem L2); `05:197` path collector thiếu `{client_slug}`; token `measurement_status: no_published_urls_yet` nằm trong 05 nhưng 05 chỉ được load khi… đã có publish; 05 dùng `analytics/learning_log.md`+`comment_signal_log.md` nhưng cây thư mục chuẩn của 07 chỉ có `metrics_log.md`; glob health-check `inbox/YYYY-MM/*/collector_status.json` thiếu 1 cấp thư mục (đúng là `YYYY-MM/{client_slug}/{run_id}/`) — lặp ở `04:283,295,570` + `ENTRYPOINT:25`.

## 6. Private data source / Local Collector

- 🔴 **P1. Các nhánh "từ chối/hoãn" bỏ qua Automation Resync.** `02:196-200, 269-273, 274-279` — checklist tại điểm quyết định không có bước resync (các nhánh "yes" thì có) → task prompt giữ snapshot cũ, ngược hard gate `02:25` và master.
- 🔴 **P2. Không có protocol hậu-thu-thập trong 08; Completion Gate của GATE yếu hơn master.** Sau khi collector chạy xong, ngữ cảnh agent (08+GATE) không nhắc: load Stage 10, cập nhật idea matrix/best idea/drafts, cập nhật lane private + daily **không đụng lane public**, reconciliation bộ report. GATE:106-114 chỉ đòi "report/idea matrix/drafts updated".
- 🟠 **P3. Kỹ thuật drift:** `08:1235-1251` layout output/jobs cũ (`inbox/YYYY-MM/YYYY-MM-DD_client_slug/`, `jobs/YYYY-MM/`) ngược chính `08:34` và code (`{client_slug}/{run_id}`, `jobs/pending|claimed|completed`); RUNBOOK bảo agent có thể tự start bridge (`:48,372` — ngược 08:18), nói bridge chạy tuần tự (`:274` — code chạy song song per client), poll sai path (`:328`).
- 🟠 **P4.** `POST /shutdown` trong template setup gọi không token (`08:482,662,742`) nhưng code bọc `requireToken` (token per-run chỉ extension có) → bước 1 của mọi restart luôn 401, không có đường graceful-shutdown thực tế.
- 🟠 **P5.** Phân loại private vs public mâu thuẫn ở rìa: `02:99-106` xếp fanpage/TikTok/LinkedIn competitor vào private (đa số xem được logged-out) trong khi `GATE:46` cho phép browser agent với "public pages", "proven public" không định nghĩa → 2 agent tuân thủ hành xử ngược nhau; có lối thoát dùng browser cấm. **Sửa:** tie-breaker "mọi trang social của client/competitor đang theo dõi = collector-only".
- 🟠 **P6.** Luật master "đừng skip private sources chỉ vì config nói public_only/postponed — verify runtime trước" **vắng mặt trong cả GATE/02/08**; GATE:3 chỉ trigger khi "human asks" — scheduled run (đường vào phổ biến nhất) không kích hoạt gate theo đúng chữ.
- 🟡 **P7.** `manifest.json` đòi `short_name` nhưng helper script không patch trường đó; danh sách file extension canonical thiếu `filtering.js/readability.js/infinity_loops.js`; từ vựng blocker status trôi tự do (`collector_unavailable` vs `collector_offline_or_unreachable`…) — cần 1 bảng enum; ví dụ payload tối thiểu thiếu `allowed_extension_instance_ids` → nguy cơ extension client khác claim job. Chưa có luật retention/PII redaction cho raw HTML private trong `inbox/`.

## 7. Storage / Stage 9 / Update

- 🔴 **T1. Cây thư mục §7 của 07 hỏng cấu trúc:** `07:161-210` indent sai khiến mọi thứ thành sibling của `daily-content-pipeline/`; block automation bị lặp đôi (`:169-174` vs `:175-180`); `07:212-213` profile đặt sai cấp; subtree collector là layout legacy chết (khác Latest Override `07:37-92` và thực tế). Stage 7 là "storage authority" → agent theo cây §7 sẽ tạo file sai chỗ.
- 🔴 **T2. `extension_registry.json` không có schema ở bất kỳ đâu** dù master/11/entrypoints đều bắt đọc/ghi nó (scheduled run phải đọc mapping client→extension_instance_id từ đây). `agent_registry.json` xuất hiện đúng 1 lần (dangling). `update_watch_task_prompt_pending` không có field chứa.
- 🔴 **T3. Stage 9 có thể rubber-stamp về cấu trúc:** ~200 checkbox tự khai, 14 checklist, "trước MỖI reply" (~40 mục), **không yêu cầu evidence nào** (scrub gate là yes/no tự khai, không grep; không diff count). **Sửa:** scope checklist theo flow + bắt evidence cơ học cho 5 gate kiểm được (grep scrub, `ls` bộ 8 file, field report_state, ledger counts, dòng notification_log) + format in `gate | evidence | pass/miss`.
- 🟠 **T4.** `09:81-95` Minimum resync audit thiếu bước 9 của master (update_state/update_log); dry-read mơ hồ hơn danh sách 8 file của master; `09 §26` không đòi "Stage 10 was loaded".
- 🟡 **T5.** ~230 dòng §23.x trùng nguyên văn giữa 07:1652-1874 và 09:940-1291 (cơ chế sinh drift kiểu R1); `report_merge_contract` (`07:496`) không có giá trị nào khớp kiến trúc hiện tại; `TODO.md` có 3 mục đã implement phần lớn (Offer Map, Asset Bank, Lead Handling) — nên chú thích "đã có/còn thiếu" để agent không xây trùng; `test_logs/` không có trong cây 07.

---

## TOP 10 việc nên sửa trước (xếp theo đòn bẩy)

1. **Chốt link handoff chuẩn** (R1): sửa master §319-331 → `client-report.html`, 3 file HTML = staging; reword `09:58`. Một câu, hết mâu thuẫn user-visible lớn nhất.
2. **Thêm 1 dòng delegation vào client-specific prompt** (`SCHEDULED_RUN_ENTRYPOINT.md:55-72`): tuân generic 28-step + Loading Contract 04 + LOAD LEDGER (L3 + A1). Sửa 1 chỗ vá được cả chục miss.
3. **Thêm value-first gate vào 00 §E/§F + Hard Gates 03 + checklist 3 skill viết** (C1).
4. **Vá adapter với 4 override cho skill vendored**: `source="idea"` (C2), `create_image` credit (C3), `output_type`/`source="blog|audio_url"` (C4), "render từ UI"; + định nghĩa saved approval record chứa `production_mode` (C5).
5. **Renderer + SKILL.md**: scrub mặc định bật hoặc thêm flags vào SKILL examples; ghi `*.blocked.html` khi fail; nút copy trong package; status `generated_degraded` cho reportlab (R2-R6).
6. **Sửa protocol ledger**: định nghĩa cách đếm dòng + case `lines_read > manifest`; viết nội dung Final Hard Gate rỗng của 05 (L1, L2).
7. **Thêm bước vào các thuật toán đánh số** (04, 09, 01 §15, 08 hậu-thu-thập): INTERNAL_REPORT, report_state.json, scrub gate, latest copies, Stage 5, Stage 10, reconciliation (A2, R7, R8, P2).
8. **Thêm Automation Resync vào các nhánh declined/postponed của 02** (P1) + luật "verify collector runtime trước khi tin public_only" vào GATE/08 (P6).
9. **Viết lại cây thư mục §7 của 07 + schema `extension_registry.json`** (T1, T2); sửa layout cũ `08:1235-1251` và 3 chỗ RUNBOOK (P3).
10. **Stage 9 evidence-based** (T3) + thêm Stage 10/update_state vào checklist 09 (T4); dọn "16-item list", TODO:201, đánh số section hỏng.

## Điểm đã verify SẠCH (không cần sửa)

- Manifest line counts khớp 100% thực tế (52 file), mọi file kết thúc `\n`, không có tham chiếu playbook nào đứt.
- `api_key_env`/`api_key_local` nhất quán khắp nơi; server WideCast `widecast.ai/app/dashboard` + cấm `api.widecast.ai` nhất quán; các bước setup API key nhất quán 3 chỗ trong 03.
- Luật no-regeneration 5 phiên bản, No-Local-DIY-Video, editing pass + fresh render approval: master/03/adapter nhất quán.
- Endpoint list, `/status` fields, header extension, job queue lifecycle trong 08 khớp code bridge Go từng field.
- 04 enforce đúng hard stop "Setup không chạy report"; naming task/extension khớp master; update-watch không gửi notification khớp.
- Renderer: `render`/`package` + flags như tài liệu, scrub term list khớp `06:316-334` từng chữ, rewrite sibling links → anchors hoạt động, chuỗi PDF chrome hoạt động.
