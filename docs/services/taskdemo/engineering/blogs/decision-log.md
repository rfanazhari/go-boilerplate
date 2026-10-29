# Decision Log: Blogs

Technical decisions in this feature that exist because of business decisions or constraints. Each entry records why, what it means for the product, and when it should be revisited.

Sources: [blogs capability](../../business/capabilities/blogs.md) (approved OQ-01–14, 2026-10-29) and [blogs delta report](../../delta-report/delta-report-blogs.md).

## D1: Soft delete keeps the writing, and the public address stays reserved

- **Business driver:** Authors and admins must be able to withdraw a writing while keeping the record (`BR-TASKDEMO-066`, `BR-TASKDEMO-073`, `BR-TASKDEMO-074`). Deletion is not a third publication status (`BR-TASKDEMO-061`, OQ-09). A deleted public address must not be given to a later writing (OQ-07). Restore, hard delete, and a retention period are out of the initial scope (OQ-09).
- **Technical decision:** Each blog row keeps its publication status (`draft` or `published`) and a separate `deleted_at` timestamp. Setting `deleted_at` is the soft delete. The row stays in the database. The public address (slug) stays unique across every row, including deleted ones, so it is never issued again. Admin delete of an already deleted blog succeeds and changes nothing. A non-admin, including the author, receives “not found” on any later attempt to see or change that blog. An admin can still open it on the admin/management read, and cannot edit, publish, or unpublish it.
- **What this means for the product:** Readers and the author lose the writing as soon as it is deleted. An admin can still open the record to see what was withdrawn. Publishing it again, editing it, or permanently erasing it is not available in this release. A new blog may reuse the same title, but it receives a new public address; the old address does not start working again.
- **Alternatives considered:**
  - **A third status such as `deleted`:** One column would be simpler, and it would contradict the approved rule that publication status is only draft or published. A deleted blog can still be published underneath; hiding it is a separate fact.
  - **Hard delete:** The row would disappear, so an admin could not inspect a withdrawn writing, which the capability requires.
  - **Free the public address when the blog is deleted:** New writings could inherit a well-known URL. The approved rule is that the address stays reserved.
- **Trade-offs accepted:** Deleted rows accumulate. There is no automatic purge. Admin tools must treat `deleted_at` as the switch for “withdrawn”, and must not assume a published status means the public can see the writing.
- **Revisit when:** Product asks for restore, permanent erasure, or a retention window.
- **Status:** active

## D2: Public reading and signed-in management use different addresses

- **Business driver:** Anyone can read the public list and a public writing without logging in (`BR-TASKDEMO-070`). Changing, publishing, or moderating a writing requires a signed-in active account (`BR-TASKDEMO-053`, `BR-TASKDEMO-054`, OQ-02). A public page must not reveal a draft just because the caller is the author or an admin (`BR-TASKDEMO-071`). Only a non-admin user may create a blog; an admin who tries receives forbidden (`BR-TASKDEMO-050`, OQ-01).
- **Technical decision:** Public reading is `GET /blogs` and `GET /blogs/{slug}`. Those routes do not require a token. If a token is sent, it is ignored, and the public visibility rules still apply. Signed-in actions use `/blog-records` and always require the existing Bearer check for an active, non-deleted account. Creating a blog is limited to the user role. The admin list is limited to the admin role. Edit, publish, unpublish, delete, and private read sit on `/blog-records` for any signed-in role; the use case decides whether that person is the author, an admin, or neither. The same split already used for health and login (public routes registered outside the authenticated router) is extended, rather than optional authentication on one URL.
- **What this means for the product:** A reader never has to log in to open a public writing. Logging in does not unlock drafts through the public address; the author or an admin uses the management address and the blog’s id. An admin who tries to create a blog is forbidden. A user who opens the admin list is forbidden. A caller who can see a writing but is not allowed to change it is forbidden. A writing they are not allowed to know about is not found.
- **Alternatives considered:**
  - **One `/blogs/{id}` URL whose behavior changes when a token is present:** The public address is a slug, not the id (D3). A single path cannot mean both “anonymous slug” and “signed-in id” without the public route capturing the management route. It would also make a public request with a leftover token behave differently from a request with no token, which breaks the rule that the public channel stays public.
  - **Role checks only inside the use case, with every blog URL behind login:** Anonymous reading would be impossible.
  - **Role checks only in routing, including author-or-admin:** The router can express “user” or “admin”. It cannot express “author of this blog, or an admin” without loading the blog in the router.
- **Trade-offs accepted:** Clients learn two prefixes. A new management URL must be registered under `/blog-records` and behind the existing login check, or it will be public. Author and admin permissions still have to be tested in the use case, because the router cannot see them.
- **Revisit when:** The service grows a second way to call these actions (for example a worker) that has no HTTP router, or product wants the public and management addresses to share one prefix.
- **Status:** active

## D3: Public pages use a fixed slug; management uses the blog id

- **Business driver:** A public writing is opened by a title-based address, not by the internal id (`BR-TASKDEMO-084`, OQ-06, OQ-07). Two writings may share a title. The address is chosen when the writing is created, stays the same when the title changes, and is not reused after soft delete. Management, private read, and admin read use the id.
- **Technical decision:** The server builds the slug at create. The title is trimmed, unicode-normalized so accented letters fold to ASCII letters, lowercased, and turned into hyphenated words. A title that produces no words uses the base `blog`. The base is cut at 80 characters. An 8-character random hexadecimal suffix is added (`catatan-go-a1b2c3d4`). The full slug is unique. On a rare collision the server tries a new suffix, up to five times. Rename does not change the slug. Public detail looks up that slug only. `PATCH`, publish, unpublish, delete, and private read look up the UUID on `/blog-records/{id}`.
- **What this means for the product:** Readers see a stable, readable address. Changing the title does not break an existing link. Two posts titled “Catatan Go” get two different addresses. After deletion, the old address stays dead. Authors and admins manage the writing with its id, which is returned when the writing is created.
- **Alternatives considered:**
  - **Public detail by UUID:** Stable and unique, and it fails the requirement that the public address is based on the title.
  - **Slug equal to the title, updated on every rename:** Links would break when the author edits the title, and identical titles could not all be public.
  - **Reuse the slug after soft delete:** A new writing could take over an address readers still have. The capability forbids that.
- **Trade-offs accepted:** The slug does not follow a later title change, so the address can lag the headline. Non-Latin titles that do not yield ASCII words become `blog-` plus the suffix, so those addresses are less descriptive. The suffix is random, not a sequence number.
- **Revisit when:** Product wants the public address to track renames, wants non-Latin characters kept in the path, or wants a shorter or longer suffix.
- **Status:** active

## D4: Public and admin lists use page numbers

- **Business driver:** The public list is paged with `page` and `pageSize`, default size 20, maximum 100, newest first, with the id breaking ties (OQ-13). The admin list is also required (OQ-09). Search, other public filters, and a private list for the author are deferred (OQ-13).
- **Technical decision:** Both lists use the same paging rules. `page` starts at 1. Omitted `page` is 1 and omitted `pageSize` is 20. `pageSize` above 100, below 1, or not an integer is rejected, as is `page` below 1 or not an integer. A page past the end returns an empty list and the real total, not “not found”. Order is creation time newest first; when those times match, the higher id comes first. Each public item has id, slug, title, author name, and creation time. Full content is only on a detail response. The admin list uses the same paging and adds status, owner id, and deletion time, plus a `deleted` filter: `exclude` (default), `only`, or `all`.
- **What this means for the product:** A reader can move through the public writings in stable pages of up to 100. The list stays light: opening a page does not download every article body. Moderators get the same paging on their list, and can choose to see current writings, withdrawn writings, or both. There is no search box in this release, and an author does not get a “my drafts” list; they open a draft by its id.
- **Alternatives considered:**
  - **Cursor paging:** More stable when new posts are inserted at the top while someone is paging. The approved contract names `page` and `pageSize`, so cursors would be a different contract.
  - **Return every public blog in one response:** Contradicts the maximum page size.
  - **Leave the admin list unpaged:** A moderator could receive an unbounded payload. The same page rules keep that list operable without inventing a second contract.
- **Trade-offs accepted:** Inserting a new blog at the top can shift later pages. Clients need `total` to know when to stop; that count is part of this contract even though the capability only named the item fields. Very deep pages are allowed and can become slow; there is no cursor to avoid that.
- **Revisit when:** Product adds search or filters, asks for an author draft list, or the public catalogue is large enough that offset paging is too slow.
- **Status:** active

## D5: A deactivated author disappears from the public pages without unpublishing the blog

- **Business driver:** A published blog whose author is non-active or soft-deleted must leave the public list and the public detail (`BR-TASKDEMO-075`, OQ-08). The blog’s own publication status does not change. When that author is active and not soft-deleted again, the same published blog is public again without a new publish. Deactivating an account already revokes tokens; that is not enough, because public readers are anonymous.
- **Technical decision:** Public list and public detail both check, at the moment of reading, that the blog is published, the blog is not soft-deleted, and the author’s account is active and not soft-deleted. No job copies the account status onto the blog when an admin deactivates or deletes the account. The name shown on the public page is the author’s current account name.
- **What this means for the product:** Turning an account off takes that person’s published writings off the public site immediately, and turning the account back on brings those same writings back. Drafts are unaffected because they were never public. An admin can still open the writings. A rename of the author shows up on the public pages without editing each blog.
- **Alternatives considered:**
  - **Unpublish every blog when the account is deactivated:** The writings would stay hidden after reactivation until each one was published again, which contradicts OQ-08.
  - **Store “author was active” on the blog at publish time:** A later deactivation would leave the writing public until something rewrote that copy.
  - **Rely on token revocation only:** Anonymous readers do not send a token, so they would keep seeing the writing.
- **Trade-offs accepted:** Every public read checks the author’s account. A blog does not keep the name it had when it was published; a later rename changes the byline.
- **Revisit when:** Product wants a frozen byline, or wants deactivation to unpublish writings permanently.
- **Status:** active
